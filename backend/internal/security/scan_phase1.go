package security

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/edy/ops-platform/internal/models"
	"gorm.io/gorm"
)

type scanPhase1SchemaSupport struct {
	runExtensions    bool
	targetsTable     bool
	evidencesTable   bool
	occurrencesTable bool
}

func (s scanPhase1SchemaSupport) foundationTablesReady() bool {
	return s.targetsTable && s.evidencesTable && s.occurrencesTable
}

var (
	scanPhase1SchemaOnce   sync.Once
	scanPhase1SchemaCached scanPhase1SchemaSupport
)

func getScanPhase1SchemaSupport(tx *gorm.DB) scanPhase1SchemaSupport {
	scanPhase1SchemaOnce.Do(func() {
		scanPhase1SchemaCached = scanPhase1SchemaSupport{
			runExtensions:    scanPhase1ColumnExists(tx, "security_scan_runs", "phase"),
			targetsTable:     scanPhase1TableExists(tx, "security_scan_targets"),
			evidencesTable:   scanPhase1TableExists(tx, "security_scan_evidences"),
			occurrencesTable: scanPhase1TableExists(tx, "security_scan_finding_occurrences"),
		}
	})
	return scanPhase1SchemaCached
}

func scanPhase1TableExists(tx *gorm.DB, tableName string) bool {
	if tx == nil {
		return false
	}
	var count int64
	if err := tx.Raw(
		"SELECT COUNT(1) FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?",
		tableName,
	).Scan(&count).Error; err != nil {
		return false
	}
	return count > 0
}

func scanPhase1ColumnExists(tx *gorm.DB, tableName string, columnName string) bool {
	if tx == nil {
		return false
	}
	var count int64
	if err := tx.Raw(
		"SELECT COUNT(1) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ?",
		tableName,
		columnName,
	).Scan(&count).Error; err != nil {
		return false
	}
	return count > 0
}

func phase1RunConfigSnapshot(task models.SecurityScanTask) *string {
	return phase1JSONString(map[string]interface{}{
		"task_name":        task.Name,
		"scan_type":        task.ScanType,
		"target_type":      task.TargetType,
		"target":           task.Target,
		"nuclei_version":   task.NucleiVersion,
		"template_version": task.TemplateVersion,
		"created_by":       task.CreatedBy,
		"created_at":       task.CreatedAt,
		"latest_run_id":    task.LatestRunID,
		"current_run_id":   task.CurrentRunID,
	})
}

func phase1JSONString(payload interface{}) *string {
	if payload == nil {
		return nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil
	}
	serialized := string(data)
	return &serialized
}

