package security

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jenvenson/ops-platform/internal/alert"
	"github.com/jenvenson/ops-platform/internal/cmdb"
	"github.com/jenvenson/ops-platform/internal/database"
	"github.com/jenvenson/ops-platform/internal/models"
	"github.com/jenvenson/ops-platform/internal/secureconfig"
	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

type FIMPolicyFilter struct {
	Keyword  string
	Enabled  *bool
	Page     int
	PageSize int
}

type FIMAlertFilter struct {
	Status   string
	Severity string
	Page     int
	PageSize int
}

type FIMSnapshotFilter struct {
	PolicyID     uint
	ServerID     uint
	SnapshotType string
	Status       string
	Page         int
	PageSize     int
}

type FIMService struct{}

var fimExecutionLocks sync.Map

func NewFIMService() *FIMService {
	return &FIMService{}
}

func (s *FIMService) ListPolicies(filter FIMPolicyFilter) ([]FIMPolicy, int64, error) {
	query := database.DB.Model(&FIMPolicy{})
	if filter.Keyword != "" {
		like := "%" + filter.Keyword + "%"
		query = query.Where("name LIKE ? OR description LIKE ?", like, like)
	}
	if filter.Enabled != nil {
		query = query.Where("enabled = ?", *filter.Enabled)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page := maxInt(filter.Page, 1)
	pageSize := clampInt(filter.PageSize, 10, 100)
	var items []FIMPolicy
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *FIMService) CreatePolicy(req CreateFIMPolicyRequest, operator string) (*FIMPolicy, error) {
	item := &FIMPolicy{
		Name:            req.Name,
		Description:     req.Description,
		Enabled:         req.Enabled,
		Severity:        defaultString(req.Severity, "high"),
		NotifyChannels:  normalizeNotifyChannels(req.NotifyChannels),
		ScanIntervalSec: defaultInt(req.ScanIntervalSec, 300),
		HashMode:        defaultString(req.HashMode, "changed_only"),
		CompareMode:     defaultString(req.CompareMode, "baseline"),
		CreatedBy:       operator,
		UpdatedBy:       operator,
	}
	if err := database.DB.Create(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}

func (s *FIMService) UpdatePolicy(id uint, req UpdateFIMPolicyRequest, operator string) (*FIMPolicy, error) {
	var item FIMPolicy
	if err := database.DB.First(&item, id).Error; err != nil {
		return nil, err
	}
	if req.Name != "" {
		item.Name = req.Name
	}
	if req.Description != "" {
		item.Description = req.Description
	}
	if req.Enabled != nil {
		item.Enabled = *req.Enabled
	}
	if req.Severity != "" {
		item.Severity = req.Severity
	}
	item.NotifyChannels = normalizeNotifyChannels(req.NotifyChannels)
	if req.ScanIntervalSec > 0 {
		item.ScanIntervalSec = req.ScanIntervalSec
	}
	if req.HashMode != "" {
		item.HashMode = req.HashMode
	}
	if req.CompareMode != "" {
		item.CompareMode = req.CompareMode
	}
	item.UpdatedBy = operator
	if err := database.DB.Save(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *FIMService) DeletePolicy(id uint) error {
	return database.DB.Delete(&FIMPolicy{}, id).Error
}

func (s *FIMService) ClearPolicyHistory(policyID uint) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("policy_id = ?", policyID).Delete(&FIMAlert{}).Error; err != nil {
			return err
		}
		if err := tx.Where("policy_id = ?", policyID).Delete(&FIMDiffEvent{}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (s *FIMService) SetPolicyEnabled(id uint, enabled bool, operator string) error {
	return database.DB.Model(&FIMPolicy{}).Where("id = ?", id).Updates(map[string]any{
		"enabled":    enabled,
		"updated_by": operator,
		"updated_at": time.Now(),
	}).Error
}

func (s *FIMService) ListTargets(policyID uint) ([]FIMPolicyTarget, error) {
	var items []FIMPolicyTarget
	err := database.DB.Where("policy_id = ?", policyID).Order("id DESC").Find(&items).Error
	if err != nil {
		return nil, err
	}
	if err := s.enrichTargets(items); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *FIMService) AddTargets(policyID uint, serverIDs []uint) error {
	for _, serverID := range serverIDs {
		item := FIMPolicyTarget{PolicyID: policyID, ServerID: serverID, Enabled: true}
		if err := database.DB.Where("policy_id = ? AND server_id = ?", policyID, serverID).FirstOrCreate(&item).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *FIMService) DeleteTarget(policyID uint, targetID uint) error {
	return database.DB.Where("policy_id = ? AND id = ?", policyID, targetID).Delete(&FIMPolicyTarget{}).Error
}

func (s *FIMService) ListWatchPaths(policyID uint) ([]FIMWatchPath, error) {
	var items []FIMWatchPath
	err := database.DB.Where("policy_id = ?", policyID).Order("id DESC").Find(&items).Error
	return items, err
}

func (s *FIMService) CreateWatchPath(policyID uint, req CreateFIMWatchPathRequest) (*FIMWatchPath, error) {
	item := &FIMWatchPath{
		PolicyID:        policyID,
		Path:            req.Path,
		ScanMode:        normalizeFIMWatchPathScanMode(req.ScanMode),
		Recursive:       req.Recursive,
		MaxDepth:        req.MaxDepth,
		FileGlob:        req.FileGlob,
		ExcludeGlob:     req.ExcludeGlob,
		HashOnMatchOnly: req.HashOnMatchOnly,
	}
	if err := database.DB.Create(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}

func (s *FIMService) UpdateWatchPath(id uint, req UpdateFIMWatchPathRequest) (*FIMWatchPath, error) {
	var item FIMWatchPath
	if err := database.DB.First(&item, id).Error; err != nil {
		return nil, err
	}
	if req.Path != "" {
		item.Path = req.Path
	}
	if req.ScanMode != "" {
		item.ScanMode = normalizeFIMWatchPathScanMode(req.ScanMode)
	}
	if req.Recursive != nil {
		item.Recursive = *req.Recursive
	}
	if req.MaxDepth >= 0 {
		item.MaxDepth = req.MaxDepth
	}
	if req.FileGlob != "" {
		item.FileGlob = req.FileGlob
	}
	if req.ExcludeGlob != "" {
		item.ExcludeGlob = req.ExcludeGlob
	}
	if req.HashOnMatchOnly != nil {
		item.HashOnMatchOnly = *req.HashOnMatchOnly
	}
	if err := database.DB.Save(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *FIMService) DeleteWatchPath(id uint) error {
	return database.DB.Delete(&FIMWatchPath{}, id).Error
}

func (s *FIMService) BuildBaseline(policyID uint, serverID uint, operator string) (*FIMSnapshot, error) {
	item, err := s.executeSnapshot(policyID, serverID, "baseline", operator)
	if err != nil {
		return item, err
	}
	if err := s.ActivateBaseline(item.ID, operator); err != nil {
		return item, err
	}
	return item, nil
}

func (s *FIMService) RunScan(policyID uint, serverID uint, scanType string, operator string) (*FIMSnapshot, error) {
	return s.executeSnapshot(policyID, serverID, normalizeFIMSnapshotType(scanType), operator)
}

func (s *FIMService) ListSnapshots(filter FIMSnapshotFilter) ([]FIMSnapshot, int64, error) {
	query := database.DB.Model(&FIMSnapshot{})
	if filter.PolicyID != 0 {
		query = query.Where("policy_id = ?", filter.PolicyID)
	}
	if filter.ServerID != 0 {
		query = query.Where("server_id = ?", filter.ServerID)
	}
	if filter.SnapshotType != "" {
		query = query.Where("snapshot_type = ?", filter.SnapshotType)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []FIMSnapshot
	page := maxInt(filter.Page, 1)
	pageSize := clampInt(filter.PageSize, 10, 100)
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	if err := s.enrichSnapshots(items); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *FIMService) GetSnapshotDetail(id uint) (*FIMSnapshot, error) {
	var item FIMSnapshot
	if err := database.DB.First(&item, id).Error; err != nil {
		return nil, err
	}
	items := []FIMSnapshot{item}
	if err := s.enrichSnapshots(items); err != nil {
		return nil, err
	}
	item = items[0]
	return &item, nil
}

func (s *FIMService) ActivateBaseline(snapshotID uint, operator string) error {
	var item FIMSnapshot
	if err := database.DB.First(&item, snapshotID).Error; err != nil {
		return err
	}
	if item.Status != "success" {
		return fmt.Errorf("snapshot %d is not ready for baseline activation", snapshotID)
	}
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&FIMSnapshot{}).
			Where("policy_id = ? AND server_id = ? AND id <> ? AND snapshot_type = ?", item.PolicyID, item.ServerID, item.ID, "baseline").
			Updates(map[string]any{
				"snapshot_type": gorm.Expr("?", "manual"),
				"origin_type":   gorm.Expr("CASE WHEN origin_type = '' OR origin_type IS NULL THEN ? ELSE origin_type END", "baseline"),
			}).Error; err != nil {
			return err
		}
		return tx.Model(&FIMSnapshot{}).
			Where("id = ?", item.ID).
			Updates(map[string]any{
				"snapshot_type": "baseline",
				"finished_at":   time.Now(),
			}).Error
	})
}

func (s *FIMService) ListDiffEvents(page, pageSize int) ([]FIMDiffEvent, int64, error) {
	var total int64
	if err := database.DB.Model(&FIMDiffEvent{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []FIMDiffEvent
	if err := database.DB.Order("id DESC").Offset((maxInt(page, 1) - 1) * clampInt(pageSize, 10, 100)).Limit(clampInt(pageSize, 10, 100)).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	if err := s.enrichDiffEvents(items); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *FIMService) DeleteDiffEvent(id uint) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("diff_event_id = ?", id).Delete(&FIMAlert{}).Error; err != nil {
			return err
		}
		return tx.Delete(&FIMDiffEvent{}, id).Error
	})
}

func (s *FIMService) ListAlerts(filter FIMAlertFilter) ([]FIMAlert, int64, error) {
	query := database.DB.Model(&FIMAlert{})
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Severity != "" {
		query = query.Where("severity = ?", filter.Severity)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []FIMAlert
	page := maxInt(filter.Page, 1)
	pageSize := clampInt(filter.PageSize, 10, 100)
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	if err := s.enrichAlerts(items); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *FIMService) GetAlertDetail(id uint) (*FIMAlert, error) {
	var item FIMAlert
	if err := database.DB.First(&item, id).Error; err != nil {
		return nil, err
	}
	items := []FIMAlert{item}
	if err := s.enrichAlerts(items); err != nil {
		return nil, err
	}
	return &items[0], nil
}

func (s *FIMService) DeleteAlert(id uint) error {
	return database.DB.Delete(&FIMAlert{}, id).Error
}

func (s *FIMService) UpdateAlertStatus(id uint, status string, operator string, comment string) error {
	return database.DB.Model(&FIMAlert{}).Where("id = ?", id).Updates(map[string]any{
		"status":       status,
		"assignee":     operator,
		"updated_at":   time.Now(),
		"last_seen_at": time.Now(),
	}).Error
}

func (s *FIMService) enrichTargets(items []FIMPolicyTarget) error {
	serverIDs := make([]uint, 0, len(items))
	for _, item := range items {
		if item.ServerID != 0 {
			serverIDs = append(serverIDs, item.ServerID)
		}
	}
	serverMap, err := s.loadServerMap(serverIDs)
	if err != nil {
		return err
	}
	for index := range items {
		if server, ok := serverMap[items[index].ServerID]; ok {
			items[index].ServerName = server.Hostname
			items[index].ServerIP = server.IP
		}
	}
	return nil
}

func (s *FIMService) enrichSnapshots(items []FIMSnapshot) error {
	policyIDs := make([]uint, 0, len(items))
	serverIDs := make([]uint, 0, len(items))
	for _, item := range items {
		if item.PolicyID != 0 {
			policyIDs = append(policyIDs, item.PolicyID)
		}
		if item.ServerID != 0 {
			serverIDs = append(serverIDs, item.ServerID)
		}
	}
	policyMap, err := s.loadPolicyMap(policyIDs)
	if err != nil {
		return err
	}
	serverMap, err := s.loadServerMap(serverIDs)
	if err != nil {
		return err
	}
	for index := range items {
		if policy, ok := policyMap[items[index].PolicyID]; ok {
			items[index].PolicyName = policy.Name
		}
		if server, ok := serverMap[items[index].ServerID]; ok {
			items[index].ServerName = server.Hostname
			items[index].ServerIP = server.IP
		}
	}
	return nil
}

func (s *FIMService) enrichDiffEvents(items []FIMDiffEvent) error {
	policyIDs := make([]uint, 0, len(items))
	serverIDs := make([]uint, 0, len(items))
	for _, item := range items {
		if item.PolicyID != 0 {
			policyIDs = append(policyIDs, item.PolicyID)
		}
		if item.ServerID != 0 {
			serverIDs = append(serverIDs, item.ServerID)
		}
	}
	policyMap, err := s.loadPolicyMap(policyIDs)
	if err != nil {
		return err
	}
	serverMap, err := s.loadServerMap(serverIDs)
	if err != nil {
		return err
	}
	for index := range items {
		if policy, ok := policyMap[items[index].PolicyID]; ok {
			items[index].PolicyName = policy.Name
		}
		if server, ok := serverMap[items[index].ServerID]; ok {
			items[index].ServerName = server.Hostname
			items[index].ServerIP = server.IP
		}
	}
	return nil
}

func (s *FIMService) enrichAlerts(items []FIMAlert) error {
	policyIDs := make([]uint, 0, len(items))
	serverIDs := make([]uint, 0, len(items))
	for _, item := range items {
		if item.PolicyID != 0 {
			policyIDs = append(policyIDs, item.PolicyID)
		}
		if item.ServerID != 0 {
			serverIDs = append(serverIDs, item.ServerID)
		}
	}
	policyMap, err := s.loadPolicyMap(policyIDs)
	if err != nil {
		return err
	}
	serverMap, err := s.loadServerMap(serverIDs)
	if err != nil {
		return err
	}
	for index := range items {
		if policy, ok := policyMap[items[index].PolicyID]; ok {
			items[index].PolicyName = policy.Name
		}
		if server, ok := serverMap[items[index].ServerID]; ok {
			items[index].ServerName = server.Hostname
			items[index].ServerIP = server.IP
		}
	}
	return nil
}

func (s *FIMService) loadPolicyMap(ids []uint) (map[uint]FIMPolicy, error) {
	uniqueIDs := uniqueUintIDs(ids)
	if len(uniqueIDs) == 0 {
		return map[uint]FIMPolicy{}, nil
	}
	var items []FIMPolicy
	if err := database.DB.Where("id IN ?", uniqueIDs).Find(&items).Error; err != nil {
		return nil, err
	}
	result := make(map[uint]FIMPolicy, len(items))
	for _, item := range items {
		result[item.ID] = item
	}
	return result, nil
}

func (s *FIMService) loadServerMap(ids []uint) (map[uint]cmdb.Server, error) {
	uniqueIDs := uniqueUintIDs(ids)
	if len(uniqueIDs) == 0 {
		return map[uint]cmdb.Server{}, nil
	}
	var items []cmdb.Server
	if err := database.DB.Where("id IN ?", uniqueIDs).Find(&items).Error; err != nil {
		return nil, err
	}
	result := make(map[uint]cmdb.Server, len(items))
	for _, item := range items {
		result[item.ID] = item
	}
	return result, nil
}

type fimSSHConfig struct {
	User    string
	Auth    []ssh.AuthMethod
	Timeout time.Duration
}

type fimCollectedEntry struct {
	EntryType string
	Path   string
	Size   int64
	Mtime  *time.Time
	SHA256 string
}

type fimDiffRecord struct {
	Path      string
	EventType string
	Severity  string
	OldValue  map[string]any
	NewValue  map[string]any
}

func (s *FIMService) executeSnapshot(policyID uint, serverID uint, snapshotType string, operator string) (*FIMSnapshot, error) {
	snapshotType = normalizeFIMSnapshotType(snapshotType)
	lockKey := buildFIMExecutionLockKey(policyID, serverID)
	release, acquired := acquireFIMExecutionLock(lockKey)
	if !acquired {
		return nil, fmt.Errorf("fim execution already running for policy %d and server %d", policyID, serverID)
	}
	defer release()

	policy, server, watchPaths, err := s.loadScanContext(policyID, serverID)
	if err != nil {
		return nil, err
	}

	item := &FIMSnapshot{
		PolicyID:     policyID,
		ServerID:     serverID,
		OriginType:   defaultString(snapshotType, "scheduled"),
		SnapshotType: snapshotType,
		Status:       "running",
		Operator:     defaultString(strings.TrimSpace(operator), "system"),
		StartedAt:    time.Now(),
	}
	if err := database.DB.Create(item).Error; err != nil {
		return nil, err
	}

	entries, err := s.collectSnapshotEntries(server, watchPaths)
	if err != nil {
		s.failSnapshot(item, err)
		return item, err
	}

	if err := s.persistSnapshot(item, entries); err != nil {
		s.failSnapshot(item, err)
		return item, err
	}

	if snapshotType != "baseline" {
		if err := s.generateDiffsAndAlerts(item, policy); err != nil {
			s.failSnapshot(item, err)
			return item, err
		}
	}

	if err := database.DB.First(item, item.ID).Error; err != nil {
		return item, err
	}
	_ = operator
	return item, nil
}

func buildFIMExecutionLockKey(policyID uint, serverID uint) string {
	return fmt.Sprintf("%d:%d", policyID, serverID)
}

func acquireFIMExecutionLock(lockKey string) (func(), bool) {
	if _, loaded := fimExecutionLocks.LoadOrStore(lockKey, struct{}{}); loaded {
		return func() {}, false
	}
	return func() { fimExecutionLocks.Delete(lockKey) }, true
}

func normalizeFIMWatchPathScanMode(scanMode string) string {
	switch strings.TrimSpace(scanMode) {
	case "presence_only":
		return "presence_only"
	default:
		return "full_hash"
	}
}

func normalizeFIMSnapshotType(snapshotType string) string {
	switch strings.TrimSpace(snapshotType) {
	case "baseline", "manual", "scheduled":
		return strings.TrimSpace(snapshotType)
	default:
		return "manual"
	}
}

func (s *FIMService) loadScanContext(policyID uint, serverID uint) (*FIMPolicy, *cmdb.Server, []FIMWatchPath, error) {
	var policy FIMPolicy
	if err := database.DB.First(&policy, policyID).Error; err != nil {
		return nil, nil, nil, err
	}

	var target FIMPolicyTarget
	if err := database.DB.Where("policy_id = ? AND server_id = ? AND enabled = ?", policyID, serverID, true).First(&target).Error; err != nil {
		return nil, nil, nil, fmt.Errorf("fim target not found for policy %d and server %d: %w", policyID, serverID, err)
	}

	var server cmdb.Server
	if err := database.DB.First(&server, serverID).Error; err != nil {
		return nil, nil, nil, err
	}

	var watchPaths []FIMWatchPath
	if err := database.DB.Where("policy_id = ?", policyID).Order("id ASC").Find(&watchPaths).Error; err != nil {
		return nil, nil, nil, err
	}
	if len(watchPaths) == 0 {
		return nil, nil, nil, fmt.Errorf("fim policy %d has no watch paths configured", policyID)
	}

	return &policy, &server, watchPaths, nil
}

func (s *FIMService) collectSnapshotEntries(server *cmdb.Server, watchPaths []FIMWatchPath) ([]fimCollectedEntry, error) {
	command, err := buildFIMCollectCommand(watchPaths)
	if err != nil {
		return nil, err
	}
	output, err := runFIMSSHCommand(*server, command)
	if err != nil {
		return nil, err
	}
	return parseFIMCollectOutput(output)
}

func (s *FIMService) persistSnapshot(snapshot *FIMSnapshot, entries []fimCollectedEntry) error {
	entryModels := make([]FIMSnapshotEntry, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if entry.Path == "" {
			continue
		}
		if _, ok := seen[entry.Path]; ok {
			continue
		}
		seen[entry.Path] = struct{}{}
		entryType := strings.TrimSpace(entry.EntryType)
		if entryType == "" {
			entryType = "file"
		}
		entryModels = append(entryModels, FIMSnapshotEntry{
			SnapshotID: snapshot.ID,
			Path:       entry.Path,
			EntryType:  entryType,
			Size:       entry.Size,
			Mtime:      entry.Mtime,
			SHA256:     entry.SHA256,
		})
	}

	finishedAt := time.Now()
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if len(entryModels) > 0 {
			if err := tx.Create(&entryModels).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&FIMSnapshot{}).Where("id = ?", snapshot.ID).Updates(map[string]any{
			"status":      "success",
			"finished_at": &finishedAt,
			"entry_count": len(entryModels),
		}).Error; err != nil {
			return err
		}
		return tx.Model(&FIMPolicyTarget{}).
			Where("policy_id = ? AND server_id = ?", snapshot.PolicyID, snapshot.ServerID).
			Updates(map[string]any{
				"last_scan_at":     &finishedAt,
				"last_scan_status": "success",
				"updated_at":       finishedAt,
			}).Error
	})
}

func (s *FIMService) generateDiffsAndAlerts(snapshot *FIMSnapshot, policy *FIMPolicy) error {
	compareSnapshot, err := s.findCompareSnapshot(snapshot, policy)
	if err != nil {
		return err
	}
	if compareSnapshot == nil {
		return nil
	}

	currentEntries, err := s.loadSnapshotEntries(snapshot.ID)
	if err != nil {
		return err
	}
	baselineEntries, err := s.loadSnapshotEntries(compareSnapshot.ID)
	if err != nil {
		return err
	}

	diffs := compareFIMEntries(baselineEntries, currentEntries, policy.Severity)
	if len(diffs) == 0 {
		return nil
	}

	now := time.Now()
	serverMap, err := s.loadServerMap([]uint{snapshot.ServerID})
	if err != nil {
		return err
	}
	server := serverMap[snapshot.ServerID]
	diffModels := make([]FIMDiffEvent, 0, len(diffs))
	newAlertModels := make([]FIMAlert, 0, len(diffs))
	for _, diff := range diffs {
		oldJSON, _ := json.Marshal(diff.OldValue)
		newJSON, _ := json.Marshal(diff.NewValue)
		diffModels = append(diffModels, FIMDiffEvent{
			PolicyID:           snapshot.PolicyID,
			ServerID:           snapshot.ServerID,
			PolicyName:         policy.Name,
			ServerName:         server.Hostname,
			ServerIP:           server.IP,
			BaselineSnapshotID: &compareSnapshot.ID,
			CurrentSnapshotID:  snapshot.ID,
			Path:               diff.Path,
			EventType:          diff.EventType,
			Severity:           diff.Severity,
			OldValueJSON:       string(oldJSON),
			NewValueJSON:       string(newJSON),
			OccurredAt:         now,
		})
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&diffModels).Error; err != nil {
			return err
		}
		existingAlerts, err := loadReusableFIMAlerts(tx, snapshot.PolicyID, snapshot.ServerID, diffModels)
		if err != nil {
			return err
		}

		for _, diff := range diffModels {
			key := buildFIMAlertDedupKey(diff.Path, diff.EventType)
			if existing, ok := existingAlerts[key]; ok {
				updates := buildFIMAlertReuseUpdates(existing, diff, now)
				updates["updated_at"] = now
				if err := tx.Model(&FIMAlert{}).Where("id = ?", existing.ID).Updates(updates).Error; err != nil {
					return err
				}
				continue
			}

			newAlert := FIMAlert{
				DiffEventID:     diff.ID,
				PolicyID:        diff.PolicyID,
				ServerID:        diff.ServerID,
				Path:            diff.Path,
				EventType:       diff.EventType,
				Title:           buildFIMAlertTitle(diff),
				Summary:         buildFIMAlertSummary(diff),
				Severity:        diff.Severity,
				Status:          "open",
				OccurrenceCount: 1,
				FirstSeenAt:     now,
				LastSeenAt:      now,
			}
			newAlertModels = append(newAlertModels, newAlert)
			existingAlerts[key] = newAlert
		}

		if len(newAlertModels) == 0 {
			return nil
		}
		if err := tx.Create(&newAlertModels).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	if err := s.notifyFIMAlerts(policy, snapshot, newAlertModels); err != nil {
		log.Printf("[fim notify] send failed policy=%d snapshot=%d: %v", snapshot.PolicyID, snapshot.ID, err)
	}
	return nil
}

func loadReusableFIMAlerts(tx *gorm.DB, policyID, serverID uint, diffs []FIMDiffEvent) (map[string]FIMAlert, error) {
	paths := make([]string, 0, len(diffs))
	eventTypes := make([]string, 0, len(diffs))
	for _, diff := range diffs {
		if strings.TrimSpace(diff.Path) != "" {
			paths = append(paths, diff.Path)
		}
		if strings.TrimSpace(diff.EventType) != "" {
			eventTypes = append(eventTypes, diff.EventType)
		}
	}
	if len(paths) == 0 || len(eventTypes) == 0 {
		return map[string]FIMAlert{}, nil
	}

	var items []FIMAlert
	if err := tx.Where("policy_id = ? AND server_id = ? AND status IN ? AND path IN ? AND event_type IN ?", policyID, serverID, []string{"open", "acknowledged"}, uniqueStrings(paths), uniqueStrings(eventTypes)).
		Order("id DESC").
		Find(&items).Error; err != nil {
		return nil, err
	}

	result := make(map[string]FIMAlert, len(items))
	for _, item := range items {
		key := buildFIMAlertDedupKey(item.Path, item.EventType)
		if _, exists := result[key]; exists {
			continue
		}
		result[key] = item
	}
	return result, nil
}

func buildFIMAlertDedupKey(path, eventType string) string {
	return strings.TrimSpace(path) + "::" + strings.TrimSpace(eventType)
}

func buildFIMAlertReuseUpdates(existing FIMAlert, diff FIMDiffEvent, now time.Time) map[string]any {
	return map[string]any{
		"diff_event_id":    diff.ID,
		"title":            buildFIMAlertTitle(diff),
		"summary":          buildFIMAlertSummary(diff),
		"severity":         diff.Severity,
		"last_seen_at":     now,
		"occurrence_count": existing.OccurrenceCount + 1,
	}
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func (s *FIMService) notifyFIMAlerts(policy *FIMPolicy, snapshot *FIMSnapshot, alerts []FIMAlert) error {
	if policy == nil || strings.TrimSpace(policy.NotifyChannels) == "" || len(alerts) == 0 {
		return nil
	}

	serverMap, err := s.loadServerMap([]uint{snapshot.ServerID})
	if err != nil {
		return err
	}
	server := serverMap[snapshot.ServerID]

	channels, err := s.loadNotifyChannels(policy.NotifyChannels)
	if err != nil {
		return err
	}
	if len(channels) == 0 {
		return nil
	}

	var sendErrors []string
	for _, item := range alerts {
		msg := buildFIMNotifyMessage(policy, server, item)
		for _, channel := range channels {
			if err := alert.SendNotification(&channel, msg); err != nil {
				sendErrors = append(sendErrors, fmt.Sprintf("%s: %v", channel.Name, err))
			}
		}
	}
	if len(sendErrors) > 0 {
		return fmt.Errorf("fim alert notifications failed: %s", strings.Join(sendErrors, "; "))
	}
	return nil
}

func (s *FIMService) loadNotifyChannels(raw string) ([]alert.NotifyChannel, error) {
	ids := parseNotifyChannelIDs(raw)
	if len(ids) == 0 {
		return nil, nil
	}

	var channels []alert.NotifyChannel
	if err := database.DB.Where("id IN ? AND enabled = ?", ids, true).Find(&channels).Error; err != nil {
		return nil, err
	}
	return channels, nil
}

func buildFIMNotifyMessage(policy *FIMPolicy, server cmdb.Server, item FIMAlert) *alert.NotifyMessage {
	source := strings.TrimSpace(server.Hostname)
	if server.IP != "" {
		if source != "" {
			source = fmt.Sprintf("%s (%s)", source, server.IP)
		} else {
			source = server.IP
		}
	}
	if source == "" {
		source = fmt.Sprintf("server-%d", item.ServerID)
	}

	content := item.Summary
	if content == "" {
		content = item.Title
	}
	content = fmt.Sprintf("%s 时间: %s", strings.TrimSpace(content), alert.FormatTimeLocal(item.FirstSeenAt))

	return &alert.NotifyMessage{
		Title:    fmt.Sprintf("【完整性告警】%s", item.Title),
		RuleName: policy.Name,
		Content:  content,
		Severity: mapFIMSeverityToAlertSeverity(item.Severity),
		Status:   "firing",
		Source:   source,
		Category: "system",
		Time:     alert.FormatTimeLocal(item.FirstSeenAt),
	}
}

func parseNotifyChannelIDs(raw string) []uint {
	parts := strings.Split(raw, ",")
	ids := make([]uint, 0, len(parts))
	seen := make(map[uint]struct{}, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		value, err := strconv.ParseUint(part, 10, 64)
		if err != nil || value == 0 {
			continue
		}
		id := uint(value)
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

func normalizeNotifyChannels(raw string) string {
	ids := parseNotifyChannelIDs(raw)
	if len(ids) == 0 {
		return ""
	}
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, strconv.FormatUint(uint64(id), 10))
	}
	return strings.Join(parts, ",")
}

func mapFIMSeverityToAlertSeverity(severity string) string {
	switch severity {
	case "critical":
		return "critical"
	case "high", "warning", "medium":
		return "warning"
	default:
		return "info"
	}
}

func (s *FIMService) findCompareSnapshot(snapshot *FIMSnapshot, policy *FIMPolicy) (*FIMSnapshot, error) {
	if policy != nil && policy.CompareMode == "last_snapshot" {
		return s.findPreviousSuccessfulSnapshot(snapshot.PolicyID, snapshot.ServerID, snapshot.ID)
	}
	return s.findBaselineSnapshot(snapshot.PolicyID, snapshot.ServerID, snapshot.ID)
}

func (s *FIMService) findBaselineSnapshot(policyID uint, serverID uint, excludeID uint) (*FIMSnapshot, error) {
	var item FIMSnapshot
	err := database.DB.
		Where("policy_id = ? AND server_id = ? AND snapshot_type = ? AND status = ? AND id <> ?", policyID, serverID, "baseline", "success", excludeID).
		Order("id DESC").
		First(&item).Error
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "record not found") {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (s *FIMService) findPreviousSuccessfulSnapshot(policyID uint, serverID uint, excludeID uint) (*FIMSnapshot, error) {
	var item FIMSnapshot
	err := database.DB.
		Where("policy_id = ? AND server_id = ? AND status = ? AND id <> ?", policyID, serverID, "success", excludeID).
		Order("id DESC").
		First(&item).Error
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "record not found") {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (s *FIMService) loadSnapshotEntries(snapshotID uint) (map[string]FIMSnapshotEntry, error) {
	var entries []FIMSnapshotEntry
	if err := database.DB.Where("snapshot_id = ?", snapshotID).Find(&entries).Error; err != nil {
		return nil, err
	}
	result := make(map[string]FIMSnapshotEntry, len(entries))
	for _, entry := range entries {
		result[entry.Path] = entry
	}
	return result, nil
}

func (s *FIMService) failSnapshot(snapshot *FIMSnapshot, runErr error) {
	if snapshot == nil {
		return
	}
	message := runErr.Error()
	if len(message) > 4000 {
		message = message[:4000]
	}
	finishedAt := time.Now()
	_ = database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&FIMSnapshot{}).Where("id = ?", snapshot.ID).Updates(map[string]any{
			"status":        "failed",
			"finished_at":   &finishedAt,
			"error_message": message,
		}).Error; err != nil {
			return err
		}
		return tx.Model(&FIMPolicyTarget{}).
			Where("policy_id = ? AND server_id = ?", snapshot.PolicyID, snapshot.ServerID).
			Updates(map[string]any{
				"last_scan_at":     &finishedAt,
				"last_scan_status": "failed",
				"updated_at":       finishedAt,
			}).Error
	})
}

func runFIMSSHCommand(server cmdb.Server, command string) (string, error) {
	sshConfig, err := loadFIMSSHConfig()
	if err != nil {
		return "", err
	}

	address := fmt.Sprintf("%s:%d", server.IP, defaultInt(server.SSHPort, 22))
	
	// 使用严格模式的主机密钥验证
	strictCallback := NewStrictPostKeyCallback(&server.ID, nil, nil)
	
	client, err := ssh.Dial("tcp", address, &ssh.ClientConfig{
		User:            sshConfig.User,
		Auth:            sshConfig.Auth,
		HostKeyCallback: strictCallback.VerifyHostKey,
		Timeout:         sshConfig.Timeout,
	})
	if err != nil {
		return "", fmt.Errorf("ssh dial %s failed: %w", address, err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("ssh session create failed for %s: %w", address, err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		return string(output), fmt.Errorf("ssh command failed for %s: %w; output=%s", address, err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

func loadFIMSSHConfig() (*fimSSHConfig, error) {
	if stored, err := loadStoredFIMSSHConfig(); err != nil {
		return nil, err
	} else if stored != nil {
		return stored, nil
	}

	user := strings.TrimSpace(os.Getenv("FIM_SSH_USER"))
	if user == "" {
		return nil, fmt.Errorf("missing FIM_SSH_USER")
	}

	auth := make([]ssh.AuthMethod, 0, 2)
	if password := os.Getenv("FIM_SSH_PASSWORD"); password != "" {
		auth = append(auth, ssh.Password(password))
	}

	privateKey := strings.TrimSpace(os.Getenv("FIM_SSH_PRIVATE_KEY"))
	if privateKey == "" {
		if keyPath := strings.TrimSpace(os.Getenv("FIM_SSH_PRIVATE_KEY_PATH")); keyPath != "" {
			content, err := os.ReadFile(keyPath)
			if err != nil {
				return nil, fmt.Errorf("read FIM ssh private key failed: %w", err)
			}
			privateKey = string(content)
		}
	}
	if privateKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(privateKey))
		if err != nil {
			return nil, fmt.Errorf("parse FIM ssh private key failed: %w", err)
		}
		auth = append(auth, ssh.PublicKeys(signer))
	}
	if len(auth) == 0 {
		return nil, fmt.Errorf("missing FIM ssh auth, set FIM_SSH_PASSWORD or FIM_SSH_PRIVATE_KEY/FIM_SSH_PRIVATE_KEY_PATH")
	}

	timeoutSec := 15
	if raw := strings.TrimSpace(os.Getenv("FIM_SSH_TIMEOUT_SEC")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			timeoutSec = parsed
		}
	}

	return &fimSSHConfig{
		User:    user,
		Auth:    auth,
		Timeout: time.Duration(timeoutSec) * time.Second,
	}, nil
}

func loadStoredFIMSSHConfig() (*fimSSHConfig, error) {
	var setting models.FIMSSHSetting
	if err := database.DB.First(&setting).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	user := strings.TrimSpace(setting.SSHUser)
	if user == "" {
		return nil, nil
	}

	auth := make([]ssh.AuthMethod, 0, 1)
	switch setting.AuthMode {
	case "", "password":
		password, err := secureconfig.DecryptString(setting.PasswordEncrypted)
		if err != nil {
			return nil, fmt.Errorf("decrypt fim ssh password failed: %w", err)
		}
		if strings.TrimSpace(password) == "" {
			return nil, fmt.Errorf("missing stored FIM ssh password")
		}
		auth = append(auth, ssh.Password(password))
	case "private_key":
		privateKey, err := secureconfig.DecryptString(setting.PrivateKeyEncrypted)
		if err != nil {
			return nil, fmt.Errorf("decrypt fim ssh private key failed: %w", err)
		}
		if strings.TrimSpace(privateKey) == "" {
			return nil, fmt.Errorf("missing stored FIM ssh private key")
		}
		signer, err := ssh.ParsePrivateKey([]byte(privateKey))
		if err != nil {
			return nil, fmt.Errorf("parse stored FIM ssh private key failed: %w", err)
		}
		auth = append(auth, ssh.PublicKeys(signer))
	default:
		return nil, fmt.Errorf("unsupported stored FIM ssh auth mode: %s", setting.AuthMode)
	}

	timeoutSec := setting.TimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = 15
	}
	return &fimSSHConfig{
		User:    user,
		Auth:    auth,
		Timeout: time.Duration(timeoutSec) * time.Second,
	}, nil
}

// 安全路径验证：只允许字母、数字、下划线、斜杠、点、连字符
var safePathPattern = regexp.MustCompile(`^[a-zA-Z0-9_/.-]+$`)

// 安全 glob 验证：允许字母、数字、下划线、点、连字符、通配符 * 和 ?
var safeGlobPattern = regexp.MustCompile(`^[a-zA-Z0-9_.*?-]+$`)

func shellSingleQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

// validateWatchPathInput 验证监控路径输入的安全性
func validateWatchPathInput(watchPath FIMWatchPath) error {
	path := strings.TrimSpace(watchPath.Path)
	if path == "" {
		return nil // 跳过空路径
	}

	// 路径必须是绝对路径
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("path must be absolute: %s", path)
	}

	// 禁止危险字符
	if !safePathPattern.MatchString(path) {
		return fmt.Errorf("path contains dangerous characters: %s", path)
	}

	// 禁止路径遍历（防止 /../ 攻击）
	if strings.Contains(path, "..") {
		return fmt.Errorf("path traversal not allowed: %s", path)
	}

	// 验证 FileGlob
	if glob := strings.TrimSpace(watchPath.FileGlob); glob != "" {
		if !safeGlobPattern.MatchString(glob) {
			return fmt.Errorf("file_glob contains dangerous characters: %s", glob)
		}
	}

	// 验证 ExcludeGlob
	if exclude := strings.TrimSpace(watchPath.ExcludeGlob); exclude != "" {
		if !safeGlobPattern.MatchString(exclude) {
			return fmt.Errorf("exclude_glob contains dangerous characters: %s", exclude)
		}
	}

	return nil
}

func buildFIMCollectCommand(watchPaths []FIMWatchPath) (string, error) {
	// 预编译 safePathPattern（如果尚未编译）
	if !safePathPattern.MatchString("/test") {
		return "", fmt.Errorf("safe path pattern validation failed")
	}

	parts := []string{"set -eu"}
	validCount := 0

	for _, watchPath := range watchPaths {
		path := strings.TrimSpace(watchPath.Path)
		if path == "" {
			continue
		}

		// 安全验证
		if err := validateWatchPathInput(watchPath); err != nil {
			return "", fmt.Errorf("invalid watch path input: %w", err)
		}

		validCount++
		quotedPath := shellSingleQuote(path)
		scanMode := normalizeFIMWatchPathScanMode(watchPath.ScanMode)

		// 构建 find 命令 - 使用数组而非字符串拼接
		findParts := []string{"find", quotedPath}
		if !watchPath.Recursive {
			findParts = append(findParts, "-maxdepth", "1")
		} else if watchPath.MaxDepth > 0 {
			findParts = append(findParts, "-maxdepth", strconv.Itoa(watchPath.MaxDepth))
		}
		findParts = append(findParts, "-type", "f")

		if glob := strings.TrimSpace(watchPath.FileGlob); glob != "" {
			findParts = append(findParts, "-name", shellSingleQuote(glob))
		}
		if exclude := strings.TrimSpace(watchPath.ExcludeGlob); exclude != "" {
			if strings.Contains(exclude, "/") {
				findParts = append(findParts, "!", "-path", shellSingleQuote(exclude))
			} else {
				findParts = append(findParts, "!", "-name", shellSingleQuote(exclude))
			}
		}
		findParts = append(findParts, "-print0")

		// 使用简化命令，减少 shell 特殊字符依赖
		// 注意：find -print0 输出由 NUL 分隔，由 read -d '' 读取
	collectOutput := strings.Join(findParts, " ")+` | while IFS= read -r -d '' file; do
  printf 'P\t%s\n' "$file"
done`
		if scanMode != "presence_only" {
			collectOutput = strings.Join(findParts, " ")+` | while IFS= read -r -d '' file; do
  size=$(stat -c %s "$file" 2>/dev/null || stat -f %z "$file" 2>/dev/null || echo 0)
  mtime=$(stat -c %Y "$file" 2>/dev/null || stat -f %m "$file" 2>/dev/null || echo 0)
  sha=$(sha256sum "$file" 2>/dev/null | awk '{print $1}')
  if [ -z "$sha" ]; then
    sha=$(shasum -a 256 "$file" 2>/dev/null | awk '{print $1}')
  fi
  printf 'F\t%s\t%s\t%s\t%s\n' "$file" "$size" "$mtime" "$sha"
done`
		}
		parts = append(parts,
			fmt.Sprintf("if [ -e %s ]; then", quotedPath),
			collectOutput,
			"fi",
		)
	}
	if validCount == 0 {
		return "", fmt.Errorf("no valid fim watch paths configured")
	}
	return strings.Join(parts, "\n"), nil
}

func parseFIMCollectOutput(output string) ([]fimCollectedEntry, error) {
	lines := strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n")
	items := make([]fimCollectedEntry, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) == 2 && fields[0] == "P" {
			items = append(items, fimCollectedEntry{
				EntryType: "presence",
				Path:      fields[1],
			})
			continue
		}
		if len(fields) != 5 || fields[0] != "F" {
			return nil, fmt.Errorf("unexpected fim collect output line: %s", line)
		}
		size, err := strconv.ParseInt(fields[2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse fim file size failed for %s: %w", fields[1], err)
		}
		mtimeUnix, err := strconv.ParseInt(fields[3], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse fim mtime failed for %s: %w", fields[1], err)
		}
		mtime := time.Unix(mtimeUnix, 0)
		items = append(items, fimCollectedEntry{
			EntryType: "file",
			Path:      fields[1],
			Size:      size,
			Mtime:     &mtime,
			SHA256:    fields[4],
		})
	}
	return items, nil
}

func compareFIMEntries(baseline map[string]FIMSnapshotEntry, current map[string]FIMSnapshotEntry, defaultSeverity string) []fimDiffRecord {
	records := make([]fimDiffRecord, 0)

	for path, currentEntry := range current {
		baselineEntry, exists := baseline[path]
		if !exists {
			continue
		}
		if fimEntryPresenceOnly(baselineEntry, currentEntry) {
			continue
		}
		if fimEntryChanged(baselineEntry, currentEntry) {
			records = append(records, fimDiffRecord{
				Path:      path,
				EventType: "modify",
				Severity:  deriveFIMSeverity("modify", defaultSeverity),
				OldValue:  snapshotEntryToMap(baselineEntry),
				NewValue:  snapshotEntryToMap(currentEntry),
			})
		}
	}

	for path, baselineEntry := range baseline {
		if _, exists := current[path]; exists {
			continue
		}
		records = append(records, fimDiffRecord{
			Path:      path,
			EventType: "delete",
			Severity:  deriveFIMSeverity("delete", defaultSeverity),
			OldValue:  snapshotEntryToMap(baselineEntry),
		})
	}

	return records
}

func fimEntryPresenceOnly(entries ...FIMSnapshotEntry) bool {
	for _, entry := range entries {
		if strings.TrimSpace(entry.EntryType) == "presence" {
			return true
		}
	}
	return false
}

func fimEntryChanged(baseline FIMSnapshotEntry, current FIMSnapshotEntry) bool {
	if baseline.Size != current.Size {
		return true
	}
	if baseline.SHA256 != current.SHA256 {
		return true
	}
	if baseline.Mtime == nil && current.Mtime == nil {
		return false
	}
	if baseline.Mtime == nil || current.Mtime == nil {
		return true
	}
	return !baseline.Mtime.Equal(*current.Mtime)
}

func deriveFIMSeverity(eventType string, fallback string) string {
	switch eventType {
	case "delete", "modify":
		return "high"
	default:
		return defaultString(fallback, "high")
	}
}

func snapshotEntryToMap(entry FIMSnapshotEntry) map[string]any {
	result := map[string]any{
		"path":   entry.Path,
		"size":   entry.Size,
		"sha256": entry.SHA256,
	}
	if entry.Mtime != nil {
		result["mtime"] = entry.Mtime.Format(time.RFC3339)
	}
	return result
}

func buildFIMAlertTitle(diff FIMDiffEvent) string {
	switch diff.EventType {
	case "delete":
		return fmt.Sprintf("检测到文件删除: %s", diff.Path)
	default:
		return fmt.Sprintf("检测到文件变更: %s", diff.Path)
	}
}

func buildFIMAlertSummary(diff FIMDiffEvent) string {
	policyName := strings.TrimSpace(diff.PolicyName)
	if policyName == "" {
		policyName = fmt.Sprintf("策略 #%d", diff.PolicyID)
	}

	serverName := strings.TrimSpace(diff.ServerName)
	switch {
	case serverName != "" && diff.ServerIP != "":
		serverName = fmt.Sprintf("%s (%s)", serverName, diff.ServerIP)
	case serverName == "" && diff.ServerIP != "":
		serverName = diff.ServerIP
	case serverName == "":
		serverName = fmt.Sprintf("主机 #%d", diff.ServerID)
	}

	return fmt.Sprintf("%s 在 %s 上检测到%s，路径 %s。", policyName, serverName, formatFIMEventTypeLabel(diff.EventType), diff.Path)
}

func formatFIMEventTypeLabel(eventType string) string {
	switch eventType {
	case "delete":
		return "文件删除"
	case "modify":
		return "文件变更"
	default:
		return eventType
	}
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func uniqueUintIDs(ids []uint) []uint {
	if len(ids) == 0 {
		return nil
	}
	result := make([]uint, 0, len(ids))
	seen := make(map[uint]struct{}, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func defaultInt(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

func maxInt(value, min int) int {
	if value < min {
		return min
	}
	return value
}

func clampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
