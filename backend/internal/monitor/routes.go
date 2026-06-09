package monitor

import (
	"net/http"

	"github.com/jenvenson/ops-platform/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/jenvenson/ops-platform/internal/cmdb"
	"github.com/jenvenson/ops-platform/internal/database"
	"github.com/jenvenson/ops-platform/pkg/config"
)

// RegisterRoutes 注册监控路由
func RegisterRoutes(r *gin.Engine, cfg *config.Config) {
	monitor := r.Group("/api/monitor")
	monitor.Use(auth.AuthMiddleware(cfg.JWT.Secret))
	{
		// 获取服务器状态（缓存）
		monitor.GET("/servers", GetServersStatusHandler)

		// 实时 ping 检测所有服务器（必须放在 :id 路由之前）
		monitor.GET("/servers/ping", PingAllServersHandler)

		// 实时 ping 检测单个服务器
		monitor.GET("/servers/:id/ping", PingServerHandler)

		// 手动触发后台健康检查（异步更新数据库）
		monitor.POST("/check", TriggerCheckHandler)
	}

	// Grafana 代理路由
	if cfg.Grafana.URL != "" {
		proxy := NewGrafanaProxy(cfg.Grafana)
		grafana := r.Group("/api/grafana")
		grafana.Use(auth.AuthMiddleware(cfg.JWT.Secret))
		{
			grafana.GET("/health", proxy.HealthHandler)
			grafana.GET("/dashboards", proxy.DashboardsHandler)
			grafana.GET("/dashboards/:uid", proxy.DashboardDetailHandler)
			grafana.GET("/datasources", proxy.DatasourcesHandler)
			grafana.GET("/alerts", proxy.AlertsHandler)
			grafana.GET("/alert-rules", proxy.AlertRulesHandler)
			grafana.GET("/prometheus-rules", proxy.PrometheusRulesHandler)
			grafana.GET("/query", proxy.PrometheusQueryHandler)
			grafana.GET("/query_range", proxy.PrometheusQueryRangeHandler)
			grafana.GET("/url", proxy.GrafanaURLHandler)
		}
	}
}

// GetServersStatusHandler 获取所有服务器状态（缓存）
func GetServersStatusHandler(c *gin.Context) {
	statuses := GetServerStatus()
	c.JSON(http.StatusOK, gin.H{"data": statuses})
}

// PingAllServersHandler 实时 ping 检测所有服务器
func PingAllServersHandler(c *gin.Context) {
	results := CheckAllServers()
	c.JSON(http.StatusOK, gin.H{"data": results})
}

// PingServerHandler 实时 ping 检测单个服务器
func PingServerHandler(c *gin.Context) {
	id := c.Param("id")

	var server cmdb.Server
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", id).First(&server).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}

	online, latency, err := CheckServerOnline(server.IP, server.SSHPort)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"server_id":  server.ID,
			"hostname":   server.Hostname,
			"ip":         server.IP,
			"online":     false,
			"latency_ms": 0,
			"error":      err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"server_id":  server.ID,
		"hostname":   server.Hostname,
		"ip":         server.IP,
		"online":     online,
		"latency_ms": latency,
	})
}

// TriggerCheckHandler 手动触发健康检查（异步更新数据库）
func TriggerCheckHandler(c *gin.Context) {
	go CheckAllServers()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "健康检查已开始",
	})
}