func phase1Digest(parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		_, _ = h.Write([]byte(part))
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func phase1TerminalRunPhase(status string) string {
	switch strings.TrimSpace(status) {
	case models.TaskStatusCompleted:
		return "completed"
	case models.TaskStatusFailed:
		return "failed"
	case models.TaskStatusCancelled:
		return "cancelled"
	default:
		return ""
	}
}

func phase1VerificationStatus(vuln models.SecurityVulnerability) string {
	if status := strings.TrimSpace(vuln.VerificationStatus); status != "" {
		return status
	}
	if status := strings.TrimSpace(vuln.ReviewStatus); status != "" {
		return status
	}
	return "pending"
}

func phase1FindingKey(vuln models.SecurityVulnerability) string {
	identifier := firstNonEmpty(
		strings.TrimSpace(vuln.PrimaryCVEID),
		strings.TrimSpace(vuln.CVEID),
		strings.TrimSpace(vuln.TemplateID),
		strings.TrimSpace(vuln.Title),
	)
	return strings.Join([]string{
		firstNonEmpty(strings.TrimSpace(vuln.FindingSource), "legacy"),
		firstNonEmpty(normalizeScanHost(vuln.IP), strings.TrimSpace(vuln.IP)),
		strconv.Itoa(vuln.Port),
		identifier,
		firstNonEmpty(strings.TrimSpace(vuln.MatchMode), "unknown"),
	}, "|")
}

func phase1TrimText(value string, limit int) string {
	value = strings.TrimSpace(strings.ToValidUTF8(value, ""))
	if limit <= 0 {
		return value
	}
	return truncateUTF8ByBytes(value, limit)
}

type hostScanPhase1Context struct {
	taskID         uint
	runID          uint
	ipTarget       *models.SecurityScanTarget
	serviceTargets map[int]*models.SecurityScanTarget
}

func (c *hostScanPhase1Context) targetIDForPort(port int) *uint {
	if c == nil {
		return nil
	}
	if target, ok := c.serviceTargets[port]; ok && target != nil {
		return &target.ID
	}
	if c.ipTarget != nil {
		return &c.ipTarget.ID
	}
	return nil
}

func loadCurrentRunIDForTask(tx *gorm.DB, taskID uint) (uint, error) {
	var task models.SecurityScanTask
	if err := tx.Select("current_run_id").First(&task, taskID).Error; err != nil {
		return 0, err
	}
	if task.CurrentRunID == nil {
		return 0, nil
	}
	return *task.CurrentRunID, nil
}

func hostIPTargetKey(host string) string {
	return "ip|" + firstNonEmpty(normalizeScanHost(host), strings.TrimSpace(host))
}

func hostServiceTargetKey(host string, port int, service string) string {
	return strings.Join([]string{
		"service",
		firstNonEmpty(normalizeScanHost(host), strings.TrimSpace(host)),
		strconv.Itoa(port),
		strings.ToLower(strings.TrimSpace(service)),
	}, "|")
}

func upsertPhase1Target(tx *gorm.DB, target models.SecurityScanTarget) (*models.SecurityScanTarget, error) {
	var existing models.SecurityScanTarget
	err := tx.Where("run_id = ? AND normalized_target = ?", target.RunID, target.NormalizedTarget).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		if err := tx.Create(&target).Error; err != nil {
			return nil, err
		}
		return &target, nil
	}
	if err != nil {
		return nil, err
	}

	updates := map[string]interface{}{
		"parent_target_id": existing.ParentTargetID,
		"host":             target.Host,
		"port":             target.Port,
		"scheme":           target.Scheme,
		"path":             target.Path,
		"service_name":     target.ServiceName,
		"product_name":     target.ProductName,
		"version":          target.Version,
		"status":           target.Status,
		"discovery_source": target.DiscoverySource,
		"started_at":       target.StartedAt,
		"completed_at":     target.CompletedAt,
		"metadata_json":    target.MetadataJSON,
		"updated_at":       time.Now(),
	}
	if target.ParentTargetID != nil {
		updates["parent_target_id"] = *target.ParentTargetID
	}
	if err := tx.Model(&existing).Updates(updates).Error; err != nil {
		return nil, err
	}

	existing.Host = target.Host
	existing.Port = target.Port
	existing.Scheme = target.Scheme
	existing.Path = target.Path
	existing.ServiceName = target.ServiceName
	existing.ProductName = target.ProductName
	existing.Version = target.Version
	existing.Status = target.Status
	existing.DiscoverySource = target.DiscoverySource
	existing.StartedAt = target.StartedAt
	existing.CompletedAt = target.CompletedAt
	existing.MetadataJSON = target.MetadataJSON
	if target.ParentTargetID != nil {
		existing.ParentTargetID = target.ParentTargetID
	}
	return &existing, nil
}

func upsertPhase1Evidence(tx *gorm.DB, evidence models.SecurityScanEvidence) (*models.SecurityScanEvidence, error) {
	var existing models.SecurityScanEvidence
	query := tx.Where("run_id = ? AND digest = ?", evidence.RunID, evidence.Digest)
	if strings.TrimSpace(evidence.Digest) == "" {
		query = tx.Where(
			"run_id = ? AND target_id = ? AND evidence_type = ? AND source_engine = ?",
			evidence.RunID,
			evidence.TargetID,
			evidence.EvidenceType,
			evidence.SourceEngine,
		)
	}
	err := query.First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		if err := tx.Create(&evidence).Error; err != nil {
			return nil, err
		}
		return &evidence, nil
	}
	if err != nil {
		return nil, err
	}

	updates := map[string]interface{}{
		"request_excerpt":  evidence.RequestExcerpt,
		"response_excerpt": evidence.ResponseExcerpt,
		"payload_excerpt":  evidence.PayloadExcerpt,
		"metadata_json":    evidence.MetadataJSON,
		"raw_json":         evidence.RawJSON,
		"storage_ref":      evidence.StorageRef,
		"updated_at":       time.Now(),
	}
	if err := tx.Model(&existing).Updates(updates).Error; err != nil {
		return nil, err
	}
	existing.RequestExcerpt = evidence.RequestExcerpt
	existing.ResponseExcerpt = evidence.ResponseExcerpt
	existing.PayloadExcerpt = evidence.PayloadExcerpt
	existing.MetadataJSON = evidence.MetadataJSON
	existing.RawJSON = evidence.RawJSON
	existing.StorageRef = evidence.StorageRef
	return &existing, nil
}

