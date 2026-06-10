// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package monitor

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jenvenson/ops-platform/pkg/config"
)

// GrafanaProxy Grafana API 代理
type GrafanaProxy struct {
	URL      string
	Username string
	Password string
	Client   *http.Client
}

// NewGrafanaProxy 创建 Grafana 代理
func NewGrafanaProxy(cfg config.GrafanaConfig) *GrafanaProxy {
	return &GrafanaProxy{
		URL:      cfg.URL,
		Username: cfg.Username,
		Password: cfg.Password,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// getAuthHeader 生成 Basic Auth header
func (g *GrafanaProxy) getAuthHeader() string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(g.Username+":"+g.Password))
}

// proxyRequest 代理请求到 Grafana
func (g *GrafanaProxy) proxyRequest(c *gin.Context, path string) {
	if strings.TrimSpace(g.URL) == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Grafana 未配置"})
		return
	}

	url := g.URL + path

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建请求失败"})
		return
	}

	if strings.TrimSpace(g.Username) != "" || strings.TrimSpace(g.Password) != "" {
		req.Header.Set("Authorization", g.getAuthHeader())
	}
	req.Header.Set("Accept", "application/json")

	resp, err := g.Client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("连接 Grafana 失败: %v", err)})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取响应失败"})
		return
	}

	// Grafana 上游鉴权失败不应被前端误判成平台 JWT 失效。
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		payload := gin.H{
			"error":           "Grafana 认证失败，请检查监控代理配置",
			"upstream_status": resp.StatusCode,
		}

		var upstreamJSON map[string]any
		if json.Unmarshal(body, &upstreamJSON) == nil && len(upstreamJSON) > 0 {
			payload["upstream_body"] = upstreamJSON
		} else if message := strings.TrimSpace(string(body)); message != "" {
			payload["upstream_body"] = message
		}

		c.JSON(http.StatusBadGateway, payload)
		return
	}

	// 设置响应头
	c.Header("Content-Type", resp.Header.Get("Content-Type"))
	c.Status(resp.StatusCode)
	c.Writer.Write(body)
}

// HealthHandler 获取 Grafana 健康状态
func (g *GrafanaProxy) HealthHandler(c *gin.Context) {
	g.proxyRequest(c, "/api/health")
}

// DashboardsHandler 获取仪表盘列表
func (g *GrafanaProxy) DashboardsHandler(c *gin.Context) {
	g.proxyRequest(c, "/api/search?type=dash-db")
}

// DashboardDetailHandler 获取仪表盘详情
func (g *GrafanaProxy) DashboardDetailHandler(c *gin.Context) {
	uid := c.Param("uid")
	g.proxyRequest(c, "/api/dashboards/uid/"+uid)
}

// DatasourcesHandler 获取数据源列表
func (g *GrafanaProxy) DatasourcesHandler(c *gin.Context) {
	g.proxyRequest(c, "/api/datasources")
}

// AlertsHandler 获取告警规则
func (g *GrafanaProxy) AlertsHandler(c *gin.Context) {
	g.proxyRequest(c, "/api/alerts")
}

// AlertRulesHandler 获取 Unified Alerting 规则
func (g *GrafanaProxy) AlertRulesHandler(c *gin.Context) {
	g.proxyRequest(c, "/api/ruler/grafana/api/v1/rules")
}

// PrometheusRulesHandler 获取 Prometheus 告警规则（通过 Grafana datasource proxy）
func (g *GrafanaProxy) PrometheusRulesHandler(c *gin.Context) {
	g.proxyRequest(c, "/api/datasources/proxy/1/api/v1/rules")
}

// PrometheusQueryHandler 代理 Prometheus 查询
func (g *GrafanaProxy) PrometheusQueryHandler(c *gin.Context) {
	datasourceUID := c.DefaultQuery("datasource", "")
	expr := c.Query("expr")

	if expr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "expr 参数不能为空"})
		return
	}

	// 使用 url.Values 正确编码查询参数（处理 {}[]() 等特殊字符）
	params := url.Values{}
	params.Set("query", expr)

	// 通过 Grafana 的 datasource proxy 查询 Prometheus
	path := fmt.Sprintf("/api/datasources/proxy/1/api/v1/query?%s", params.Encode())
	if datasourceUID != "" {
		path = fmt.Sprintf("/api/datasources/uid/%s/resources/api/v1/query?%s", datasourceUID, params.Encode())
	}

	g.proxyRequest(c, path)
}

// PrometheusQueryRangeHandler 代理 Prometheus 范围查询
func (g *GrafanaProxy) PrometheusQueryRangeHandler(c *gin.Context) {
	expr := c.Query("expr")
	start := c.DefaultQuery("start", "")
	end := c.DefaultQuery("end", "")
	step := c.DefaultQuery("step", "60")

	if expr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "expr 参数不能为空"})
		return
	}

	// 使用 url.Values 正确编码查询参数
	params := url.Values{}
	params.Set("query", expr)
	params.Set("start", start)
	params.Set("end", end)
	params.Set("step", step)

	path := fmt.Sprintf("/api/datasources/proxy/1/api/v1/query_range?%s", params.Encode())

	g.proxyRequest(c, path)
}

// GrafanaURLHandler 返回 Grafana 地址（用于前端跳转）
func (g *GrafanaProxy) GrafanaURLHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"url":      g.URL,
		"username": g.Username,
	})
}