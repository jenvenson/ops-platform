// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package security

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jenvenson/ops-platform/internal/models"
)

// SchedulerConfig 调度器配置
type SchedulerConfig struct {
	EnableNVDCallback   bool          // 启用 NVD 自动同步
	NVDCallbackInterval time.Duration // NVD 同步间隔（默认每天一次）
	EnableCNVDSync      bool          // 启用 CNVD 同步
	CNVDSyncInterval    time.Duration // CNVD 同步间隔（默认每周一次）
	SyncTimeOfDay       string        // 同步时间 (HH:MM 格式)
}

// DefaultSchedulerConfig 默认配置
var DefaultSchedulerConfig = SchedulerConfig{
	EnableNVDCallback:   false,
	NVDCallbackInterval: 24 * time.Hour,
	EnableCNVDSync:      false,
	CNVDSyncInterval:    7 * 24 * time.Hour,
	SyncTimeOfDay:       "02:00", // 凌晨2点执行
}

// Scheduler 定时任务调度器
type Scheduler struct {
	config    SchedulerConfig
	vulnDB    *VulnDBService
	isRunning bool
	stopChan  chan struct{}
	wg        sync.WaitGroup
	mu        sync.Mutex
}

// NewScheduler 创建调度器
func NewScheduler(config SchedulerConfig) *Scheduler {
	if config.NVDCallbackInterval == 0 {
		config.NVDCallbackInterval = DefaultSchedulerConfig.NVDCallbackInterval
	}
	if config.CNVDSyncInterval == 0 {
		config.CNVDSyncInterval = DefaultSchedulerConfig.CNVDSyncInterval
	}
	if config.SyncTimeOfDay == "" {
		config.SyncTimeOfDay = DefaultSchedulerConfig.SyncTimeOfDay
	}

	return &Scheduler{
		config:   config,
		vulnDB:   NewVulnDBService(),
		stopChan: make(chan struct{}),
	}
}

// Start 启动调度器
func (s *Scheduler) Start() {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		log.Println("[Scheduler] Scheduler already running")
		return
	}
	s.isRunning = true
	s.mu.Unlock()

	log.Println("[Scheduler] Starting vulnerability sync scheduler...")
	if !s.config.EnableNVDCallback && !s.config.EnableCNVDSync {
		log.Println("[Scheduler] Vulnerability sync is disabled")
		return
	}

	// 初始化漏洞库
	if err := s.vulnDB.InitVulnDB(); err != nil {
		log.Printf("[Scheduler] Failed to initialize vulnerability database: %v", err)
	}

	s.wg.Add(1)
	go s.runNVDCallbackSync()

	s.wg.Add(1)
	go s.runCNVDSync()

	log.Println("[Scheduler] Scheduler started successfully")
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.isRunning {
		s.mu.Unlock()
		return
	}
	s.isRunning = false
	s.mu.Unlock()

	log.Println("[Scheduler] Stopping scheduler...")
	close(s.stopChan)
	s.wg.Wait()
	log.Println("[Scheduler] Scheduler stopped")
}

// runNVDCallbackSync 运行 NVD 定时同步
func (s *Scheduler) runNVDCallbackSync() {
	defer s.wg.Done()

	if !s.config.EnableNVDCallback {
		log.Println("[Scheduler] NVD callback sync is disabled")
		return
	}

	ticker := time.NewTicker(s.config.NVDCallbackInterval)
	defer ticker.Stop()

	// 首次执行：检查是否需要全量同步
	s.performNVDSync()

	for {
		select {
		case <-ticker.C:
			s.performNVDSync()
		case <-s.stopChan:
			return
		}
	}
}

// runCNVDSync 运行 CNVD 定时同步
func (s *Scheduler) runCNVDSync() {
	defer s.wg.Done()

	if !s.config.EnableCNVDSync {
		log.Println("[Scheduler] CNVD sync is disabled")
		return
	}

	ticker := time.NewTicker(s.config.CNVDSyncInterval)
	defer ticker.Stop()

	// 首次执行
	s.performCNVDSync()

	for {
		select {
		case <-ticker.C:
			s.performCNVDSync()
		case <-s.stopChan:
			return
		}
	}
}

// performNVDSync 执行 NVD 同步
func (s *Scheduler) performNVDSync() {
	log.Println("[Scheduler] Starting NVD sync...")

	// 获取上次同步时间
	lastSync := s.vulnDB.GetLastSyncTime()
	var daysSinceLastSync int
	if lastSync != nil {
		daysSinceLastSync = int(time.Since(*lastSync).Hours() / 24)
	} else {
		daysSinceLastSync = 30 // 首次同步最近30天
	}

	// 如果超过30天没有同步，执行全量同步
	if daysSinceLastSync > 30 {
		log.Printf("[Scheduler] Full sync needed (last sync: %d days ago)", daysSinceLastSync)
		count, err := s.vulnDB.FullSyncNVD()
		if err != nil {
			log.Printf("[Scheduler] NVD full sync failed: %v", err)
			return
		}
		log.Printf("[Scheduler] NVD full sync completed: %d vulnerabilities imported", count)
	} else {
		// 增量同步最近的数据
		log.Printf("[Scheduler] Incremental sync (last sync: %d days ago)", daysSinceLastSync)
		count, err := s.vulnDB.SyncRecentNVD(daysSinceLastSync + 1)
		if err != nil {
			log.Printf("[Scheduler] NVD incremental sync failed: %v", err)
			return
		}
		log.Printf("[Scheduler] NVD incremental sync completed: %d vulnerabilities imported", count)
	}

	// 记录同步统计
	s.logSyncStats()
}