func upsertPhase1Occurrence(tx *gorm.DB, occurrence models.SecurityScanFindingOccurrence) error {
	var existing models.SecurityScanFindingOccurrence
	query := tx.Where("run_id = ? AND finding_key = ?", occurrence.RunID, occurrence.FindingKey)
	if occurrence.LegacyVulnerabilityID != nil {
		query = tx.Where("run_id = ? AND legacy_vulnerability_id = ?", occurrence.RunID, *occurrence.LegacyVulnerabilityID)
	}

	err := query.First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return tx.Create(&occurrence).Error
	}
	if err != nil {
		return err
	}

	updates := map[string]interface{}{
		"target_id":           occurrence.TargetID,
		"finding_key":         occurrence.FindingKey,
		"finding_family":      occurrence.FindingFamily,
		"finding_source":      occurrence.FindingSource,
		"severity":            occurrence.Severity,
		"confidence":          occurrence.Confidence,
		"match_mode":          occurrence.MatchMode,
		"primary_cve_id":      occurrence.PrimaryCVEID,
		"vuln_db_id":          occurrence.VulnDBID,
		"title":               occurrence.Title,
		"status":              occurrence.Status,
		"verification_status": occurrence.VerificationStatus,
		"evidence_count":      occurrence.EvidenceCount,
		"last_seen_at":        occurrence.LastSeenAt,
		"metadata_json":       occurrence.MetadataJSON,
		"updated_at":          time.Now(),
	}
	if existing.FirstSeenAt == nil && occurrence.FirstSeenAt != nil {
		updates["first_seen_at"] = occurrence.FirstSeenAt
	}
	return tx.Model(&existing).Updates(updates).Error
}

func ensureHostScanPhase1Context(tx *gorm.DB, taskID uint, host string, ports []NmapPort) (*hostScanPhase1Context, error) {
	support := getScanPhase1SchemaSupport(tx)
	if !support.foundationTablesReady() {
		return nil, nil
	}

	runID, err := loadCurrentRunIDForTask(tx, taskID)
	if err != nil || runID == 0 {
		return nil, err
	}

	now := time.Now()
	host = firstNonEmpty(normalizeScanHost(host), strings.TrimSpace(host))
	ipTarget, err := upsertPhase1Target(tx, models.SecurityScanTarget{
		RunID:            runID,
		TaskID:           taskID,
		TargetKind:       "ip",
		NormalizedTarget: hostIPTargetKey(host),
		Host:             host,
		Status:           "completed",
		DiscoverySource:  "nmap",
		StartedAt:        &now,
		CompletedAt:      &now,
		MetadataJSON:     phase1JSONString(map[string]interface{}{"port_count": len(ports)}),
	})
	if err != nil {
		return nil, err
	}

	ctx := &hostScanPhase1Context{
		taskID:         taskID,
		runID:          runID,
		ipTarget:       ipTarget,
		serviceTargets: make(map[int]*models.SecurityScanTarget, len(ports)),
	}

	for _, port := range ports {
		portValue := port.PortID
		targetMetadata := phase1JSONString(map[string]interface{}{
			"state":      strings.TrimSpace(port.State),
			"product":    strings.TrimSpace(port.Product),
			"cpe":        strings.TrimSpace(port.CPE),
			"method":     strings.TrimSpace(port.Method),
			"confidence": port.Confidence,
		})
		serviceTarget, err := upsertPhase1Target(tx, models.SecurityScanTarget{
			RunID:            runID,
			TaskID:           taskID,
			ParentTargetID:   &ipTarget.ID,
			TargetKind:       "service",
			NormalizedTarget: hostServiceTargetKey(host, port.PortID, port.Service),
			Host:             host,
			Port:             &portValue,
			Scheme:           strings.TrimSpace(port.Protocol),
			ServiceName:      strings.TrimSpace(port.Service),
			ProductName:      strings.TrimSpace(port.Product),
			Version:          strings.TrimSpace(port.Version),
			Status:           "completed",
			DiscoverySource:  "nmap",
			StartedAt:        &now,
			CompletedAt:      &now,
			MetadataJSON:     targetMetadata,
		})
		if err != nil {
			return nil, err
		}
		ctx.serviceTargets[port.PortID] = serviceTarget

		raw := phase1JSONString(port)
		banner := phase1TrimText(port.Banner, 4096)
		_, err = upsertPhase1Evidence(tx, models.SecurityScanEvidence{
			RunID:           runID,
			TaskID:          taskID,
			TargetID:        &serviceTarget.ID,
			EvidenceType:    "nmap-service",
			SourceEngine:    "nmap",
			Digest:          phase1Digest("nmap-service", host, strconv.Itoa(port.PortID), strings.TrimSpace(port.Service), strings.TrimSpace(port.Version), strings.TrimSpace(port.CPE), strings.TrimSpace(port.Banner)),
			ResponseExcerpt: banner,
			MetadataJSON:    targetMetadata,
			RawJSON:         raw,
		})
		if err != nil {
			return nil, err
		}
	}

	return ctx, nil
}

