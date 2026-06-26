// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package security

import (
	"sync"
)

type ScanJob struct {
	TaskID     uint
	Target     string
	TargetType string
	ScanType   string
	WebConfig  *WebScanConfig
}

var (
	scanQueue  chan ScanJob
	queueOnce  sync.Once
	maxWorkers = 2 // 限制并发扫描任务数为 2，可根据实际系统负载调整
)

// InitScanQueue 初始化任务队列和工作协程
func InitScanQueue() {
	queueOnce.Do(func() {
		scanQueue = make(chan ScanJob, 100) // 队列缓冲区大小
		for i := 0; i < maxWorkers; i++ {
			go scanWorker()
		}
	})
}

// EnqueueScan 将扫描任务加入队列
func EnqueueScan(job ScanJob) {
	InitScanQueue()
	scanQueue <- job
}

// scanWorker 从队列中消费任务并执行
func scanWorker() {
	for job := range scanQueue {
		runScan(job.TaskID, job.Target, job.TargetType, job.ScanType, job.WebConfig)
	}
}