// performCNVDSync 执行 CNVD 同步
func (s *Scheduler) performCNVDSync() {
	log.Println("[Scheduler] Starting CNVD sync...")

	// CNVD API 需要单独实现
	// 这里暂时跳过，等 CNVD API 实现后补充
	log.Println("[Scheduler] CNVD sync: API not implemented yet")
}

// logSyncStats 记录同步统计
func (s *Scheduler) logSyncStats() {
	stats, err := s.vulnDB.GetStats()
	if err != nil {
		log.Printf("[Scheduler] Failed to get stats: %v", err)
		return
	}

	log.Printf("[Scheduler] Vulnerability database stats:")
	log.Printf("  Total: %d", stats["total"])
	log.Printf("  Critical: %d", stats["critical"])
	log.Printf("  High: %d", stats["high"])
	log.Printf("  Medium: %d", stats["medium"])
	log.Printf("  Low: %d", stats["low"])
	log.Printf("  This week: %d", stats["this_week"])
}

// GetSchedulerStatus 获取调度器状态
func (s *Scheduler) GetSchedulerStatus() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	stats, _ := s.vulnDB.GetStats()
	lastSync := s.vulnDB.GetLastSyncTime()

	status := map[string]interface{}{
		"is_running":       s.isRunning,
		"config":           s.config,
		"vuln_stats":       stats,
		"last_sync_time":   lastSync,
		"last_sync_format": "unknown",
	}

	if lastSync != nil {
		status["last_sync_format"] = lastSync.Format("2006-01-02 15:04:05")
	}

	return status
}

// ManualSync 手动触发同步
func (s *Scheduler) ManualSync(source string) (string, error) {
	switch source {
	case "nvd":
		go s.performNVDSync()
		return "NVD sync started", nil
	case "cnvd":
		go s.performCNVDSync()
		return "CNVD sync started", nil
	case "all":
		go s.performNVDSync()
		go s.performCNVDSync()
		return "All sync started", nil
	default:
		return "", fmt.Errorf("unknown sync source: %s", source)
	}
}

// VulnScheduler 全局漏洞同步调度器实例
var VulnScheduler *Scheduler

// StartVulnScheduler 启动漏洞同步调度器
func StartVulnScheduler() {
	if VulnScheduler != nil {
		log.Println("[Scheduler] Scheduler already started")
		return
	}

	VulnScheduler = NewScheduler(DefaultSchedulerConfig)
	if !VulnScheduler.config.EnableNVDCallback && !VulnScheduler.config.EnableCNVDSync {
		log.Println("[Scheduler] Vulnerability sync feature disabled; scheduler will not start")
		VulnScheduler = nil
		return
	}
	VulnScheduler.Start()
}

// StopVulnScheduler 停止漏洞同步调度器
func StopVulnScheduler() {
	if VulnScheduler != nil {
		VulnScheduler.Stop()
		VulnScheduler = nil
	}
}

// GetVulnScheduler 获取调度器实例
func GetVulnScheduler() *Scheduler {
	return VulnScheduler
}

// SyncTaskResponse 同步任务响应
type SyncTaskResponse struct {
	ID           uint   `json:"id"`
	Source       string `json:"source"`
	Status       string `json:"status"`
	TotalCount   int    `json:"total_count"`
	SuccessCount int    `json:"success_count"`
	FailCount    int    `json:"fail_count"`
	StartTime    string `json:"start_time"`
	EndTime      string `json:"end_time"`
	ErrorMessage string `json:"error_message"`
}

// GetSyncTasks 获取同步任务历史
func (s *VulnDBService) GetSyncTaskHistory(limit int) ([]SyncTaskResponse, error) {
	tasks, err := s.GetSyncTasks(limit)
	if err != nil {
		return nil, err
	}

	var responses []SyncTaskResponse
	for _, task := range tasks {
		responses = append(responses, SyncTaskResponse{
			ID:           task.ID,
			Source:       task.Source,
			Status:       task.Status,
			TotalCount:   task.TotalCount,
			SuccessCount: task.SuccessCount,
			FailCount:    task.FailCount,
			StartTime:    task.StartTime.Format("2006-01-02 15:04:05"),
			EndTime:      task.EndTime.Format("2006-01-02 15:04:05"),
			ErrorMessage: task.ErrorMessage,
		})
	}

	return responses, nil
}

// Ensure models is imported
var _ = models.VulnerabilityDatabase{}