func recordVersionMatchOccurrence(tx *gorm.DB, ctx *hostScanPhase1Context, portID int, vuln models.SecurityVulnerability) error {
	if ctx == nil {
		return nil
	}

	targetID := ctx.targetIDForPort(portID)
	raw := phase1JSONString(map[string]interface{}{
		"cve_id":         vuln.CVEID,
		"primary_cve_id": vuln.PrimaryCVEID,
		"matched_on":     vuln.MatchedOn,
		"match_mode":     vuln.MatchMode,
		"confidence":     vuln.Confidence,
		"scanner":        vuln.Scanner,
		"vuln_url":       vuln.VulnURL,
	})
	evidence, err := upsertPhase1Evidence(tx, models.SecurityScanEvidence{
		RunID:        ctx.runID,
		TaskID:       ctx.taskID,
		TargetID:     targetID,
		EvidenceType: "version-match",
		SourceEngine: "vuln-matcher",
		Digest:       phase1Digest("version-match", vuln.IP, strconv.Itoa(vuln.Port), vuln.CVEID, vuln.PrimaryCVEID, vuln.MatchMode, vuln.MatchedOn),
		PayloadExcerpt: phase1TrimText(
			firstNonEmpty(vuln.MatchedOn, vuln.Title),
			2048,
		),
		MetadataJSON: raw,
		RawJSON:      raw,
	})
	if err != nil {
		return err
	}

	now := time.Now()
	legacyID := vuln.ID
	return upsertPhase1Occurrence(tx, models.SecurityScanFindingOccurrence{
		RunID:                 ctx.runID,
		TaskID:                ctx.taskID,
		TargetID:              targetID,
		LegacyVulnerabilityID: &legacyID,
		FindingKey:            phase1FindingKey(vuln),
		FindingFamily:         vuln.FindingFamily,
		FindingSource:         vuln.FindingSource,
		Severity:              vuln.Severity,
		Confidence:            vuln.Confidence,
		MatchMode:             vuln.MatchMode,
		PrimaryCVEID:          vuln.PrimaryCVEID,
		VulnDBID:              vuln.VulnDBID,
		Title:                 vuln.Title,
		Status:                firstNonEmpty(strings.TrimSpace(vuln.Status), "open"),
		VerificationStatus:    phase1VerificationStatus(vuln),
		EvidenceCount:         1,
		FirstSeenAt:           &now,
		LastSeenAt:            &now,
		MetadataJSON: phase1JSONString(map[string]interface{}{
			"asset_id":     vuln.AssetID,
			"matched_on":   vuln.MatchedOn,
			"scanner":      vuln.Scanner,
			"evidence_id":  evidence.ID,
			"priority":     vuln.Priority,
			"vuln_url":     vuln.VulnURL,
			"source_vuln":  vuln.SourceVulnID,
			"confirmed_id": vuln.ConfirmedVulnID,
		}),
	})
}

