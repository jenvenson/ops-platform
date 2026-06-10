// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package security

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/jenvenson/ops-platform/internal/models"
)

// ============================================
// 模块四：Web漏洞扫描层 (Web Vulnerability Scanner Module)
// 职责：针对 HTTP/HTTPS 服务进行 Web 漏洞检测
// ============================================

// NucleiWebScanConfig Nuclei Web 扫描配置（与 handler.WebScanConfig 分开定义）
type NucleiWebScanConfig struct {
	Target     string   // 目标 URL
	Options    []string // 扫描选项（标签过滤）
	AuthMode   string   // 认证模式: none, cookie, bearer, basic
	Credential string   // 凭据
	AuthHeader string   // 自定义认证头
	Timeout    time.Duration
	Concurrent int // 并发数
}

// WebVulnResult Web 漏洞扫描结果
type WebVulnResult struct {
	TemplateID  string   // Nuclei 模板 ID
	URL         string   // 漏洞所在 URL
	Host        string   // 主机名
	Port        int      // 端口
	CVEID       string   // CVE 编号
	CVSS        float64  // CVSS 分数
	Severity    string   // 严重程度
	Title       string   // 漏洞标题
	Description string   // 漏洞描述
	Solution    string   // 解决方案
	Matched     string   // 匹配的内容
	Request     string   // 请求片段
	Response    string   // 响应片段
	Tags        []string // 标签
}

// WebScanner Web 漏洞扫描器
type WebScanner struct{}

// NewWebScanner 创建 Web 扫描器
func NewWebScanner() *WebScanner {
	return &WebScanner{}
}

// Execute 执行 Web 漏洞扫描
func (s *WebScanner) Execute(config *NucleiWebScanConfig) ([]WebVulnResult, error) {
	var results []WebVulnResult

	// 构建 Nuclei 命令
	args := s.buildNucleiArgs(config)

	cmd := exec.Command("nuclei", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("Nuclei scan stderr: %s\n", stderr.String())
		// Nuclei 可能因为没有漏洞而返回非 0，仍然继续解析结果
	}

	// 解析 JSON 结果
	results = s.parseNucleiOutput(stdout.String(), config.Target)

	fmt.Printf("Web scan results for %s: %d vulnerabilities found\n", config.Target, len(results))

	return results, nil
}

// buildNucleiArgs 构建 Nuclei 命令参数
func (s *WebScanner) buildNucleiArgs(config *NucleiWebScanConfig) []string {
	args := []string{
		"-u", config.Target,
		"-json",
		"-c", "25",
		"-silent",
		"-timeout", "10s",
	}

	// 添加扫描选项（标签过滤）
	if len(config.Options) > 0 {
		var tags []string
		for _, opt := range config.Options {
			if tag, ok := NucleiTags[opt]; ok {
				tags = append(tags, tag)
			}
		}
		if len(tags) > 0 {
			args = append(args, "-tags", strings.Join(tags, ","))
		}
	}

	// 添加认证头
	if config.AuthMode != "" && config.AuthMode != "none" && config.Credential != "" {
		var headerValue string
		switch config.AuthMode {
		case "cookie":
			headerValue = config.Credential
		case "bearer":
			if !strings.HasPrefix(config.Credential, "Bearer ") && !strings.HasPrefix(config.Credential, "bearer ") {
				headerValue = "Bearer " + config.Credential
			} else {
				headerValue = config.Credential
			}
		case "basic":
			headerValue = "Basic " + config.Credential
		}

		if headerValue != "" {
			headerName := config.AuthHeader
			if headerName == "" {
				headerName = "Authorization"
			}
			args = append(args, "-H", headerName+": "+headerValue)
		}
	}

	return args
}

