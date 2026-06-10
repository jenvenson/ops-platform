// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package monitor

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// 服务器指标
	serverOnlineCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ops_server_online_count",
		Help: "Number of online/offline servers",
	}, []string{"status"})

	// 部署指标
	deployCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ops_deploy_total",
		Help: "Total number of deployments",
	}, []string{"app", "env", "status"})

	// HTTP 请求指标
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ops_http_requests_total",
		Help: "Total HTTP requests",
	}, []string{"method", "path", "status"})

	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ops_http_request_duration_seconds",
		Help:    "HTTP request duration",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	// 归档指标
	archiveCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ops_archive_total",
		Help: "Total number of archives",
	}, []string{"app", "env", "deploy_type", "status"})

	// Agent 心跳指标
	agentHeartbeatTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ops_agent_heartbeat_total",
		Help: "Total agent heartbeats",
	}, []string{"server_ip", "status"})

	// 服务器资源使用率
	serverCPUUsage = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ops_server_cpu_usage_percent",
		Help: "CPU usage percentage",
	}, []string{"server_ip", "hostname"})

	serverMemoryUsage = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ops_server_memory_usage_percent",
		Help: "Memory usage percentage",
	}, []string{"server_ip", "hostname"})

	serverDiskUsage = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ops_server_disk_usage_percent",
		Help: "Disk usage percentage",
	}, []string{"server_ip", "hostname"})
)

// RecordDeploy 记录部署事件
func RecordDeploy(app, env, status string) {
	deployCounter.WithLabelValues(app, env, status).Inc()
}

// RecordArchive 记录归档事件
func RecordArchive(app, env, deployType, status string) {
	archiveCounter.WithLabelValues(app, env, deployType, status).Inc()
}

// RecordAgentHeartbeat 记录 Agent 心跳
func RecordAgentHeartbeat(serverIP, status string) {
	agentHeartbeatTotal.WithLabelValues(serverIP, status).Inc()
}

// UpdateServerResources 更新服务器资源使用率
func UpdateServerResources(serverIP, hostname string, cpu, memory, disk float64) {
	serverCPUUsage.WithLabelValues(serverIP, hostname).Set(cpu)
	serverMemoryUsage.WithLabelValues(serverIP, hostname).Set(memory)
	serverDiskUsage.WithLabelValues(serverIP, hostname).Set(disk)
}

// UpdateServerStatus 更新服务器状态计数
func UpdateServerStatus(online, offline int) {
	serverOnlineCount.WithLabelValues("online").Set(float64(online))
	serverOnlineCount.WithLabelValues("offline").Set(float64(offline))
}