func recordNucleiFindingOccurrence(tx *gorm.DB, ctx *hostScanPhase1Context, portID int, vuln models.SecurityVulnerability, result NucleiResult) error {
	if ctx == nil {
		return nil
	}

	targetID := ctx.targetIDForPort(portID)
	raw := phase1JSONString(result)
	evidenceMeta := phase1JSONString(map[string]interface{}{
		"matched_at": result.MatchedAt,
		"url":        result.URL,
		"host":       result.Host,
		"port":       result.Port,
		"template":   result.TemplateID,
		"severity":   result.Severity,
		"cves":       result.CVEs,
		"tags":       result.Info.Tags,
	})
	evidence, err := upsertPhase1Evidence(tx, models.SecurityScanEvidence{
		RunID:           ctx.runID,
		TaskID:          ctx.taskID,
		TargetID:        targetID,
		EvidenceType:    "nuclei-result",
		SourceEngine:    "nuclei",
		Digest:          phase1Digest("nuclei-result", vuln.IP, strconv.Itoa(vuln.Port), result.TemplateID, result.MatchedAt, vuln.PrimaryCVEID, vuln.Title),
		RequestExcerpt:  phase1TrimText(result.Request, 8192),
		ResponseExcerpt: phase1TrimText(result.Response, 8192),
		PayloadExcerpt:  phase1TrimText(firstNonEmpty(result.URL, result.MatchedAt), 2048),
		MetadataJSON:    evidenceMeta,
		RawJSON:         raw,
	})
	if err != nil {
		return err
	}

	now := time.Now()
	legacyID := vuln.ID
	return upsertPhase1Occurrence(tx, models.SecurityScanFindingOccurrence{
		RunID:                 ctx.runID,
		TaskID:                ctx.taskID,
		TargetID:              targetID,
		LegacyVulnerabilityID: &legacyID,
		FindingKey:            phase1FindingKey(vuln),
		FindingFamily:         vuln.FindingFamily,
		FindingSource:         vuln.FindingSource,
		Severity:              vuln.Severity,
		Confidence:            vuln.Confidence,
		MatchMode:             vuln.MatchMode,
		PrimaryCVEID:          vuln.PrimaryCVEID,
		VulnDBID:              vuln.VulnDBID,
		Title:                 vuln.Title,
		Status:                firstNonEmpty(strings.TrimSpace(vuln.Status), "open"),
		VerificationStatus:    phase1VerificationStatus(vuln),
		EvidenceCount:         1,
		FirstSeenAt:           &now,
		LastSeenAt:            &now,
		MetadataJSON: phase1JSONString(map[string]interface{}{
			"asset_id":     vuln.AssetID,
			"template_id":  vuln.TemplateID,
			"scanner":      vuln.Scanner,
			"scan_method":  vuln.ScanMethod,
			"vuln_url":     vuln.VulnURL,
			"evidence_id":  evidence.ID,
			"source_vuln":  vuln.SourceVulnID,
			"confirmed_id": vuln.ConfirmedVulnID,
		}),
	})
}

func initializePhase1RunFields(tx *gorm.DB, task models.SecurityScanTask, runID uint) error {
	support := getScanPhase1SchemaSupport(tx)
	if !support.runExtensions {
		return nil
	}

	var maxRunNo int
	if err := tx.Raw(
		"SELECT COALESCE(MAX(run_no), 0) FROM security_scan_runs WHERE task_id = ? AND id <> ?",
		task.ID,
		runID,
	).Scan(&maxRunNo).Error; err != nil {
		return err
	}

	updates := map[string]interface{}{
		"run_no":          maxRunNo + 1,
		"phase":           "prepare",
		"config_snapshot": phase1RunConfigSnapshot(task),
	}
	return tx.Table("security_scan_runs").Where("id = ?", runID).Updates(updates).Error
}