// parseNucleiOutput 解析 Nuclei JSON 输出
func (s *WebScanner) parseNucleiOutput(output, targetURL string) []WebVulnResult {
	var results []WebVulnResult

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "{") {
			continue
		}

		var nucleiResult NucleiResult
		if err := json.Unmarshal([]byte(line), &nucleiResult); err != nil {
			continue
		}

		// 提取 CVE ID
		var cveID string
		for _, ref := range nucleiResult.Info.Reference {
			if strings.HasPrefix(ref, "CVE-") {
				cveID = ref
				break
			}
		}

		webResult := WebVulnResult{
			TemplateID:  nucleiResult.TemplateID,
			URL:         targetURL,
			Host:        nucleiResult.Host,
			Port:        nucleiResult.Port,
			CVEID:       cveID,
			CVSS:        nucleiResult.CVSSScore,
			Severity:    strings.ToLower(nucleiResult.Severity),
			Title:       nucleiResult.Info.Name,
			Description: nucleiResult.Info.Description,
			Solution:    nucleiResult.Info.Solution,
			Matched:     nucleiResult.MatchedAt,
			Request:     nucleiResult.Request,
			Response:    nucleiResult.Response,
			Tags:        nucleiResult.Info.Tags,
		}

		results = append(results, webResult)
	}

	return results
}

// ExecuteMultiple 对多个 URL 执行扫描
func (s *WebScanner) ExecuteMultiple(urls []string, config *NucleiWebScanConfig) ([]WebVulnResult, error) {
	var allResults []WebVulnResult

	for i, url := range urls {
		config.Target = url
		results, err := s.Execute(config)
		if err != nil {
			fmt.Printf("Web scan failed for %s: %v\n", url, err)
			continue
		}
		allResults = append(allResults, results...)
		fmt.Printf("Progress: %d/%d\n", i+1, len(urls))
	}

	return allResults, nil
}

// ScanWebPorts 对发现的 Web 端口执行扫描
func (s *WebScanner) ScanWebPorts(ip string, webPorts []PortInfo, options []string) ([]WebVulnResult, error) {
	var allResults []WebVulnResult

	for _, port := range webPorts {
		var protocol string
		if port.PortID == 443 || (port.Product != "" && strings.Contains(strings.ToLower(port.Product), "ssl")) {
			protocol = "https"
		} else {
			protocol = "http"
		}

		url := fmt.Sprintf("%s://%s:%d", protocol, ip, port.PortID)

		config := &NucleiWebScanConfig{
			Target:  url,
			Options: options,
		}

		results, err := s.Execute(config)
		if err != nil {
			fmt.Printf("Web scan failed for %s: %v\n", url, err)
			continue
		}

		allResults = append(allResults, results...)
	}

	return allResults, nil
}

// ToVulnerability 转换为数据库模型
func (r *WebVulnResult) ToVulnerability(taskID, assetID uint) models.SecurityVulnerability {
	// 确定优先级
	priority := "优先级低"
	if r.Severity == "critical" || r.Severity == "high" {
		priority = "优先级高"
	} else if r.Severity == "medium" {
		priority = "优先级中"
	}

	// 获取漏洞类型
	vulnType := extractVulnType(r.Tags, "")

	// 确定扫描方式
	scanMethod := "非授权扫描"
	for _, tag := range r.Tags {
		if tag == "active" || tag == "exploit" {
			scanMethod = "主动探测"
			break
		}
	}

	vuln := models.SecurityVulnerability{
		AssetID:       assetID,
		IP:            r.Host,
		Port:          r.Port,
		Protocol:      "tcp",
		Severity:      r.Severity,
		CVSSScore:     r.CVSS,
		CVEID:         r.CVEID,
		Title:         r.Title,
		Description:   r.Description,
		VulnType:      vulnType,
		Solution:      r.Solution,
		Scanner:       "nuclei",
		TemplateID:    r.TemplateID,
		ScanMethod:    scanMethod,
		VulnURL:       r.URL,
		Payload:       r.Matched,
		Request:       r.Request,
		Response:      r.Response,
		Priority:      priority,
		FalsePositive: false,
	}
	assignVulnerabilityTaskTracking(&vuln, taskID)
	return vuln
}