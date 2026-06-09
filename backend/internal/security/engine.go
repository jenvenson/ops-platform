package security

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"github.com/jenvenson/ops-platform/internal/database"
	"github.com/jenvenson/ops-platform/internal/models"
)

// stripANSI 移除 ANSI 转义码和非打印字符
func stripANSI(input string) string {
	// 1. 首先解码 XML 实体编码
	// &#45; -> - (常见的 Nmap 输出中的编码连字符)
	// 十进制实体
	re := regexp.MustCompile(`&#(\d+);`)
	matches := re.FindAllStringSubmatch(input, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			if num, err := strconv.Atoi(match[1]); err == nil {
				input = strings.ReplaceAll(input, match[0], string(rune(num)))
			}
		}
	}

	// 十六进制实体 &#xHH;
	re = regexp.MustCompile(`&#x([0-9a-fA-F]+);`)
	matches = re.FindAllStringSubmatch(input, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			if num, err := strconv.ParseUint(match[1], 16, 32); err == nil {
				input = strings.ReplaceAll(input, match[0], string(rune(num)))
			}
		}
	}

	// 2. 移除 ANSI 转义序列
	// SGR (Select Graphic Rendition) 序列: ESC [ <params> m
	// 包括标准 SGR: ESC [ n m, ESC [ n ; m m
	// 以及 24 位颜色: ESC [ 38 ; 2 ; R ; G ; B m, ESC [ 48 ; 2 ; R ; G ; B m
	// 使用更宽松的模式匹配所有 ESC [ ... m 序列
	re = regexp.MustCompile(`\x1b\[[0-9;]*m`)
	input = re.ReplaceAllString(input, "")

	// 24 位颜色扩展: ESC [ 38 ; 2 ; R ; G ; B m 或 ESC [ 48 ; 2 ; R ; G ; B m
	re = regexp.MustCompile(`\x1b\[38;2;[0-9;]*m`)
	input = re.ReplaceAllString(input, "")
	re = regexp.MustCompile(`\x1b\[48;2;[0-9;]*m`)
	input = re.ReplaceAllString(input, "")

	// 其他 ANSI 控制序列: ESC [ <letters>
	re = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	input = re.ReplaceAllString(input, "")

	// ESC [<letter> 格式 (如 ESC M, ESC E)
	re = regexp.MustCompile(`\x1b[a-zA-Z]`)
	input = re.ReplaceAllString(input, "")

	// 单独的 ESC 字符
	re = regexp.MustCompile(`\x1b`)
	input = re.ReplaceAllString(input, "")

	// 3. 移除其他控制字符 (0x00-0x08, 0x0B-0x0C, 0x0E-0x1F, 0x7F)
	re = regexp.MustCompile(`[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]`)
	input = re.ReplaceAllString(input, "")

	// 4. 清理多余的空白字符
	re = regexp.MustCompile(`[ \t]+`)
	input = re.ReplaceAllString(input, " ")

	// 移除每行开头和结尾的空白
	lines := strings.Split(input, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	input = strings.Join(lines, "\n")

	return input
}

// UpdateTaskProgress 更新任务进度
func UpdateTaskProgress(taskID uint, progress int, scannedIPs int, totalIPs int, message string) {
	updates := map[string]interface{}{
		"progress":    progress,
		"scanned_ips": scannedIPs,
		"total_ips":   totalIPs,
		"message":     message,
	}
	if err := UpdateTaskAndCurrentRun(taskID, updates); err != nil {
		fmt.Printf("UpdateTaskProgress failed for task %d: %v\n", taskID, err)
	}
}

func buildPortInfosFromNmap(ports []NmapPort) []PortInfo {
	result := make([]PortInfo, 0, len(ports))
	for _, port := range ports {
		result = append(result, PortInfo{
			PortID:   port.PortID,
			Protocol: port.Protocol,
			State:    port.State,
			Service:  port.Service,
			Product:  port.Product,
			Version:  port.Version,
			CPE:      port.CPE,
			Banner:   port.Banner,
		})
	}
	return result
}

type hostServiceTarget struct {
	URL     string
	Port    int
	Service string
	Tags    []string
}

func scanProgressWeights(scanType string) (int, int) {
	switch scanType {
	case "port":
		return 90, 0
	case "host-vuln":
		return 30, 50
	case "web":
		return 30, 50
	default:
		return 25, 55
	}
}

func discoverServiceTargets(ip string, scanType string, ports []NmapPort) ([]string, []hostServiceTarget) {
	var webTargets []string
	var hostTargets []hostServiceTarget

	for _, port := range ports {
		if IsWebPort(port.PortID, port.Service) {
			protocol := "http"
			if port.PortID == 443 || port.PortID == 8443 ||
				strings.Contains(strings.ToLower(port.Service), "ssl") ||
				strings.Contains(strings.ToLower(port.Service), "https") {
				protocol = "https"
			}
			url := fmt.Sprintf("%s://%s:%d", protocol, ip, port.PortID)
			webTargets = append(webTargets, url)
			fmt.Printf("Found Web service: %s (port=%d, service=%s)\n", url, port.PortID, port.Service)
			continue
		}

		if scanType != "host-vuln" {
			continue
		}

		tags := GetServiceTags(port.Service)
		if len(tags) == 0 {
			continue
		}

		target := hostServiceTarget{
			Port:    port.PortID,
			Service: port.Service,
			Tags:    tags,
		}
		if strings.Contains(strings.ToLower(port.Service), "http") {
			protocol := "http"
			if port.PortID == 443 || strings.Contains(strings.ToLower(port.Service), "ssl") {
				protocol = "https"
			}
			target.URL = fmt.Sprintf("%s://%s:%d", protocol, ip, port.PortID)
		} else {
			target.URL = fmt.Sprintf("%s:%d", ip, port.PortID)
		}

		hostTargets = append(hostTargets, target)
		fmt.Printf("Found host service: port=%d, service=%s, tags=%v\n", port.PortID, port.Service, tags)
	}

	return webTargets, hostTargets
}

func matchHostVersionVulnerabilities(taskID uint, ip string, ports []NmapPort) (int, int, int, error) {
	matcher := NewVulnMatcher()
	versionMatches := matcher.MatchMultiplePorts(buildPortInfosFromNmap(ports))
	if len(versionMatches) == 0 {
		fmt.Printf("No version-based vulnerabilities matched on %s\n", ip)
		return 0, 0, 0, nil
	}

	high, medium, low, err := saveVersionMatchResults(taskID, ip, ports, versionMatches)
	if err != nil {
		return 0, 0, 0, err
	}
	fmt.Printf("Version vulnerability matches for %s: high=%d, medium=%d, low=%d\n", ip, high, medium, low)
	return high, medium, low, nil
}

func verifyHostServiceTargets(engine *ScanEngine, taskID uint, ip string, scannedCount int, totalIPs int, nmapProgress int, nucleiProgress int, targets []hostServiceTarget) []NucleiResult {
	results := make([]NucleiResult, 0)
	totalHostTargets := len(targets)

	for idx, target := range targets {
		progressBase := nmapProgress
		currentProgress := progressBase + int((float64(scannedCount)/float64(totalIPs))*float64(nucleiProgress)) + int((float64(idx+1)/float64(totalHostTargets))*20)
		UpdateTaskProgress(taskID, currentProgress, scannedCount, totalIPs, fmt.Sprintf("正在检测 %s 主机漏洞 (%d/%d)...", target.URL, idx+1, totalHostTargets))

		targetResults, _ := engine.ExecuteNucleiWithTags(target.URL, target.Tags)
		actionableResults := filterHostNucleiResultsByService(target.Service, targetResults)
		fmt.Printf("HOST_VULN_DEBUG summary target=%s service=%s tags=%v raw=%d actionable=%d\n", target.URL, target.Service, target.Tags, len(targetResults), len(actionableResults))

		if len(targetResults) == 0 {
			fmt.Printf("HOST_VULN_DEBUG raw_empty target=%s service=%s tags=%v\n", target.URL, target.Service, target.Tags)
		} else {
			for _, result := range targetResults {
				fmt.Printf("HOST_VULN_DEBUG raw_result target=%s service=%s %s\n", target.URL, target.Service, formatNucleiResultSummary(result))
			}
		}

		if len(targetResults) > 0 && len(actionableResults) == 0 {
			fmt.Printf("HOST_VULN_DEBUG filtered_all target=%s service=%s tags=%v\n", target.URL, target.Service, target.Tags)
		}

		for _, result := range actionableResults {
			fmt.Printf("HOST_VULN_DEBUG actionable_result target=%s service=%s %s\n", target.URL, target.Service, formatNucleiResultSummary(result))
		}

		for i := range actionableResults {
			if actionableResults[i].Host == "" {
				actionableResults[i].Host = ip
			}
			if actionableResults[i].Port == 0 {
				actionableResults[i].Port = target.Port
			}
		}

		results = append(results, actionableResults...)
	}

	return results
}

func isCandidateHostVersionMatch(vulnerability models.SecurityVulnerability) bool {
	return inferFindingSource(&vulnerability) == "host-version-match"
}

func countsTowardTaskRisk(vulnerability models.SecurityVulnerability) bool {
	if inferFindingFamily(&vulnerability) == "inventory" {
		return false
	}
	return !isCandidateHostVersionMatch(vulnerability)
}

func saveVersionMatchResults(taskID uint, host string, ports []NmapPort, matches map[int][]VulnMatchResult) (int, int, int, error) {
	if len(matches) == 0 {
		return 0, 0, 0, nil
	}

	vulnDB := NewVulnDBService()
	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	portMeta := make(map[int]NmapPort, len(ports))
	for _, port := range ports {
		portMeta[port.PortID] = port
	}

	foundationCtx, err := ensureHostScanPhase1Context(tx, taskID, host, ports)
	if err != nil {
		tx.Rollback()
		return 0, 0, 0, err
	}

	var highRisk, mediumRisk, lowRisk int

	for portID, vulnMatches := range matches {
		meta := portMeta[portID]

		var asset models.SecurityAsset
		err = tx.Where("task_id = ? AND ip = ? AND port = ?", taskID, host, portID).First(&asset).Error
		if err == gorm.ErrRecordNotFound {
			asset = models.SecurityAsset{
				TaskID:      taskID,
				IP:          host,
				Port:        portID,
				Protocol:    meta.Protocol,
				ServiceName: meta.Service,
				Version:     meta.Version,
				Banner:      meta.Banner,
			}
			if err := tx.Create(&asset).Error; err != nil {
				tx.Rollback()
				return 0, 0, 0, err
			}
		} else if err != nil {
			tx.Rollback()
			return 0, 0, 0, err
		}

		for _, match := range vulnMatches {
			vulnerability := match.ToVulnerability(host, portID, taskID, asset.ID)
			vulnerability.Protocol = meta.Protocol
			if vulnerability.MatchMode == "" {
				vulnerability.MatchMode = "fuzzy-product"
			}
			if vulnerability.Confidence == "" {
				vulnerability.Confidence = "medium"
			}
			if vulnerability.PrimaryCVEID != "" {
				if record := vulnDB.LookupCVE(vulnerability.PrimaryCVEID); record != nil {
					id := record.ID
					vulnerability.VulnDBID = &id
				}
			}

			// host-version-match 统一保留为候选线索，不计入正式风险汇总。
			if countsTowardTaskRisk(vulnerability) {
				switch vulnerability.Severity {
				case "critical", "high":
					highRisk++
				case "medium":
					mediumRisk++
				default:
					lowRisk++
				}
			}

			var existingVuln models.SecurityVulnerability
			err := tx.Where("task_id = ? AND ip = ? AND port = ? AND cve_id = ? AND scanner = ?", taskID, host, portID, vulnerability.CVEID, "vuln-matcher").First(&existingVuln).Error
			if err == gorm.ErrRecordNotFound {
				if err := tx.Create(&vulnerability).Error; err != nil {
					tx.Rollback()
					return 0, 0, 0, err
				}
				if err := recordVersionMatchOccurrence(tx, foundationCtx, portID, vulnerability); err != nil {
					tx.Rollback()
					return 0, 0, 0, err
				}
				continue
			}
			if err != nil {
				tx.Rollback()
				return 0, 0, 0, err
			}

			updates := map[string]interface{}{
				"asset_id":       asset.ID,
				"severity":       vulnerability.Severity,
				"cvss_score":     vulnerability.CVSSScore,
				"title":          vulnerability.Title,
				"description":    vulnerability.Description,
				"solution":       vulnerability.Solution,
				"vuln_type":      vulnerability.VulnType,
				"matched_on":     vulnerability.MatchedOn,
				"exploit_prereq": vulnerability.ExploitPrereq,
				"protocol":       vulnerability.Protocol,
				"primary_cve_id": vulnerability.PrimaryCVEID,
				"finding_source": vulnerability.FindingSource,
				"finding_family": vulnerability.FindingFamily,
				"confidence":     vulnerability.Confidence,
				"match_mode":     vulnerability.MatchMode,
				"priority":       vulnerability.Priority,
				"updated_at":     time.Now(),
			}
			applyVulnerabilityTaskTrackingUpdates(updates, &existingVuln, taskID)
			if vulnerability.VulnDBID != nil {
				updates["vuln_db_id"] = *vulnerability.VulnDBID
			} else {
				updates["vuln_db_id"] = nil
			}
			if err := tx.Model(&existingVuln).Updates(updates).Error; err != nil {
				tx.Rollback()
				return 0, 0, 0, err
			}

			updatedVuln := existingVuln
			updatedVuln.AssetID = asset.ID
			updatedVuln.Protocol = vulnerability.Protocol
			updatedVuln.Severity = vulnerability.Severity
			updatedVuln.CVSSScore = vulnerability.CVSSScore
			updatedVuln.Title = vulnerability.Title
			updatedVuln.Description = vulnerability.Description
			updatedVuln.Solution = vulnerability.Solution
			updatedVuln.VulnType = vulnerability.VulnType
			updatedVuln.MatchedOn = vulnerability.MatchedOn
			updatedVuln.ExploitPrereq = vulnerability.ExploitPrereq
			updatedVuln.PrimaryCVEID = vulnerability.PrimaryCVEID
			updatedVuln.FindingSource = vulnerability.FindingSource
			updatedVuln.FindingFamily = vulnerability.FindingFamily
			updatedVuln.Confidence = vulnerability.Confidence
			updatedVuln.MatchMode = vulnerability.MatchMode
			updatedVuln.Priority = vulnerability.Priority
			if vulnerability.VulnDBID != nil {
				updatedVuln.VulnDBID = vulnerability.VulnDBID
			} else {
				updatedVuln.VulnDBID = nil
			}
			if err := recordVersionMatchOccurrence(tx, foundationCtx, portID, updatedVuln); err != nil {
				tx.Rollback()
				return 0, 0, 0, err
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		return 0, 0, 0, err
	}

	return highRisk, mediumRisk, lowRisk, nil
}

func normalizeCIDRTarget(target string) (string, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", fmt.Errorf("invalid CIDR: empty target")
	}

	if !strings.Contains(target, "/") {
		ip := net.ParseIP(target)
		if ip == nil || ip.To4() == nil {
			return "", fmt.Errorf("invalid CIDR: invalid CIDR address: %s", target)
		}
		mask := net.CIDRMask(24, 32)
		network := ip.Mask(mask)
		return (&net.IPNet{IP: network, Mask: mask}).String(), nil
	}

	_, ipNet, err := net.ParseCIDR(target)
	if err != nil {
		return "", fmt.Errorf("invalid CIDR: %w", err)
	}
	return ipNet.String(), nil
}

// ParseTargetIPs 解析目标为 IP 列表
func ParseTargetIPs(target string, targetType string) ([]string, error) {
	var ips []string

	if targetType == "ip_list" {
		// 直接分割逗号分隔的 IP 列表
		items := strings.Split(target, ",")
		for _, item := range items {
			ip := strings.TrimSpace(item)
			if ip != "" {
				ips = append(ips, ip)
			}
		}
	} else if targetType == "cidr" {
		// 解析 CIDR 网段
		normalizedTarget, err := normalizeCIDRTarget(target)
		if err != nil {
			return nil, err
		}

		networkIP, ipNet, err := net.ParseCIDR(normalizedTarget)
		if err != nil {
			return nil, fmt.Errorf("invalid CIDR: %w", err)
		}

		// IPv4 CIDR 默认跳过网络地址和广播地址，避免扫描无效目标
		if ipv4 := networkIP.To4(); ipv4 != nil {
			maskSize, bits := ipNet.Mask.Size()
			if maskSize == 32 {
				ips = append(ips, ipv4.String())
				return ips, nil
			}

			base := uint64(binary.BigEndian.Uint32(ipv4))
			hostCount := uint64(1) << uint(bits-maskSize)

			startOffset := uint64(0)
			endOffset := hostCount
			if maskSize < 31 {
				startOffset = 1
				endOffset = hostCount - 1
			}

			for offset := startOffset; offset < endOffset; offset++ {
				current := make(net.IP, net.IPv4len)
				binary.BigEndian.PutUint32(current, uint32(base+offset))
				ips = append(ips, current.String())
			}
			return ips, nil
		}

		// IPv6 保持遍历所有地址，由上层目标控制范围
		for ip := append(net.IP(nil), ipNet.IP...); ipNet.Contains(ip); incrementIP(ip) {
			ips = append(ips, ip.String())
		}
	}

	return ips, nil
}

// incrementIP IP 地址递增
func incrementIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// GetTaskStatus 获取任务当前状态
func GetTaskStatus(taskID uint) string {
	var task models.SecurityScanTask
	if err := database.DB.First(&task, taskID).Error; err != nil {
		return ""
	}
	return task.Status
}

// CheckTaskStatus 检查任务状态，返回是否应该继续扫描
func CheckTaskStatus(taskID uint) (bool, string) {
	status := GetTaskStatus(taskID)
	switch status {
	case models.TaskStatusPaused:
		return false, "暂停请求已生效"
	case models.TaskStatusCancelled:
		return false, "取消请求已生效"
	case models.TaskStatusFailed:
		return false, "任务已停止"
	case models.TaskStatusCompleted:
		return false, "任务已结束"
	default:
		return true, ""
	}
}

// NmapResult Nmap 扫描结果
type NmapResult struct {
	Host     string         `json:"host"`
	Ports    []NmapPort     `json:"ports"`
	OSMatch  []NmapOSMatch  `json:"osmatch,omitempty"`
	Hostname []NmapHostname `json:"hostnames,omitempty"`
}

type NmapPort struct {
	PortID     int    `json:"portid"`
	Protocol   string `json:"protocol"`
	State      string `json:"state"`
	Service    string `json:"service"`
	Version    string `json:"version"`
	Product    string `json:"product"`
	CPE        string `json:"cpe"`
	Banner     string `json:"banner"`
	Method     string `json:"method,omitempty"`
	Confidence int    `json:"confidence,omitempty"`
}

type NmapOSMatch struct {
	Name     string  `json:"name"`
	Accuracy float64 `json:"accuracy"`
}

type NmapHostname struct {
	Name string `json:"name"`
}

// NucleiResult Nuclei 扫描结果
type NucleiResult struct {
	TemplateID   string     `json:"template-id"`
	TemplateName string     `json:"template-name,omitempty"`
	Info         NucleiInfo `json:"info,omitempty"`
	Host         string     `json:"host"`
	PortStr      string     `json:"port,omitempty"` // 原始字符串
	Port         int        `json:"-"`              // 解析后的整数值
	MatchedAt    string     `json:"matched-at"`     // Nuclei v3 使用 matched-at
	Request      string     `json:"request,omitempty"`
	Response     string     `json:"response,omitempty"`
	CVEs         []string   `json:"cve-id,omitempty"`
	Severity     string     `json:"severity"`
	CVSSScore    float64    `json:"cvss-score,omitempty"`
	Type         string     `json:"type,omitempty"` // http, network, etc.
	URL          string     `json:"url,omitempty"`
	IP           string     `json:"ip,omitempty"`
	Timestamp    string     `json:"timestamp,omitempty"`
}

// AfterJSONUnmarshal 在JSON解析后处理
func (r *NucleiResult) AfterJSONUnmarshal() {
	// 将PortStr转换为Port
	if r.PortStr != "" {
		fmt.Sscanf(r.PortStr, "%d", &r.Port)
	}
}

type NucleiInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Severity    string   `json:"severity"`
	Reference   []string `json:"reference,omitempty"`
	Solution    string   `json:"solution,omitempty"`
	Type        string   `json:"type,omitempty"` // 漏洞类型
	Tags        []string `json:"tags,omitempty"` // 标签，可能包含漏洞类型信息
}

// ScanEngine 扫描引擎
type ScanEngine struct{}

// NewScanEngine 创建扫描引擎
func NewScanEngine() *ScanEngine {
	return &ScanEngine{}
}

func nucleiCommandTimeout() time.Duration {
	raw := strings.TrimSpace(os.Getenv("OPS_NUCLEI_CMD_TIMEOUT"))
	if raw == "" {
		return 45 * time.Second
	}

	if seconds, err := strconv.Atoi(raw); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}

	if duration, err := time.ParseDuration(raw); err == nil && duration > 0 {
		return duration
	}

	return 45 * time.Second
}

func sanitizeScanURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	parsed.Fragment = ""
	return parsed.String()
}

func resolveLoginFormURL(targetURL string, config *WebScanConfig) string {
	if config == nil {
		return sanitizeScanURL(targetURL)
	}
	if strings.TrimSpace(config.LoginURL) != "" {
		return sanitizeScanURL(config.LoginURL)
	}
	return sanitizeScanURL(targetURL)
}

func extractTokenFromJSON(payload interface{}, path string) string {
	if payload == nil {
		return ""
	}
	if path != "" {
		current := payload
		for _, part := range strings.Split(path, ".") {
			obj, ok := current.(map[string]interface{})
			if !ok {
				return ""
			}
			current, ok = obj[part]
			if !ok {
				return ""
			}
		}
		switch v := current.(type) {
		case string:
			return strings.TrimSpace(v)
		}
		return ""
	}

	candidates := []string{"token", "access_token"}
	if obj, ok := payload.(map[string]interface{}); ok {
		for _, key := range candidates {
			if value, ok := obj[key].(string); ok && strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		}
		for _, nested := range []string{"data", "result"} {
			if child, ok := obj[nested]; ok {
				if token := extractTokenFromJSON(child, ""); token != "" {
					return token
				}
			}
		}
	}
	return ""
}

func performLoginFormAuth(targetURL string, config *WebScanConfig) (string, error) {
	if config == nil {
		return "", fmt.Errorf("login-form config is required")
	}
	loginURL := resolveLoginFormURL(targetURL, config)
	if loginURL == "" {
		return "", fmt.Errorf("login URL is required")
	}
	username := strings.TrimSpace(config.Username)
	password := config.Password
	if username == "" || password == "" {
		return "", fmt.Errorf("username and password are required")
	}

	method := strings.ToUpper(strings.TrimSpace(config.LoginMethod))
	if method == "" {
		method = http.MethodPost
	}
	usernameField := strings.TrimSpace(config.UsernameField)
	if usernameField == "" {
		usernameField = "username"
	}
	passwordField := strings.TrimSpace(config.PasswordField)
	if passwordField == "" {
		passwordField = "password"
	}

	form := url.Values{}
	form.Set(usernameField, username)
	form.Set(passwordField, password)

	jar, err := cookiejar.New(nil)
	if err != nil {
		return "", err
	}
	client := &http.Client{
		Timeout: 15 * time.Second,
		Jar:     jar,
	}

	req, err := http.NewRequest(method, loginURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "ops-platform-web-scan/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 8192))

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("login request returned status %d", resp.StatusCode)
	}

	parsedLoginURL, err := url.Parse(loginURL)
	if err != nil {
		return "", err
	}
	cookies := jar.Cookies(parsedLoginURL)
	if len(cookies) == 0 {
		return "", fmt.Errorf("login succeeded but no cookies were set")
	}

	var parts []string
	for _, c := range cookies {
		parts = append(parts, c.Name+"="+c.Value)
	}
	return strings.Join(parts, "; "), nil
}

func performLoginTokenAuth(targetURL string, config *WebScanConfig) (string, error) {
	if config == nil {
		return "", fmt.Errorf("login-token config is required")
	}
	loginURL := resolveLoginFormURL(targetURL, config)
	if loginURL == "" {
		return "", fmt.Errorf("login URL is required")
	}
	username := strings.TrimSpace(config.Username)
	password := config.Password
	if username == "" || password == "" {
		return "", fmt.Errorf("username and password are required")
	}

	method := strings.ToUpper(strings.TrimSpace(config.LoginMethod))
	if method == "" {
		method = http.MethodPost
	}
	usernameField := strings.TrimSpace(config.UsernameField)
	if usernameField == "" {
		usernameField = "username"
	}
	passwordField := strings.TrimSpace(config.PasswordField)
	if passwordField == "" {
		passwordField = "password"
	}
	contentType := strings.ToLower(strings.TrimSpace(config.LoginContentType))
	if contentType == "" {
		contentType = "form"
	}

	form := url.Values{}
	form.Set(usernameField, username)
	form.Set(passwordField, password)

	var body io.Reader
	requestContentType := "application/x-www-form-urlencoded"
	if contentType == "json" {
		payload, err := json.Marshal(map[string]string{
			usernameField: username,
			passwordField: password,
		})
		if err != nil {
			return "", err
		}
		body = bytes.NewReader(payload)
		requestContentType = "application/json"
	} else {
		body = strings.NewReader(form.Encode())
	}

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(method, loginURL, body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", requestContentType)
	req.Header.Set("User-Agent", "ops-platform-web-scan/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("login request returned status %d", resp.StatusCode)
	}

	var parsed interface{}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("login response is not valid JSON")
	}

	token := extractTokenFromJSON(parsed, strings.TrimSpace(config.TokenField))
	if token == "" {
		return "", fmt.Errorf("login succeeded but no token was found in response")
	}

	return token, nil
}

func resolveWebAuth(targetURL string, config *WebScanConfig) (string, string, string, error) {
	if config == nil || strings.TrimSpace(config.AuthMode) == "" || strings.TrimSpace(config.AuthMode) == "none" {
		return "", "", "", nil
	}

	if config.AuthFlow != nil || strings.TrimSpace(config.AuthMode) == "advanced" {
		resolved, err := resolveAdvancedAuth(targetURL, config)
		if err != nil {
			return "", "", "", err
		}
		if len(resolved.Headers) == 0 {
			return "", "", "", nil
		}
		parts := make([]string, 0, len(resolved.Headers))
		for _, header := range resolved.Headers {
			name := strings.TrimSpace(header.Name)
			if name == "" {
				continue
			}
			parts = append(parts, name+": "+header.Value)
		}
		return "multi-header", strings.Join(parts, "\n"), "", nil
	}

	authMode := strings.TrimSpace(config.AuthMode)
	authHeader := strings.TrimSpace(config.AuthHeader)
	credential := strings.TrimSpace(config.Credential)

	if authMode == "login-form" {
		cookieValue, err := performLoginFormAuth(targetURL, config)
		if err != nil {
			return "", "", "", err
		}
		if authHeader == "" {
			authHeader = "Cookie"
		}
		return "cookie", cookieValue, authHeader, nil
	}
	if authMode == "login-token" {
		tokenValue, err := performLoginTokenAuth(targetURL, config)
		if err != nil {
			return "", "", "", err
		}
		if authHeader == "" {
			authHeader = "token"
		}
		return "header", tokenValue, authHeader, nil
	}

	return authMode, credential, authHeader, nil
}

func buildNmapArgs(target string) []string {
	profile := strings.ToLower(strings.TrimSpace(os.Getenv("OPS_NMAP_PROFILE")))
	if profile == "" {
		profile = "full"
	}

	switch profile {
	case "balanced", "dev":
		return []string{
			"-sT", "-sV", "-Pn", "-T4",
			"--top-ports", "2000", "--open",
			"--version-light",
			"--version-intensity", "5",
			"--host-timeout", "180s",
			"-oX", "-",
			target,
		}
	default:
		return []string{
			"-sT", "-sV", "-Pn", "-T4",
			"-p-", "--open",
			"--version-all",
			"--version-intensity", "9",
			"--script", "banner",
			"--host-timeout", "300s",
			"-oX", "-",
			target,
		}
	}
}

func shouldRefineHTTPFingerprint(port NmapPort) bool {
	service := normalizeServiceName(port.Service)
	if port.PortID == 8500 {
		return true
	}
	productLower := strings.ToLower(strings.TrimSpace(port.Product))
	if (service == "http" || service == "https") && (port.PortID == 80 || port.PortID == 443) {
		// Common web ports are worth re-checking even if Nmap guessed a generic or misleading httpd product.
		if strings.TrimSpace(port.CPE) == "" || strings.TrimSpace(port.Version) == "" ||
			productLower == "" ||
			strings.Contains(productLower, "httpd") ||
			strings.Contains(productLower, "media receiver") ||
			strings.Contains(productLower, "telekom") {
			return true
		}
	}
	if (service == "http" || service == "https") && strings.TrimSpace(port.Product) == "" && strings.TrimSpace(port.Version) == "" && strings.TrimSpace(port.CPE) == "" {
		return true
	}
	if strings.HasSuffix(port.Service, "?") && (service == "http" || service == "https" || port.PortID == 443 || port.PortID == 8500) {
		return true
	}
	if service == "daap" && port.PortID == 8500 {
		return true
	}
	return false
}

func fetchHTTPFingerprint(rawURL string) (int, http.Header, []byte, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, nil, nil, err
	}
	req.Header.Set("User-Agent", "ops-platform-fingerprint/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return resp.StatusCode, resp.Header, body, nil
}

func refineHTTPFingerprint(host string, port *NmapPort) {
	if port == nil || !shouldRefineHTTPFingerprint(*port) {
		return
	}

	frontendService := "http"
	if port.PortID == 443 || normalizeServiceName(port.Service) == "https" {
		frontendService = "https"
	}

	type httpFrontendFingerprint struct {
		product string
		version string
		cpe     string
	}

	type httpBackendFingerprint struct {
		service string
		product string
		version string
		cpe     string
	}

	var frontend httpFrontendFingerprint
	var backend httpBackendFingerprint

	rootCandidates := []string{
		fmt.Sprintf("http://%s:%d/", host, port.PortID),
	}
	if frontendService == "https" {
		rootCandidates = append([]string{
			fmt.Sprintf("https://%s:%d/", host, port.PortID),
		}, rootCandidates...)
	}

	for _, candidate := range rootCandidates {
		_, headers, body, err := fetchHTTPFingerprint(candidate)
		if err != nil {
			continue
		}

		serverHeader := strings.TrimSpace(headers.Get("Server"))
		serverHeaderLower := strings.ToLower(serverHeader)
		bodyStr := strings.ToLower(string(body))

		switch {
		case strings.Contains(serverHeaderLower, "nginx"):
			frontend.product = "nginx"
			if match := regexp.MustCompile(`nginx/?([0-9][^ ]*)?`).FindStringSubmatch(serverHeaderLower); len(match) == 2 {
				frontend.version = strings.TrimSpace(match[1])
			}
			if frontend.version == "" {
				frontend.version = extractServerVersion(serverHeader)
			}
			frontend.cpe = "cpe:/a:nginx:nginx"
		case strings.Contains(serverHeaderLower, "apache"):
			frontend.product = "apache httpd"
			if match := regexp.MustCompile(`apache/?([0-9][^ ]*)?`).FindStringSubmatch(serverHeaderLower); len(match) == 2 {
				frontend.version = strings.TrimSpace(match[1])
			}
			if frontend.version == "" {
				frontend.version = extractServerVersion(serverHeader)
			}
			frontend.cpe = "cpe:/a:apache:http_server"
		case strings.Contains(serverHeaderLower, "openresty"):
			frontend.product = "openresty"
			if match := regexp.MustCompile(`openresty/?([0-9][^ ]*)?`).FindStringSubmatch(serverHeaderLower); len(match) == 2 {
				frontend.version = strings.TrimSpace(match[1])
			}
			if frontend.version == "" {
				frontend.version = extractServerVersion(serverHeader)
			}
			frontend.cpe = "cpe:/a:openresty:openresty"
		case strings.Contains(serverHeaderLower, "jetty"):
			frontend.product = "jetty"
			if match := regexp.MustCompile(`jetty(?:\(|/)?([0-9][^ )]*)?`).FindStringSubmatch(serverHeaderLower); len(match) == 2 {
				frontend.version = strings.TrimSpace(match[1])
			}
			if frontend.version == "" {
				frontend.version = extractServerVersion(serverHeader)
			}
			frontend.cpe = "cpe:/a:eclipse:jetty"
		case strings.Contains(bodyStr, "welcome to nginx"):
			frontend.product = "nginx"
			frontend.cpe = "cpe:/a:nginx:nginx"
		default:
			continue
		}

		break
	}

	apiCandidates := []string{
		fmt.Sprintf("http://%s:%d/v1/agent/self", host, port.PortID),
		fmt.Sprintf("http://%s:%d/v1/status/leader", host, port.PortID),
	}
	if frontendService == "https" {
		apiCandidates = append([]string{
			fmt.Sprintf("https://%s:%d/v1/agent/self", host, port.PortID),
			fmt.Sprintf("https://%s:%d/v1/status/leader", host, port.PortID),
		}, apiCandidates...)
	}

	for _, candidate := range apiCandidates {
		statusCode, headers, body, err := fetchHTTPFingerprint(candidate)
		if err != nil {
			continue
		}

		serverHeader := strings.ToLower(headers.Get("Server"))
		bodyStr := strings.ToLower(string(body))
		if statusCode == http.StatusOK && (strings.Contains(serverHeader, "consul") || strings.Contains(bodyStr, "\"config\"") || strings.Contains(bodyStr, "\"revision\"") || strings.Contains(bodyStr, "\"consul\"")) {
			backend.service = "consul"
			backend.product = "HashiCorp Consul"
			backend.cpe = "cpe:/a:hashicorp:consul"
			if backend.version == "" {
				versionPattern := regexp.MustCompile(`"Version"\s*:\s*"([^"]+)"`)
				if match := versionPattern.FindStringSubmatch(string(body)); len(match) == 2 {
					backend.version = match[1]
				}
			}
			break
		}
	}

	if frontend.product != "" {
		port.Service = frontendService
		port.Product = frontend.product
		if frontend.version != "" {
			port.Version = frontend.version
		}
		port.CPE = frontend.cpe
		port.Banner = buildHTTPCompositeBanner(port.Service, port.Product, port.Version, backend.service, backend.product, backend.version)
		fmt.Printf("Refined HTTP fingerprint: %s:%d -> service=%s product=%s version=%s backend=%s/%s %s\n",
			host, port.PortID, port.Service, port.Product, port.Version, backend.service, backend.product, backend.version)
		return
	}

	if backend.service != "" {
		port.Service = backend.service
		port.Product = backend.product
		if backend.version != "" {
			port.Version = backend.version
		}
		port.CPE = backend.cpe
		port.Banner = buildHTTPCompositeBanner(port.Service, port.Product, port.Version, "", "", "")
		fmt.Printf("Refined HTTP fingerprint: %s:%d -> service=%s product=%s version=%s\n", host, port.PortID, port.Service, port.Product, port.Version)
		return
	}
}

func extractServerVersion(serverHeader string) string {
	match := regexp.MustCompile(`/([0-9][^ ]*)`).FindStringSubmatch(serverHeader)
	if len(match) == 2 {
		return strings.TrimSpace(match[1])
	}
	return ""
}

func normalizeServiceName(serviceName string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(serviceName)), "?")
}

func markUnverifiedService(serviceName string, method string, confidence int, product string, version string, cpe string) string {
	serviceName = strings.TrimSpace(serviceName)
	if serviceName == "" || strings.HasSuffix(serviceName, "?") {
		return serviceName
	}
	if strings.TrimSpace(method) == "table" && confidence > 0 && confidence <= 3 &&
		strings.TrimSpace(product) == "" && strings.TrimSpace(version) == "" && strings.TrimSpace(cpe) == "" {
		return serviceName + "?"
	}
	return serviceName
}

// ExecuteNmap 执行 Nmap 扫描
// 使用纯 Nmap 进行端口扫描和服务版本检测
func (e *ScanEngine) ExecuteNmap(target string) (*NmapResult, error) {
	cmd := exec.Command("nmap", buildNmapArgs(target)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("Nmap stderr: %s\n", stderr.String())
		return nil, fmt.Errorf("nmap execution failed: %v", err)
	}

	// 解析 XML 输出
	result := &NmapResult{}
	result.Host = target

	fmt.Printf("DEBUG Nmap raw output for %s:\n%s\n", target, stdout.String())

	// 直接测试 XML 解析
	testXML := stdout.String()
	if strings.Contains(testXML, "<?xml") || strings.Contains(testXML, "<nmaprun") {
		fmt.Printf("DEBUG XML detection: found XML header\n")
	} else {
		fmt.Printf("DEBUG XML detection: NO XML header found, content starts with: %.100s\n", testXML)
	}

	parseNmapXML(testXML, result)

	// 调试：打印解析结果
	for i := range result.Ports {
		refineHTTPFingerprint(target, &result.Ports[i])
	}

	// 调试：打印解析结果
	fmt.Printf("DEBUG Parsed result: host=%s, open_ports=%d\n", result.Host, len(result.Ports))
	for _, port := range result.Ports {
		fmt.Printf("  - Port %d/%s: %s %s (state=%s)\n", port.PortID, port.Protocol, port.Service, port.Version, port.State)
	}

	return result, nil
}

// ServiceToNucleiTags 服务类型到 Nuclei 模板标签的映射
// 用于智能选择漏洞检测模板
var ServiceToNucleiTags = map[string][]string{
	// Web 服务
	"http":       {"http"},
	"https":      {"http"},
	"http-proxy": {"http"},
	"https-alt":  {"http"},

	// 数据库服务
	"mysql":         {"mysql", "database"},
	"mariadb":       {"mysql", "database"},
	"oracle":        {"oracle", "database"},
	"postgresql":    {"postgresql", "database"},
	"redis":         {"redis", "database"},
	"memcached":     {"memcached", "database"},
	"mongodb":       {"mongodb", "database"},
	"elasticsearch": {"elasticsearch", "database"},
	"etcd":          {"etcd", "database"},

	// 远程访问
	"ssh":    {"ssh", "network"},
	"rdp":    {"rdp", "network"},
	"telnet": {"telnet", "network"},
	"vnc":    {"vnc", "network"},

	// 文件服务
	"ftp": {"ftp", "network"},
	"smb": {"smb", "network"},
	"nfs": {"nfs", "network"},

	// 邮件服务
	"smtp": {"smtp", "mail"},
	"pop3": {"pop3", "mail"},
	"imap": {"imap", "mail"},

	// 其他服务
	"dns":       {"dns", "network"},
	"ldap":      {"ldap", "network"},
	"kerberos":  {"kerberos", "network"},
	"snmp":      {"snmp", "network"},
	"docker":    {"docker", "network"},
	"kube":      {"kubernetes", "network"},
	"k8s":       {"kubernetes", "network"},
	"etcd-api":  {"etcd", "network"},
	"consul":    {"consul", "network"},
	"zookeeper": {"zookeeper", "network"},
	"kafka":     {"kafka", "network"},
	"rabbitmq":  {"rabbitmq", "network"},
	"amqp":      {"rabbitmq", "network"},
}

// IsWebPort 判断端口是否为 Web 服务端口
// 根据端口和服务名综合判断
func IsWebPort(port int, serviceName string) bool {
	// 常见 Web 端口
	webPortSet := map[int]bool{
		80:   true,
		443:  true,
		8000: true,
		8008: true,
		8080: true,
		8081: true,
		8082: true,
		8083: true,
		8084: true,
		8085: true,
		8088: true,
		8090: true,
		8443: true,
		8888: true,
		9000: true,
		9001: true,
		9002: true,
		9090: true,
		3000: true,
		5000: true,
		7000: true,
		7001: true,
	}

	// 端口在已知 Web 端口列表中
	if webPortSet[port] {
		return true
	}

	// 服务名包含 http 关键字
	serviceLower := normalizeServiceName(serviceName)
	if strings.Contains(serviceLower, "http") ||
		strings.Contains(serviceLower, "ssl") && (port == 443 || port == 8443) {
		return true
	}

	return false
}

// GetServiceTags 根据服务名获取对应的 Nuclei 模板标签
func GetServiceTags(serviceName string) []string {
	serviceLower := normalizeServiceName(serviceName)

	// 精确匹配
	if tags, ok := ServiceToNucleiTags[serviceLower]; ok {
		return tags
	}

	// 模糊匹配
	for key, tags := range ServiceToNucleiTags {
		if strings.Contains(serviceLower, key) {
			return tags
		}
	}

	return nil
}

// BuildNucleiURL 根据 IP、端口和服务类型构建 Nuclei 扫描 URL
func BuildNucleiURL(ip string, port int, serviceName string) string {
	serviceLower := normalizeServiceName(serviceName)

	// 判断协议
	useHTTPS := port == 443 || port == 8443 ||
		strings.Contains(serviceLower, "ssl") ||
		strings.Contains(serviceLower, "https")

	// Web 服务使用 HTTP(S) 协议
	if IsWebPort(port, serviceName) {
		protocol := "http"
		if useHTTPS {
			protocol = "https"
		}
		return fmt.Sprintf("%s://%s:%d", protocol, ip, port)
	}

	// 非标准 Web 端口的 http 服务
	if strings.Contains(serviceLower, "http") {
		protocol := "http"
		if useHTTPS {
			protocol = "https"
		}
		return fmt.Sprintf("%s://%s:%d", protocol, ip, port)
	}

	// 其他服务，尝试构建 TCP 连接 URL（某些 Nuclei 模板支持）
	return ""
}

func containsAnyFold(input string, keywords []string) bool {
	input = strings.ToLower(strings.TrimSpace(input))
	for _, keyword := range keywords {
		if strings.Contains(input, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func isActionableHostNucleiResult(result NucleiResult) bool {
	severity := strings.ToLower(strings.TrimSpace(result.Severity))
	if severity == "critical" || severity == "high" || severity == "medium" {
		return true
	}

	if len(result.CVEs) > 0 {
		return true
	}

	name := strings.TrimSpace(result.Info.Name)
	templateID := strings.TrimSpace(result.TemplateID)
	noiseKeywords := []string{
		"detect",
		"detection",
		"enumeration",
		"fingerprint",
		"version",
		"banner",
		"info",
	}
	if containsAnyFold(name, noiseKeywords) || containsAnyFold(templateID, noiseKeywords) {
		return false
	}

	actionableTags := []string{
		"exploit",
		"rce",
		"lfi",
		"file-read",
		"unauth",
		"default-login",
		"default-creds",
		"weak-password",
		"misconfig",
		"exposure",
		"takeover",
		"auth-bypass",
	}
	for _, tag := range result.Info.Tags {
		if containsAnyFold(tag, actionableTags) {
			return true
		}
	}

	return false
}

func filterActionableHostNucleiResults(results []NucleiResult) []NucleiResult {
	filtered := make([]NucleiResult, 0, len(results))
	for _, result := range results {
		if isActionableHostNucleiResult(result) {
			filtered = append(filtered, result)
		}
	}
	return filtered
}

func isInventoryStyleHostService(service string) bool {
	switch normalizeServiceName(service) {
	case "mysql", "mariadb", "oracle", "postgresql", "redis", "memcached", "mongodb", "elasticsearch", "etcd", "consul", "zookeeper", "kafka", "rabbitmq", "amqp", "docker", "kube", "k8s":
		return true
	default:
		return false
	}
}

func isUsefulInventoryHostResult(service string, result NucleiResult) bool {
	if !isInventoryStyleHostService(service) {
		return false
	}

	templateID := strings.ToLower(strings.TrimSpace(result.TemplateID))
	name := strings.ToLower(strings.TrimSpace(result.Info.Name))
	severity := strings.ToLower(strings.TrimSpace(result.Severity))

	if severity != "" && severity != "info" {
		return false
	}

	inventoryKeywords := []string{
		"info",
		"enumeration",
		"enum",
		"fingerprint",
		"detect",
		"detection",
		"discovery",
		"version",
	}

	if containsAnyFold(templateID, inventoryKeywords) || containsAnyFold(name, inventoryKeywords) {
		return true
	}

	for _, tag := range result.Info.Tags {
		if containsAnyFold(tag, inventoryKeywords) {
			return true
		}
	}

	return false
}

func filterHostNucleiResultsByService(service string, results []NucleiResult) []NucleiResult {
	filtered := make([]NucleiResult, 0, len(results))
	for _, result := range results {
		if isActionableHostNucleiResult(result) || isUsefulInventoryHostResult(service, result) {
			filtered = append(filtered, result)
		}
	}
	return filtered
}

func inventoryServiceTypeFromResult(result NucleiResult) string {
	candidates := []string{
		strings.ToLower(strings.TrimSpace(result.TemplateID)),
		strings.ToLower(strings.TrimSpace(result.Info.Name)),
	}
	candidates = append(candidates, result.Info.Tags...)

	services := []string{"mysql", "mariadb", "oracle", "postgresql", "redis", "memcached", "mongodb", "elasticsearch", "etcd", "consul", "zookeeper", "kafka", "rabbitmq", "amqp", "docker", "kubernetes", "kube", "k8s"}
	for _, raw := range candidates {
		value := strings.ToLower(strings.TrimSpace(raw))
		for _, service := range services {
			if strings.Contains(value, service) {
				return service
			}
		}
	}
	return ""
}

func decorateInventoryResult(vuln *models.SecurityVulnerability, result NucleiResult) {
	if vuln == nil {
		return
	}

	serviceType := inventoryServiceTypeFromResult(result)
	if serviceType == "" || !isUsefulInventoryHostResult(serviceType, result) {
		return
	}

	vuln.VulnType = "资产识别"
	vuln.ScanMethod = "服务识别"
	if vuln.Description == "" {
		vuln.Description = "该结果用于识别服务版本与基础能力，不代表已确认可利用漏洞。"
	}
}

func formatNucleiResultSummary(result NucleiResult) string {
	name := strings.TrimSpace(result.Info.Name)
	if name == "" {
		name = "-"
	}

	templateID := strings.TrimSpace(result.TemplateID)
	if templateID == "" {
		templateID = "-"
	}

	severity := strings.TrimSpace(result.Severity)
	if severity == "" {
		severity = "-"
	}

	tags := "-"
	if len(result.Info.Tags) > 0 {
		tags = strings.Join(result.Info.Tags, ",")
	}

	cves := "-"
	if len(result.CVEs) > 0 {
		cves = strings.Join(result.CVEs, ",")
	}

	return fmt.Sprintf("template=%s severity=%s name=%q tags=%s cves=%s", templateID, severity, name, tags, cves)
}

// NucleiTags 扫描选项到 Nuclei 标签的映射
var NucleiTags = map[string]string{
	"sql-injection":          "sql-injection",
	"xss":                    "xss",
	"ssrf":                   "ssrf",
	"csrf":                   "csrf",
	"rce":                    "rce",
	"information-disclosure": "information-disclosure",
	"broken-access":          "broken-access-control",
	"file-inclusion":         "file-inclusion",
	"header-injection":       "header-injection",
}

// ScanTags 所有可用标签列表
var ScanTags = []string{
	"sql-injection",
	"xss",
	"ssrf",
	"csrf",
	"rce",
	"information-disclosure",
	"broken-access-control",
	"file-inclusion",
	"header-injection",
}

// ExecuteNuclei 执行 Nuclei 扫描
// target: 完整 URL，如 http://192.168.1.1:8080 或 https://192.168.1.1:443
func appendNucleiAuthArgs(args []string, targetURL string, config *WebScanConfig, session *WebSession) ([]string, error) {
	if session != nil && len(session.Headers) > 0 {
		for _, header := range session.Headers {
			name := strings.TrimSpace(header.Name)
			if name == "" {
				continue
			}
			args = append(args, "-H", name+": "+header.Value)
		}
		return args, nil
	}

	authMode, credential, authHeader, err := resolveWebAuth(targetURL, config)
	if err != nil {
		return nil, err
	}
	if authMode == "" || authMode == "none" || credential == "" {
		return args, nil
	}

	if authMode == "multi-header" {
		for _, headerLine := range strings.Split(credential, "\n") {
			headerLine = strings.TrimSpace(headerLine)
			if headerLine == "" {
				continue
			}
			args = append(args, "-H", headerLine)
		}
		return args, nil
	}

	var headerValue string
	switch authMode {
	case "cookie":
		headerValue = credential
	case "bearer":
		if !strings.HasPrefix(credential, "Bearer ") && !strings.HasPrefix(credential, "bearer ") {
			headerValue = "Bearer " + credential
		} else {
			headerValue = credential
		}
	case "basic":
		headerValue = "Basic " + credential
	case "header":
		headerValue = credential
	}

	if headerValue == "" {
		return args, nil
	}

	headerName := authHeader
	if headerName == "" {
		headerName = "Authorization"
	}
	return append(args, "-H", headerName+": "+headerValue), nil
}

func (e *ScanEngine) ExecuteNuclei(targetURL string, config *WebScanConfig) ([]NucleiResult, error) {
	return e.ExecuteNucleiWithSession(targetURL, config, nil)
}

func (e *ScanEngine) ExecuteNucleiWithSession(targetURL string, config *WebScanConfig, session *WebSession) ([]NucleiResult, error) {
	return e.executeNucleiWithSessionTimeout(targetURL, config, session, nucleiCommandTimeout())
}

func (e *ScanEngine) executeNucleiWithSessionTimeout(targetURL string, config *WebScanConfig, session *WebSession, commandTimeout time.Duration) ([]NucleiResult, error) {
	targetURL = sanitizeScanURL(targetURL)

	// 构建 Nuclei 命令
	args := []string{
		"-u", targetURL,
		"-jsonl", // Nuclei v3 使用 -jsonl 输出JSONL格式
		"-c", "25",
		"-silent",
		"-timeout", "10", // Nuclei v3 timeout 参数为整数（秒）
	}

	// 添加扫描选项（标签过滤）
	var options []string
	if config != nil {
		options = config.Options
	}
	if len(options) > 0 {
		var tags []string
		for _, opt := range options {
			if tag, ok := NucleiTags[opt]; ok {
				tags = append(tags, tag)
			}
		}
		if len(tags) > 0 {
			args = append(args, "-tags", strings.Join(tags, ","))
		}
	}

	// 添加认证头
	args, err := appendNucleiAuthArgs(args, targetURL, config, session)
	if err != nil {
		return nil, err
	}

	if commandTimeout <= 0 {
		commandTimeout = nucleiCommandTimeout()
	}
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "nuclei", args...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			fmt.Printf("Nuclei scan for %s timed out after %s\n", targetURL, commandTimeout)
			return nil, nil
		}
		// Nuclei 可能因为没有漏洞而返回非 0
		fmt.Printf("Nuclei scan for %s returned: %v\n", targetURL, err)
	}

	// 解析 JSON 结果
	var results []NucleiResult
	lines := strings.Split(stdout.String(), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "{") {
			continue
		}
		var result NucleiResult
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			continue
		}
		// 解析端口字符串为整数
		if result.PortStr != "" {
			fmt.Sscanf(result.PortStr, "%d", &result.Port)
		}
		results = append(results, result)
	}

	return results, nil
}

// ExecuteNucleiWithTags 使用指定标签执行 Nuclei 扫描
// 用于主机漏洞扫描，根据服务类型选择对应模板
func (e *ScanEngine) ExecuteNucleiWithTags(targetURL string, tags []string) ([]NucleiResult, error) {
	// 构建 Nuclei 命令
	args := []string{
		"-u", targetURL,
		"-jsonl", // Nuclei v3 使用 -jsonl 输出JSONL格式
		"-c", "25",
		"-silent",
		"-timeout", "10", // Nuclei v3 timeout 参数为整数（秒）
	}

	// 添加标签过滤
	if len(tags) > 0 {
		args = append(args, "-tags", strings.Join(tags, ","))
	}

	ctx, cancel := context.WithTimeout(context.Background(), nucleiCommandTimeout())
	defer cancel()

	cmd := exec.CommandContext(ctx, "nuclei", args...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			fmt.Printf("Nuclei scan with tags for %s timed out after %s\n", targetURL, nucleiCommandTimeout())
			return nil, nil
		}
		fmt.Printf("Nuclei scan with tags for %s returned: %v\n", targetURL, err)
	}

	// 解析 JSON 结果
	var results []NucleiResult
	lines := strings.Split(stdout.String(), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "{") {
			continue
		}
		var result NucleiResult
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			continue
		}
		// 解析端口字符串为整数
		if result.PortStr != "" {
			fmt.Sscanf(result.PortStr, "%d", &result.Port)
		}
		results = append(results, result)
	}

	return results, nil
}

// parseNmapGrepable 解析 Nmap grepable 输出或 RustScan 输出
// Nmap 格式:
// Host: 127.0.0.1 ()	Status: Up
// Host: 127.0.0.1 ()	Ports: 8080/open/tcp//http//Golang net|http server/,443/open/tcp//ssl|nginx//
// 或者每行一个端口：
// Host: 127.0.0.1 ()	Status: Up
// 80/open/tcp//http//Apache httpd/
// 443/open/tcp//ssl|nginx/1.18.0/
//
// RustScan 格式:
// Open 192.0.2.1:22
// Open 192.0.2.1:80
func parseNmapGrepable(output string, result *NmapResult) {
	// 先移除 ANSI 转义码
	output = stripANSI(output)

	// 调试：打印清理后的输出
	fmt.Printf("DEBUG clean output:\n%s\n", output)

	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 解析 RustScan 格式: "Open IP:port" 或 "Host: IP ()	Status: Up"
		if strings.HasPrefix(line, "Open ") {
			// RustScan 格式: Open 192.0.2.1:22
			parts := strings.SplitN(strings.TrimPrefix(line, "Open "), ":", 2)
			if len(parts) == 2 {
				host := strings.TrimSpace(parts[0])
				portStr := strings.TrimSpace(parts[1])
				if result.Host == "" {
					result.Host = host
				}
				// 解析端口号
				var portID int
				fmt.Sscanf(portStr, "%d", &portID)
				if portID > 0 {
					port := NmapPort{
						PortID:   portID,
						Protocol: "tcp",
						State:    "open",
						Service:  detectService(portID),
					}
					result.Ports = append(result.Ports, port)
				}
			}
			continue
		}

		// Host: 127.0.0.1 ()	Status: Up
		if strings.HasPrefix(line, "Host:") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				result.Host = parts[1]
			}
			continue
		}

		// 跳过 Ignored State 行
		if strings.HasPrefix(line, "Ignored State:") {
			continue
		}

		// 解析端口信息 - 可以是 "Ports:" 开头或者直接以端口信息开头
		var portStr string
		if strings.Contains(line, "Ports:") {
			portsIdx := strings.Index(line, "Ports:")
			portStr = line[portsIdx+7:] // 跳过 "Ports:"
		} else if strings.Contains(line, "/") && !strings.HasPrefix(line, "Host:") {
			// 直接以端口信息开头，如 "80/open/tcp//http//"
			portStr = line
		} else {
			continue
		}

		// 截断 "Ignored State:" 后缀
		if ignoredIdx := strings.Index(portStr, "\tIgnored State:"); ignoredIdx != -1 {
			portStr = portStr[:ignoredIdx]
		}

		// 使用逗号或制表符分割多个端口
		portItems := strings.Split(portStr, ",")
		for _, item := range portItems {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}

			// 解析端口项
			port := parsePortItem(item)
			// parsePortItem 已经过滤了非开放端口
			if port.PortID > 0 && port.State != "" {
				if port.Service == "" {
					port.Service = detectService(port.PortID)
				}
				result.Ports = append(result.Ports, port)
			}
		}
	}
}

// parsePortItem 解析单个端口项
// 格式: 端口/状态/协议///服务/版本
// 如: 22/open/tcp//ssh//OpenSSH 8.9p1 Ubuntu 3ubuntu0.4 (Ubuntu Linux; protocol 2.0)/
// 或: 8080/open/tcp//http//Golang net|http server/1.0.0/
// 状态可能是: open, closed, filtered, unfiltered, open|filtered, closed|filtered
func parsePortItem(item string) NmapPort {
	var port NmapPort

	fmt.Printf("DEBUG parsePortItem input: %s\n", item)

	// 分割: 端口/状态/协议///服务/版本
	parts := strings.Split(item, "/")
	fmt.Printf("DEBUG parts len=%d, parts=%v\n", len(parts), parts)

	if len(parts) < 3 {
		return port
	}

	// 解析端口号
	fmt.Sscanf(parts[0], "%d", &port.PortID)
	port.State = strings.TrimSpace(parts[1])
	port.Protocol = strings.TrimSpace(parts[2])

	// 只处理开放状态的端口（open 或 open|filtered）
	stateLower := strings.ToLower(port.State)
	if !strings.HasPrefix(stateLower, "open") {
		return port // 非开放端口，不保存
	}

	// 解析服务名称 (parts[4]) 和版本 (parts[5])
	if len(parts) >= 5 {
		// 服务名在 parts[4]
		if parts[4] != "" {
			serviceInfo := parts[4]
			// 检查是否有 "服务|产品" 格式
			if idx := strings.Index(serviceInfo, "|"); idx != -1 {
				port.Service = serviceInfo[:idx]
				// 版本信息在 | 后面
				if idx+1 < len(serviceInfo) {
					port.Version = strings.TrimSpace(serviceInfo[idx+1:])
				}
			} else {
				port.Service = serviceInfo
			}
		}

		// 版本信息在 parts[5] (通常是完整的版本字符串)
		if len(parts) >= 6 && parts[5] != "" {
			version := strings.TrimSpace(parts[5])
			if port.Version != "" {
				port.Version = port.Version + " " + version
			} else {
				port.Version = version
			}
		}

		// 如果 parts[6] 或更后面有内容，也拼接到版本中
		if len(parts) >= 7 {
			for i := 6; i < len(parts); i++ {
				if parts[i] != "" {
					port.Version = port.Version + " " + strings.TrimSpace(parts[i])
				}
			}
		}
	}

	fmt.Printf("DEBUG parsePortItem result: port=%d, service=%s, version=%s\n", port.PortID, port.Service, port.Version)

	// 生成 banner 信息
	if port.Service != "" && port.Version != "" {
		port.Banner = fmt.Sprintf("%s %s", port.Service, port.Version)
	} else if port.Service != "" {
		port.Banner = port.Service
	}

	return port
}

// detectService 根据端口检测服务类型
func detectService(port int) string {
	services := map[int]string{
		21:    "ftp",
		22:    "ssh",
		23:    "telnet",
		25:    "smtp",
		53:    "dns",
		80:    "http",
		110:   "pop3",
		143:   "imap",
		443:   "https",
		445:   "smb",
		1521:  "oracle",
		2181:  "zookeeper",
		2375:  "docker",
		2376:  "docker",
		2379:  "etcd",
		2380:  "etcd",
		3306:  "mysql",
		3389:  "rdp",
		5432:  "postgresql",
		5672:  "amqp",
		6443:  "kube",
		6379:  "redis",
		8080:  "http-proxy",
		8443:  "https-alt",
		8500:  "consul",
		8501:  "consul",
		9092:  "kafka",
		9200:  "elasticsearch",
		11211: "memcached",
		15672: "rabbitmq",
	}
	if s, ok := services[port]; ok {
		return s
	}
	return "unknown"
}

// NmapXMLScript Nmap XML 中的脚本信息
type NmapXMLScript struct {
	XMLName xml.Name `xml:"script"`
	ID      string   `xml:"id,attr"`
	Output  string   `xml:"output,attr"`
}

// NmapXMLService Nmap XML 中的服务信息
type NmapXMLService struct {
	XMLName   xml.Name `xml:"service"`
	Name      string   `xml:"name,attr"`
	Product   string   `xml:"product,attr"`
	Version   string   `xml:"version,attr"`
	Extrainfo string   `xml:"extrainfo,attr"`
	OSType    string   `xml:"ostype,attr"`
	Method    string   `xml:"method,attr"`
	Conf      int      `xml:"conf,attr"`
	CPEs      []string `xml:"cpe"` // service 下的 cpe 子元素
}

// NmapXMLAddress Nmap XML 中的地址信息
type NmapXMLAddress struct {
	XMLName  xml.Name `xml:"address"`
	Addr     string   `xml:"addr,attr"`
	AddrType string   `xml:"addrtype,attr"`
}

// NmapXMLState Nmap XML 中的端口状态信息
type NmapXMLState struct {
	XMLName   xml.Name `xml:"state"`
	State     string   `xml:"state,attr"`
	Reason    string   `xml:"reason,attr"`
	ReasonTTL string   `xml:"reason_ttl,attr"`
}

// NmapXMLPort Nmap XML 输出中的端口信息
type NmapXMLPort struct {
	XMLName  xml.Name        `xml:"port"`
	PortID   int             `xml:"portid,attr"`
	Protocol string          `xml:"protocol,attr"`
	State    NmapXMLState    `xml:"state"`
	Service  NmapXMLService  `xml:"service"`
	Scripts  []NmapXMLScript `xml:"script"`
}

// NmapXMLPorts Nmap XML 中的端口列表
type NmapXMLPorts struct {
	Ports []NmapXMLPort `xml:"port"`
}

// NmapXMLHost Nmap XML 输出中的主机信息
type NmapXMLHost struct {
	XMLName xml.Name       `xml:"host"`
	Address NmapXMLAddress `xml:"address"`
	Ports   NmapXMLPorts   `xml:"ports"`
}

// NmapXMLOutput Nmap XML 输出根结构
type NmapXMLOutput struct {
	XMLName xml.Name      `xml:"nmaprun"`
	Hosts   []NmapXMLHost `xml:"host"`
}

// parseNmapXML 解析 Nmap XML 输出，提取端口、服务、版本和 CPE 信息
func parseNmapXML(output string, result *NmapResult) {
	// 先移除 ANSI 转义码
	output = stripANSI(output)

	// 移除 XML 注释 (<!-- ... -->)，Nmap 输出的 XML 可能包含这些注释
	// 使用正则表达式移除所有 XML 注释
	commentRegex := regexp.MustCompile(`<!--[\s\S]*?-->`)
	output = commentRegex.ReplaceAllString(output, "")

	var nmapOutput NmapXMLOutput
	if err := xml.Unmarshal([]byte(output), &nmapOutput); err != nil {
		fmt.Printf("DEBUG XML parse error: %v\n", err)
		// XML 解析失败，回退到 Grepable 解析
		parseNmapGrepable(output, result)
		return
	}

	fmt.Printf("DEBUG XML hosts count: %d\n", len(nmapOutput.Hosts))

	// 打印原始 XML 的前 500 字符用于调试
	if len(output) > 500 {
		fmt.Printf("DEBUG XML raw (first 500): %s\n", output[:500])
	} else {
		fmt.Printf("DEBUG XML raw: %s\n", output)
	}

	for _, host := range nmapOutput.Hosts {
		if result.Host == "" && host.Address.Addr != "" {
			result.Host = host.Address.Addr
		}

		for _, port := range host.Ports.Ports {
			state := port.State.State
			fmt.Printf("DEBUG XML parsed: port=%d, state=%s, protocol=%s, service=%s\n", port.PortID, state, port.Protocol, port.Service.Name)

			if !strings.Contains(strings.ToLower(state), "open") {
				continue
			}

			service := port.Service.Name
			if service == "" {
				service = detectService(port.PortID)
			}

			product := port.Service.Product
			version := port.Service.Version
			if version == "" {
				version = strings.TrimSpace(port.Service.Extrainfo)
			}

			cpe := ""
			for _, candidate := range port.Service.CPEs {
				candidate = strings.TrimSpace(candidate)
				if candidate != "" {
					cpe = candidate
					break
				}
			}

			service = markUnverifiedService(service, port.Service.Method, port.Service.Conf, product, version, cpe)

			nmapPort := NmapPort{
				PortID:     port.PortID,
				Protocol:   port.Protocol,
				State:      state,
				Service:    service,
				Version:    version,
				Product:    product,
				CPE:        cpe,
				Banner:     buildBanner(service, product, version),
				Method:     port.Service.Method,
				Confidence: port.Service.Conf,
			}

			fmt.Printf("DEBUG parseNmapXML: port=%d, service=%s, product=%s, version=%s, cpe=%s\n",
				nmapPort.PortID, nmapPort.Service, nmapPort.Product, nmapPort.Version, nmapPort.CPE)

			result.Ports = append(result.Ports, nmapPort)
		}
	}
}

// buildBanner 构建服务 banner 字符串
func buildBanner(service, product, version string) string {
	var parts []string
	if service != "" {
		parts = append(parts, service)
	}
	if product != "" && product != service {
		parts = append(parts, product)
	}
	if version != "" {
		parts = append(parts, version)
	}
	return strings.Join(parts, " ")
}

func buildHTTPCompositeBanner(frontService, frontProduct, frontVersion, upstreamService, upstreamProduct, upstreamVersion string) string {
	banner := buildBanner(frontService, frontProduct, frontVersion)
	if upstreamService == "" && upstreamProduct == "" && upstreamVersion == "" {
		return banner
	}

	var upstreamParts []string
	if upstreamService != "" {
		upstreamParts = append(upstreamParts, upstreamService)
	}
	if upstreamProduct != "" && upstreamProduct != upstreamService {
		upstreamParts = append(upstreamParts, upstreamProduct)
	}
	if upstreamVersion != "" {
		upstreamParts = append(upstreamParts, upstreamVersion)
	}
	if len(upstreamParts) == 0 {
		return banner
	}
	if banner == "" {
		return "upstream=" + strings.Join(upstreamParts, " ")
	}
	return banner + " | upstream=" + strings.Join(upstreamParts, " ")
}

// extractVulnType 从 nuclei 模板标签中提取漏洞类型
func extractVulnType(tags []string, templateType string) string {
	// 常见的漏洞类型映射
	vulnTypeMap := map[string]string{
		"cve":               "CVE漏洞",
		"rce":               "远程代码执行",
		"xss":               "跨站脚本",
		"sqli":              "SQL注入",
		"ssrf":              "服务端请求伪造",
		"csrf":              "跨站请求伪造",
		"lfi":               "本地文件包含",
		"rfi":               "远程文件包含",
		"ssti":              "模板注入",
		"exposure":          "信息泄露",
		"misconfig":         "配置错误",
		"default-login":     "默认口令",
		"bruteforce":        "暴力破解",
		"dos":               "拒绝服务",
		"file-read":         "文件读取",
		"file-write":        "文件写入",
		"command-injection": "命令注入",
		"xxe":               "XML实体注入",
		"jsonp":             "JSONP泄露",
		"open-redirect":     "开放重定向",
	}

	// 从标签中查找漏洞类型
	for _, tag := range tags {
		tagLower := strings.ToLower(tag)
		if vt, ok := vulnTypeMap[tagLower]; ok {
			return vt
		}
	}

	// 如果没有找到，使用模板类型
	if templateType != "" {
		return templateType
	}

	return "其他"
}

// GetCVSSLevel 根据CVSS分数返回等级
func GetCVSSLevel(score float64) string {
	if score >= 9.0 {
		return "超危"
	} else if score >= 7.0 {
		return "高危"
	} else if score >= 4.0 {
		return "中危"
	} else if score > 0 {
		return "低危"
	}
	return "未知"
}

func buildWebNmapResultFromURL(targetURL string) (*NmapResult, error) {
	parsed, err := url.Parse(sanitizeScanURL(targetURL))
	if err != nil {
		return nil, err
	}

	host := parsed.Hostname()
	if host == "" {
		return nil, fmt.Errorf("invalid target URL: missing host")
	}

	port := 0
	if parsed.Port() != "" {
		fmt.Sscanf(parsed.Port(), "%d", &port)
	}
	if port == 0 {
		switch strings.ToLower(parsed.Scheme) {
		case "https":
			port = 443
		default:
			port = 80
		}
	}

	service := "http"
	if strings.EqualFold(parsed.Scheme, "https") || port == 443 || port == 8443 {
		service = "https"
	}

	return &NmapResult{
		Host: host,
		Ports: []NmapPort{
			{
				PortID:   port,
				Protocol: "tcp",
				State:    "open",
				Service:  service,
				Banner:   service,
			},
		},
	}, nil
}

func nonAuthSessionHeaders(session *WebSession) []AuthHeader {
	if session == nil || len(session.Headers) == 0 {
		return nil
	}
	headers := make([]AuthHeader, 0, len(session.Headers))
	for _, header := range session.Headers {
		name := strings.ToLower(strings.TrimSpace(header.Name))
		switch name {
		case "authorization", "cookie", "token", "x-auth-token":
			continue
		default:
			headers = append(headers, header)
		}
	}
	return headers
}

func applyHeaderList(req *http.Request, headers []AuthHeader) {
	for _, header := range headers {
		name := strings.TrimSpace(header.Name)
		if name == "" {
			continue
		}
		req.Header.Set(name, header.Value)
	}
}

func fetchWebRuleResponse(targetURL string, headers []AuthHeader) (int, string, http.Header, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sanitizeScanURL(targetURL), nil)
	if err != nil {
		return 0, "", nil, err
	}
	applyHeaderList(req, headers)

	client := &http.Client{
		Timeout: 12 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, "", nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return resp.StatusCode, "", resp.Header.Clone(), err
	}

	return resp.StatusCode, strings.TrimSpace(string(body)), resp.Header.Clone(), nil
}

func extractSensitiveMarkers(body string) []string {
	if body == "" {
		return nil
	}
	markers := []string{
		"device_id", "tenant", "permission", "plugin", "version", "secret", "token",
		"timezone", "browser_title", "theme_color", "user_client_logo", "watermark",
		"install", "path", "max_cores", "start_time", "end_time",
	}
	hits := make([]string, 0, 6)
	seen := map[string]struct{}{}
	lower := strings.ToLower(body)
	for _, marker := range markers {
		if strings.Contains(lower, marker) {
			if _, ok := seen[marker]; ok {
				continue
			}
			seen[marker] = struct{}{}
			hits = append(hits, marker)
		}
	}
	return hits
}

func buildSyntheticRuleFinding(targetURL, templateID, name, severity, vulnType, description, solution, response string, tags []string) NucleiResult {
	return NucleiResult{
		TemplateID: templateID,
		Host:       targetURL,
		MatchedAt:  targetURL,
		Request:    fmt.Sprintf("GET %s", sanitizeScanURL(targetURL)),
		Response:   response,
		Severity:   severity,
		URL:        sanitizeScanURL(targetURL),
		Type:       "http",
		Info: NucleiInfo{
			Name:        name,
			Description: description,
			Solution:    solution,
			Type:        vulnType,
			Tags:        tags,
		},
	}
}

func normalizeWebScanOptions(options []string) map[string]struct{} {
	normalized := make(map[string]struct{}, len(options)*2)
	for _, option := range options {
		option = strings.ToLower(strings.TrimSpace(option))
		if option == "" {
			continue
		}
		normalized[option] = struct{}{}
		if mapped, ok := NucleiTags[option]; ok {
			normalized[mapped] = struct{}{}
		}
	}
	return normalized
}

func webRuleEnabled(options []string, categories ...string) bool {
	if len(options) == 0 {
		return true
	}
	normalized := normalizeWebScanOptions(options)
	for _, category := range categories {
		category = strings.ToLower(strings.TrimSpace(category))
		if category == "" {
			continue
		}
		if _, ok := normalized[category]; ok {
			return true
		}
	}
	return false
}

func detectWebRuleFindings(targetURL string, session *WebSession, options []string) []NucleiResult {
	infoDisclosureEnabled := webRuleEnabled(options, "information-disclosure")
	brokenAccessEnabled := webRuleEnabled(options, "broken-access", "broken-access-control")
	if !infoDisclosureEnabled && !brokenAccessEnabled {
		return nil
	}

	publicHeaders := nonAuthSessionHeaders(session)
	unauthStatus, unauthBody, _, err := fetchWebRuleResponse(targetURL, publicHeaders)
	if err != nil {
		return nil
	}

	findings := make([]NucleiResult, 0, 2)
	trimmedUnauth := strings.TrimSpace(unauthBody)
	markers := extractSensitiveMarkers(trimmedUnauth)
	if infoDisclosureEnabled && unauthStatus == http.StatusOK && len(markers) > 0 {
		severity := "medium"
		if len(markers) >= 3 {
			severity = "high"
		}
		findings = append(findings, buildSyntheticRuleFinding(
			targetURL,
			"web-rule-info-disclosure",
			"未授权敏感信息泄露",
			severity,
			"information-disclosure",
			fmt.Sprintf("未授权访问即可读取接口响应，检测到敏感字段特征：%s。", strings.Join(markers, ", ")),
			"为该接口增加认证与最小化字段返回控制，避免匿名用户读取内部配置、功能或设备信息。",
			trimmedUnauth,
			[]string{"information-disclosure", "unauthenticated", "web-rule"},
		))
	}

	if session == nil || len(session.Headers) == 0 {
		return findings
	}

	authStatus, authBody, _, err := fetchWebRuleResponse(targetURL, session.Headers)
	if err != nil {
		return findings
	}

	trimmedAuth := strings.TrimSpace(authBody)
	if brokenAccessEnabled && unauthStatus == http.StatusOK && authStatus == http.StatusOK && trimmedUnauth != "" && trimmedUnauth == trimmedAuth {
		findings = append(findings, buildSyntheticRuleFinding(
			targetURL,
			"web-rule-unauthorized-access",
			"疑似未授权访问",
			"medium",
			"broken-access-control",
			"未授权请求与认证请求返回状态和响应内容一致，疑似存在未授权访问。",
			"为该接口增加认证和权限校验，并为匿名请求返回 401/403。",
			trimmedUnauth,
			[]string{"broken-access", "unauthenticated", "web-rule"},
		))
	}

	return findings
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func truncateUTF8ByBytes(value string, maxBytes int) string {
	if maxBytes <= 0 || len(value) <= maxBytes {
		return value
	}
	truncated := value[:maxBytes]
	for len(truncated) > 0 && !utf8.ValidString(truncated) {
		truncated = truncated[:len(truncated)-1]
	}
	return truncated
}

func sanitizeVulnerabilityTextField(value string) string {
	const maxTextColumnBytes = 60 * 1024
	return truncateUTF8ByBytes(value, maxTextColumnBytes)
}

func normalizeScanHost(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	if strings.Contains(raw, "://") {
		if parsed, err := url.Parse(raw); err == nil && parsed.Hostname() != "" {
			return parsed.Hostname()
		}
	}

	if strings.Contains(raw, "/") {
		raw = strings.SplitN(raw, "/", 2)[0]
	}

	if host, _, err := net.SplitHostPort(raw); err == nil && host != "" {
		return host
	}

	if strings.Count(raw, ":") == 1 {
		if host, _, err := net.SplitHostPort(raw); err == nil && host != "" {
			return host
		}
	}

	return raw
}

func inferPortFromScheme(scheme string) int {
	switch strings.ToLower(strings.TrimSpace(scheme)) {
	case "https":
		return 443
	case "http":
		return 80
	default:
		return 0
	}
}

func parseHostPortFromNucleiResult(vuln NucleiResult, defaultHost string, defaultPort int) (string, int) {
	host := normalizeScanHost(vuln.IP)
	port := vuln.Port

	if host == "" && strings.TrimSpace(vuln.Host) != "" {
		rawHost := strings.TrimSpace(vuln.Host)
		if strings.Contains(rawHost, "://") {
			if parsed, err := url.Parse(rawHost); err == nil {
				host = normalizeScanHost(parsed.Hostname())
				if port == 0 && parsed.Port() != "" {
					fmt.Sscanf(parsed.Port(), "%d", &port)
				}
				if port == 0 {
					port = inferPortFromScheme(parsed.Scheme)
				}
			}
		} else {
			host = normalizeScanHost(rawHost)
			if port == 0 {
				if parsedHost, parsedPort, err := net.SplitHostPort(rawHost); err == nil {
					host = normalizeScanHost(parsedHost)
					fmt.Sscanf(parsedPort, "%d", &port)
				}
			}
		}
	}

	if strings.TrimSpace(vuln.URL) != "" {
		if parsedURL, err := url.Parse(vuln.URL); err == nil {
			if host == "" {
				host = normalizeScanHost(parsedURL.Hostname())
			}
			if port == 0 && parsedURL.Port() != "" {
				fmt.Sscanf(parsedURL.Port(), "%d", &port)
			}
			if port == 0 {
				port = inferPortFromScheme(parsedURL.Scheme)
			}
		}
	}

	if host == "" {
		host = normalizeScanHost(defaultHost)
	}
	if port == 0 {
		port = defaultPort
	}

	return host, port
}

// syncToAssetCenter 同步资产到统一资产中心
func (e *ScanEngine) syncToAssetCenter(tx *gorm.DB, ip string, port NmapPort) {
	now := time.Now()

	// 检查资产是否已存在
	var existingAsset models.Asset
	err := tx.Where("ip = ? AND port = ?", ip, port.PortID).First(&existingAsset).Error

	if err == gorm.ErrRecordNotFound {
		// 新资产，创建记录
		assetType := e.detectAssetType(port.PortID, port.Service)
		newAsset := models.Asset{
			IP:          ip,
			Port:        port.PortID,
			Protocol:    port.Protocol,
			ServiceName: port.Service,
			Version:     port.Version,
			Banner:      port.Banner,
			AssetType:   assetType,
			Status:      models.AssetStatusOnline,
			FirstSeen:   now,
			LastSeen:    now,
		}
		if err := tx.Create(&newAsset).Error; err != nil {
			fmt.Printf("Failed to create asset %s:%d - %v\n", ip, port.PortID, err)
		} else {
			fmt.Printf("Created new asset: %s:%d (%s)\n", ip, port.PortID, port.Service)
		}
	} else if err == nil {
		// 资产已存在，更新信息
		updates := map[string]interface{}{
			"protocol":     port.Protocol,
			"service_name": port.Service,
			"version":      port.Version,
			"banner":       port.Banner,
			"status":       models.AssetStatusOnline,
			"last_seen":    now,
		}
		if err := tx.Model(&existingAsset).Updates(updates).Error; err != nil {
			fmt.Printf("Failed to update asset %s:%d - %v\n", ip, port.PortID, err)
		} else {
			fmt.Printf("Updated asset: %s:%d (%s)\n", ip, port.PortID, port.Service)
		}
	}
}

// detectAssetType 根据端口和服务检测资产类型
func (e *ScanEngine) detectAssetType(port int, service string) string {
	// 根据端口判断
	switch port {
	case 22, 3389:
		return models.AssetTypeServer
	case 80, 443, 8080, 8443:
		return models.AssetTypeWeb
	case 3306, 5432, 6379, 27017:
		return models.AssetTypeDatabase
	case 161, 162, 514:
		return models.AssetTypeNetwork
	}

	// 根据服务名判断
	serviceLower := normalizeServiceName(service)
	if strings.Contains(serviceLower, "ssh") || strings.Contains(serviceLower, "rdp") {
		return models.AssetTypeServer
	}
	if strings.Contains(serviceLower, "http") || strings.Contains(serviceLower, "https") {
		return models.AssetTypeWeb
	}
	if strings.Contains(serviceLower, "mysql") || strings.Contains(serviceLower, "postgres") ||
		strings.Contains(serviceLower, "redis") || strings.Contains(serviceLower, "mongo") {
		return models.AssetTypeDatabase
	}
	if strings.Contains(serviceLower, "snmp") || strings.Contains(serviceLower, "syslog") {
		return models.AssetTypeNetwork
	}

	return models.AssetTypeOther
}

// SaveScanResult 保存扫描结果到数据库（包含漏洞去重）
func (e *ScanEngine) SaveScanResult(taskID uint, nmapResult *NmapResult, nucleiResults []NucleiResult) error {
	// 初始化漏洞知识库服务
	vulnDB := NewVulnDBService()
	assetHost := normalizeScanHost(nmapResult.Host)
	if assetHost == "" {
		assetHost = strings.TrimSpace(nmapResult.Host)
	}

	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var task models.SecurityScanTask
	if err := tx.Select("id", "scan_type").First(&task, taskID).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 构建端口到资产的映射
	portToAsset := make(map[int]models.SecurityAsset)
	defaultPort := 0
	for _, port := range nmapResult.Ports {
		if defaultPort == 0 && port.PortID > 0 {
			defaultPort = port.PortID
		}
		// 检查资产是否已存在（同 IP + 端口）
		var existingAsset models.SecurityAsset
		err := tx.Where("task_id = ? AND ip = ? AND port = ?", taskID, assetHost, port.PortID).First(&existingAsset).Error

		if err == gorm.ErrRecordNotFound {
			// 创建新资产
			asset := models.SecurityAsset{
				TaskID:      taskID,
				IP:          assetHost,
				Port:        port.PortID,
				Protocol:    port.Protocol,
				ServiceName: port.Service,
				Version:     port.Version,
				Banner:      port.Banner,
			}
			if err := tx.Create(&asset).Error; err != nil {
				tx.Rollback()
				return err
			}
			portToAsset[port.PortID] = asset

			// 同步到统一资产中心
			e.syncToAssetCenter(tx, assetHost, port)
		} else if err == nil {
			// 资产已存在，更新信息
			updates := map[string]interface{}{
				"protocol":     port.Protocol,
				"service_name": port.Service,
				"version":      port.Version,
				"banner":       port.Banner,
			}
			if err := tx.Model(&existingAsset).Updates(updates).Error; err != nil {
				tx.Rollback()
				return err
			}
			portToAsset[port.PortID] = existingAsset
		}
	}

	var foundationCtx *hostScanPhase1Context
	var webPhase1Ctx *webScanPhase1Context
	var err error
	if task.ScanType == string(models.ScanTypeWeb) {
		webPhase1Ctx, err = ensureWebScanPhase1Context(tx, taskID)
	} else {
		foundationCtx, err = ensureHostScanPhase1Context(tx, taskID, assetHost, nmapResult.Ports)
	}
	if err != nil {
		tx.Rollback()
		return err
	}

	// 将 nuclei 结果按 IP 和端口分组
	type vulnKey struct {
		ip   string
		port int
	}
	vulnMap := make(map[vulnKey][]NucleiResult)
	for _, vuln := range nucleiResults {
		ip, port := parseHostPortFromNucleiResult(vuln, assetHost, defaultPort)

		key := vulnKey{ip: ip, port: port}
		vulnMap[key] = append(vulnMap[key], vuln)
	}

	// 保存漏洞（带去重）
	for key, vulns := range vulnMap {
		asset, ok := portToAsset[key.port]
		if !ok {
			// 如果找不到对应的资产，创建关联
			asset = models.SecurityAsset{
				TaskID: taskID,
				IP:     key.ip,
				Port:   key.port,
			}
		}

		for _, vuln := range vulns {
			severity := strings.ToLower(vuln.Severity)
			if severity == "critical" || severity == "high" {
				severity = "high"
			} else if severity == "medium" {
				severity = "medium"
			} else {
				severity = "low"
			}

			cveIDs := strings.Join(vuln.CVEs, ",")

			// 从 CVE 列表中获取主 CVE 并从知识库补全信息
			var mainCVE string
			if len(vuln.CVEs) > 0 {
				mainCVE = vuln.CVEs[0]
			}

			// 从知识库获取漏洞增强信息
			var cnvdID, cnnvdID, cncveID, enrichedSolution string
			var enrichedCVSSScore float64
			if mainCVE != "" {
				enrichment := vulnDB.EnrichVulnerability(mainCVE)
				if enrichment != nil {
					cnvdID = enrichment.CNVDID
					cnnvdID = enrichment.CNNVDID
					cncveID = enrichment.CNCVEID
					enrichedSolution = enrichment.Solution
					// 如果 nuclei 没有提供 CVSS 分数，使用知识库中的
					if vuln.CVSSScore == 0 && enrichment.CVSSScore > 0 {
						enrichedCVSSScore = enrichment.CVSSScore
					}
				}
			}

			// 优先使用 nuclei 提供的解决方案，如果没有则使用知识库中的
			finalSolution := vuln.Info.Solution
			if finalSolution == "" && enrichedSolution != "" {
				finalSolution = enrichedSolution
			}

			// 确定最终的 CVSS 分数
			finalCVSSScore := vuln.CVSSScore
			if finalCVSSScore == 0 && enrichedCVSSScore > 0 {
				finalCVSSScore = enrichedCVSSScore
			}

			// 确定处置优先级
			var priority string
			switch severity {
			case "high":
				priority = "优先级高"
			case "medium":
				priority = "优先级中"
			default:
				priority = "优先级低"
			}

			// 获取漏洞类型（从模板标签中提取）
			vulnType := extractVulnType(vuln.Info.Tags, vuln.Info.Type)

			// 构建漏洞地址
			vulnURL := strings.TrimSpace(vuln.URL)
			if vulnURL == "" {
				matched := strings.TrimSpace(vuln.MatchedAt)
				if strings.HasPrefix(strings.ToLower(matched), "http://") || strings.HasPrefix(strings.ToLower(matched), "https://") {
					vulnURL = matched
				}
			}
			if vulnURL == "" {
				if key.port > 0 {
					protocol := "http"
					if strings.Contains(strings.ToLower(vuln.MatchedAt), "https") || key.port == 443 {
						protocol = "https"
					}
					vulnURL = fmt.Sprintf("%s://%s:%d", protocol, key.ip, key.port)
				} else {
					vulnURL = fmt.Sprintf("http://%s", key.ip)
				}
			}

			// 确定扫描方式（根据扫描类型）
			scanMethod := "非授权扫描"
			if len(vuln.Info.Tags) > 0 {
				for _, tag := range vuln.Info.Tags {
					if tag == "active" || tag == "exploit" {
						scanMethod = "主动探测"
						break
					}
				}
			}

			// === 漏洞去重逻辑 ===
			// 检查是否已存在相同漏洞（基于 IP + 端口 + CVE ID 或 模板ID）
			var existingVuln models.SecurityVulnerability
			dedupQuery := tx.Where("ip = ? AND port = ?", key.ip, key.port)

			// 如果有 CVE ID，加入去重条件
			if mainCVE != "" {
				dedupQuery = dedupQuery.Where("cve_id LIKE ?", "%"+mainCVE+"%")
			} else {
				// 没有 CVE 时，使用模板 ID 去重
				dedupQuery = dedupQuery.Where("template_id = ?", vuln.TemplateID)
				if strings.HasPrefix(vuln.TemplateID, "web-rule-") && vulnURL != "" {
					dedupQuery = dedupQuery.Where("vuln_url = ?", vulnURL)
				}
			}

			err := dedupQuery.First(&existingVuln).Error

			if err == gorm.ErrRecordNotFound {
				// 漏洞不存在，创建新记录
				vulnerability := models.SecurityVulnerability{
					AssetID:       asset.ID,
					IP:            key.ip,
					Port:          key.port,
					Protocol:      asset.Protocol,
					Severity:      severity,
					CVSSScore:     finalCVSSScore,
					VulnType:      vulnType,
					CVEID:         cveIDs,
					CNVDID:        cnvdID,
					CNNVDID:       cnnvdID,
					CNCVEID:       cncveID,
					Title:         vuln.Info.Name,
					Description:   vuln.Info.Description,
					Solution:      finalSolution,
					Payload:       sanitizeVulnerabilityTextField(firstNonEmpty(vuln.URL, vuln.MatchedAt)),
					Request:       sanitizeVulnerabilityTextField(vuln.Request),
					Response:      sanitizeVulnerabilityTextField(vuln.Response),
					ReferenceURL:  strings.Join(vuln.Info.Reference, ","),
					Scanner:       "nuclei",
					TemplateID:    vuln.TemplateID,
					ScanMethod:    scanMethod,
					VulnURL:       vulnURL,
					PrimaryCVEID:  mainCVE,
					Priority:      priority,
					FalsePositive: false,
				}
				assignVulnerabilityTaskTracking(&vulnerability, taskID)
				decorateInventoryResult(&vulnerability, vuln)
				applyDerivedVulnerabilityMetadata(&vulnerability)
				if vulnerability.PrimaryCVEID != "" {
					if record := vulnDB.LookupCVE(vulnerability.PrimaryCVEID); record != nil {
						id := record.ID
						vulnerability.VulnDBID = &id
					}
				}
				if err := tx.Create(&vulnerability).Error; err != nil {
					tx.Rollback()
					return err
				}
				if task.ScanType == string(models.ScanTypeWeb) {
					if err := recordWebFindingOccurrence(tx, webPhase1Ctx, vulnerability, vuln); err != nil {
						tx.Rollback()
						return err
					}
				} else {
					if err := recordNucleiFindingOccurrence(tx, foundationCtx, key.port, vulnerability, vuln); err != nil {
						tx.Rollback()
						return err
					}
				}
				fmt.Printf("Created new vulnerability: %s on %s:%d (type=%s, cve=%s)\n",
					vuln.Info.Name, key.ip, key.port, vulnType, mainCVE)
			} else if err == nil {
				// 漏洞已存在，更新内容并记录最近命中的任务
				updates := map[string]interface{}{
					"severity":       severity,
					"cvss_score":     finalCVSSScore,
					"title":          vuln.Info.Name,
					"description":    vuln.Info.Description,
					"solution":       finalSolution,
					"vuln_type":      vulnType,
					"scan_method":    scanMethod,
					"payload":        sanitizeVulnerabilityTextField(firstNonEmpty(vuln.URL, vuln.MatchedAt)),
					"request":        sanitizeVulnerabilityTextField(vuln.Request),
					"response":       sanitizeVulnerabilityTextField(vuln.Response),
					"vuln_url":       vulnURL,
					"primary_cve_id": mainCVE,
					"reference_url":  strings.Join(vuln.Info.Reference, ","),
					"updated_at":     time.Now(),
				}
				applyVulnerabilityTaskTrackingUpdates(updates, &existingVuln, taskID)
				inventoryVuln := existingVuln
				inventoryVuln.VulnType = vulnType
				inventoryVuln.ScanMethod = scanMethod
				inventoryVuln.Description = vuln.Info.Description
				decorateInventoryResult(&inventoryVuln, vuln)
				updates["vuln_type"] = inventoryVuln.VulnType
				updates["scan_method"] = inventoryVuln.ScanMethod
				if inventoryVuln.Description != "" {
					updates["description"] = inventoryVuln.Description
				}
				applyDerivedVulnerabilityMetadata(&inventoryVuln)
				updates["finding_source"] = inventoryVuln.FindingSource
				updates["finding_family"] = inventoryVuln.FindingFamily
				updates["confidence"] = inventoryVuln.Confidence
				updates["match_mode"] = inventoryVuln.MatchMode
				updates["primary_cve_id"] = inventoryVuln.PrimaryCVEID
				if inventoryVuln.PrimaryCVEID != "" {
					if record := vulnDB.LookupCVE(inventoryVuln.PrimaryCVEID); record != nil {
						updates["vuln_db_id"] = record.ID
					} else {
						updates["vuln_db_id"] = nil
					}
				} else {
					updates["vuln_db_id"] = nil
				}
				if err := tx.Model(&existingVuln).Updates(updates).Error; err != nil {
					tx.Rollback()
					return err
				}
				updatedVuln := existingVuln
				updatedVuln.Severity = severity
				updatedVuln.CVSSScore = finalCVSSScore
				updatedVuln.Title = vuln.Info.Name
				updatedVuln.Description = firstNonEmpty(inventoryVuln.Description, vuln.Info.Description)
				updatedVuln.Solution = finalSolution
				updatedVuln.VulnType = inventoryVuln.VulnType
				updatedVuln.ScanMethod = inventoryVuln.ScanMethod
				updatedVuln.Payload = firstNonEmpty(vuln.URL, vuln.MatchedAt)
				updatedVuln.Request = vuln.Request
				updatedVuln.Response = vuln.Response
				updatedVuln.VulnURL = vulnURL
				updatedVuln.PrimaryCVEID = inventoryVuln.PrimaryCVEID
				updatedVuln.ReferenceURL = strings.Join(vuln.Info.Reference, ",")
				updatedVuln.FindingSource = inventoryVuln.FindingSource
				updatedVuln.FindingFamily = inventoryVuln.FindingFamily
				updatedVuln.Confidence = inventoryVuln.Confidence
				updatedVuln.MatchMode = inventoryVuln.MatchMode
				if primaryRaw, ok := updates["primary_cve_id"].(string); ok && primaryRaw != "" {
					updatedVuln.PrimaryCVEID = primaryRaw
				}
				if vulnDBID, ok := updates["vuln_db_id"].(uint); ok {
					updatedVuln.VulnDBID = &vulnDBID
				} else if vulnDBID, ok := updates["vuln_db_id"].(int); ok {
					id := uint(vulnDBID)
					updatedVuln.VulnDBID = &id
				} else if updates["vuln_db_id"] == nil {
					updatedVuln.VulnDBID = nil
				}
				if task.ScanType == string(models.ScanTypeWeb) {
					if err := recordWebFindingOccurrence(tx, webPhase1Ctx, updatedVuln, vuln); err != nil {
						tx.Rollback()
						return err
					}
				} else {
					if err := recordNucleiFindingOccurrence(tx, foundationCtx, key.port, updatedVuln, vuln); err != nil {
						tx.Rollback()
						return err
					}
				}
				fmt.Printf("Updated existing vulnerability: %s on %s:%d (type=%s)\n",
					vuln.Info.Name, key.ip, key.port, vulnType)
			}
		}
	}

	return tx.Commit().Error
}

// handleWebURLScan 处理 Web URL 扫描（直接对 URL 执行 Nuclei，无需 Nmap）
func handleWebURLScan(taskID uint, target string, engine *ScanEngine, config *WebScanConfig) {
	fmt.Printf("Starting web URL scan for task %d\n", taskID)

	if config == nil || strings.TrimSpace(config.AuthMode) == "" || strings.EqualFold(strings.TrimSpace(config.AuthMode), "none") {
		_ = UpdateTaskAndCurrentRun(taskID, map[string]interface{}{
			"status":  "failed",
			"message": "Web 扫描已禁用匿名模式，请提供登录态后重试",
		})
		return
	}

	// 解析 URL 列表
	urls := strings.Split(target, ",")
	var cleanURLs []string
	for _, url := range urls {
		url = strings.TrimSpace(url)
		if url != "" {
			cleanURLs = append(cleanURLs, url)
		}
	}

	if len(cleanURLs) == 0 {
		database.DB.Model(&models.SecurityScanTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
			"status":  "failed",
			"message": "URL 列表为空",
		})
		return
	}

	_ = UpdateTaskAndCurrentRun(taskID, map[string]interface{}{
		"config_snapshot": phase1WebConfigSnapshot(target, cleanURLs, config),
		"phase":           "discovery",
	})
	UpdateTaskProgress(taskID, 0, 0, len(cleanURLs), "开始 Web 漏洞扫描...")

	var totalHighRisk, totalMediumRisk, totalLowRisk int
	discoveryPlan := make([]DiscoveredTarget, 0, len(cleanURLs))
	seenTargets := map[string]struct{}{}
	sessionByTarget := map[string]*WebSession{}
	discoveryWarnings := make([]map[string]interface{}, 0)

	rememberSession := func(target string, session *WebSession) {
		if session == nil || len(session.Headers) == 0 {
			return
		}
		sessionByTarget[sanitizeScanURL(target)] = session
	}

	discoveryEnabled := config == nil || !strings.EqualFold(strings.TrimSpace(config.DiscoveryMode), "none")
	discoveryOpts := defaultDiscoveryOptions(config)
	discoveryMode := "http"
	if config != nil && strings.TrimSpace(config.DiscoveryMode) != "" {
		discoveryMode = strings.ToLower(strings.TrimSpace(config.DiscoveryMode))
	}

	for i, entryURL := range cleanURLs {
		if continueScan, msg := CheckTaskStatus(taskID); !continueScan {
			updates := map[string]interface{}{
				"message": msg,
			}
			if status := GetTaskStatus(taskID); status != "" {
				updates["status"] = status
				if status == models.TaskStatusCancelled {
					completedAt := time.Now()
					updates["completed_at"] = &completedAt
				}
			}
			database.DB.Model(&models.SecurityScanTask{}).Where("id = ?", taskID).Updates(updates)
			return
		}

		progress := int((float64(i+1) / float64(len(cleanURLs))) * 20)
		UpdateTaskProgress(taskID, progress, i+1, len(cleanURLs), fmt.Sprintf("正在发现 %s 的页面入口...", entryURL))

		discoveredTargets := []DiscoveredTarget{{
			URL:    sanitizeScanURL(entryURL),
			Kind:   "page",
			Depth:  0,
			Source: "entry",
		}}

		if discoveryEnabled {
			session, err := BuildAuthenticatedWebSession(entryURL, config)
			if err != nil {
				_ = UpdateTaskAndCurrentRun(taskID, map[string]interface{}{
					"status":  "failed",
					"message": fmt.Sprintf("登录态建立失败: %s", err),
				})
				return
			}
			rememberSession(entryURL, session)
			switch discoveryMode {
			case "browser":
				discovered, discoveryErr := DiscoverWithBrowser(entryURL, session, discoveryOpts)
				if discoveryErr != nil {
					fmt.Printf("DiscoverWithBrowser failed for %s: %v\n", entryURL, discoveryErr)
					discoveryWarnings = append(discoveryWarnings, phase1WebDiscoveryWarning(entryURL, "browser", "http", discoveryErr))
					if fallback, fallbackErr := DiscoverWebTargets(entryURL, session, discoveryOpts); fallbackErr == nil && len(fallback) > 0 {
						discoveredTargets = fallback
					}
				} else if len(discovered) > 0 {
					discoveredTargets = discovered
				}
			default:
				discovered, discoveryErr := DiscoverWebTargets(entryURL, session, discoveryOpts)
				if discoveryErr != nil {
					fmt.Printf("DiscoverWebTargets failed for %s: %v\n", entryURL, discoveryErr)
				} else if len(discovered) > 0 {
					discoveredTargets = discovered
				}
			}
		}

		for _, item := range discoveredTargets {
			if _, exists := seenTargets[item.URL]; exists {
				continue
			}
			seenTargets[item.URL] = struct{}{}
			if session, ok := sessionByTarget[sanitizeScanURL(entryURL)]; ok {
				rememberSession(item.URL, session)
			}
			discoveryPlan = append(discoveryPlan, item)
		}

		if err := recordWebDiscoveryPhase1(taskID, entryURL, discoveredTargets, config); err != nil {
			fmt.Printf("recordWebDiscoveryPhase1 failed for %s: %v\n", entryURL, err)
		}
	}

	if len(discoveryPlan) == 0 {
		_ = UpdateTaskAndCurrentRun(taskID, map[string]interface{}{
			"status":  "failed",
			"message": "未发现可扫描的 Web 入口",
		})
		return
	}

	verificationPlan, skippedTargets := prioritizeVerificationTargets(config, discoveryPlan, verificationTargetLimit(config))
	if len(verificationPlan) == 0 {
		_ = UpdateTaskAndCurrentRun(taskID, map[string]interface{}{
			"status":  "failed",
			"message": "未发现可验证的 Web 目标",
		})
		return
	}
	ruleOnlyTargets := 0
	for _, item := range verificationPlan {
		if !shouldRunFullNucleiForTarget(config, item) {
			ruleOnlyTargets++
		}
	}

	_ = UpdateTaskAndCurrentRun(taskID, map[string]interface{}{
		"phase":           "verification",
		"target_snapshot": phase1WebTargetSnapshotWithMeta(cleanURLs, verificationPlan, len(discoveryPlan), skippedTargets, ruleOnlyTargets, discoveryWarnings),
	})
	totalURLs := len(verificationPlan)
	startMessage := fmt.Sprintf("发现 %d 个可扫描入口，开始执行 Web 扫描...", totalURLs)
	if skippedTargets > 0 {
		startMessage = fmt.Sprintf("发现 %d 个入口，按优先级扫描前 %d 个，跳过 %d 个低优先级目标", len(discoveryPlan), totalURLs, skippedTargets)
	}
	if ruleOnlyTargets > 0 {
		startMessage += fmt.Sprintf("；%d 个低价值目标仅执行规则检测", ruleOnlyTargets)
	}
	UpdateTaskProgress(taskID, 20, 0, totalURLs, startMessage)

	// 对每个 URL 执行 Nuclei 扫描
	for i, discovered := range verificationPlan {
		if continueScan, msg := CheckTaskStatus(taskID); !continueScan {
			updates := map[string]interface{}{
				"message": msg,
			}
			if status := GetTaskStatus(taskID); status != "" {
				updates["status"] = status
				if status == models.TaskStatusCancelled {
					completedAt := time.Now()
					updates["completed_at"] = &completedAt
				}
			}
			_ = UpdateTaskAndCurrentRun(taskID, updates)
			return
		}

		targetURL := discovered.URL
		scannedCount := i + 1
		progress := 20 + int((float64(scannedCount)/float64(totalURLs))*70)
		UpdateTaskProgress(taskID, progress, scannedCount, totalURLs, fmt.Sprintf("正在扫描 %s (%d/%d)", targetURL, scannedCount, totalURLs))

		var authMode string
		var options []string
		if config != nil {
			authMode = config.AuthMode
			options = config.Options
		}
		fmt.Printf("Running nuclei on %s with options=%v, auth=%s\n", targetURL, options, authMode)
		session := sessionByTarget[sanitizeScanURL(targetURL)]
		if session == nil || len(session.Headers) == 0 {
			if built, sessionErr := BuildAuthenticatedWebSession(targetURL, config); sessionErr != nil {
				_ = UpdateTaskAndCurrentRun(taskID, map[string]interface{}{
					"status":  "failed",
					"message": fmt.Sprintf("登录态建立失败: %s", sessionErr),
				})
				return
			} else {
				session = built
				rememberSession(targetURL, session)
			}
		}

		var nucleiResults []NucleiResult
		if shouldRunFullNucleiForTarget(config, discovered) {
			var err error
			targetTimeout := webVerificationNucleiTimeout(config, discovered)
			fmt.Printf("Running nuclei on %s with command timeout %s\n", targetURL, targetTimeout)
			nucleiResults, err = engine.executeNucleiWithSessionTimeout(targetURL, config, session, targetTimeout)
			if err != nil {
				fmt.Printf("Nuclei scan failed for %s: %v\n", targetURL, err)
			}
		} else {
			fmt.Printf("Skipping full nuclei for low-value web target %s (kind=%s, source=%s)\n", targetURL, discovered.Kind, discovered.Source)
		}
		ruleResults := detectWebRuleFindings(targetURL, session, options)
		if len(ruleResults) > 0 {
			nucleiResults = append(nucleiResults, ruleResults...)
		}

		nmapResult, parseErr := buildWebNmapResultFromURL(targetURL)
		if parseErr != nil {
			fmt.Printf("Failed to parse web target %s: %v\n", targetURL, parseErr)
			UpdateTaskProgress(taskID, progress, scannedCount, totalURLs, fmt.Sprintf("解析目标失败: %s", targetURL))
			continue
		}

		for idx := range nucleiResults {
			if nucleiResults[idx].Host == "" {
				nucleiResults[idx].Host = nmapResult.Host
			}
			if nucleiResults[idx].Port == 0 && len(nmapResult.Ports) > 0 {
				nucleiResults[idx].Port = nmapResult.Ports[0].PortID
			}
		}

		UpdateTaskProgress(taskID, progress, scannedCount, totalURLs, fmt.Sprintf("正在保存 %s 结果...", targetURL))
		if err := engine.SaveScanResult(taskID, nmapResult, nucleiResults); err != nil {
			fmt.Printf("SaveScanResult failed for %s: %v\n", targetURL, err)
			UpdateTaskProgress(taskID, progress, scannedCount, totalURLs, fmt.Sprintf("保存结果失败: %s", targetURL))
			continue
		}

		for _, vuln := range nucleiResults {
			severity := strings.ToLower(vuln.Severity)
			if severity == "critical" || severity == "high" {
				totalHighRisk++
			} else if severity == "medium" {
				totalMediumRisk++
			} else {
				totalLowRisk++
			}
		}

		// 更新进度
		if scannedCount < totalURLs {
			UpdateTaskProgress(taskID, progress, scannedCount, totalURLs, fmt.Sprintf("完成 %s，准备扫描下一个...", targetURL))
		}
	}

	// 全部完成，更新任务状态为完成
	completedAt := time.Now()
	_ = UpdateTaskAndCurrentRun(taskID, map[string]interface{}{
		"status":           "completed",
		"completed_at":     &completedAt,
		"progress":         100,
		"scanned_ips":      totalURLs,
		"high_risk":        totalHighRisk,
		"medium_risk":      totalMediumRisk,
		"low_risk":         totalLowRisk,
		"message":          phase1WebCompletionMessage(len(discoveryPlan), totalURLs, skippedTargets, ruleOnlyTargets, totalHighRisk, totalMediumRisk, totalLowRisk, discoveryWarnings),
		"summary_snapshot": phase1WebSummarySnapshot(cleanURLs, len(discoveryPlan), totalURLs, skippedTargets, ruleOnlyTargets, discoveryMode, discoveryWarnings, totalHighRisk, totalMediumRisk, totalLowRisk, completedAt),
	})
}

// AsyncScan 异步执行扫描任务
func AsyncScan(taskID uint, target string, targetType string, scanType string, webConfig *WebScanConfig) {
	go func() {
		engine := NewScanEngine()

		if continueScan, msg := CheckTaskStatus(taskID); !continueScan {
			updates := map[string]interface{}{
				"message": msg,
			}
			if status := GetTaskStatus(taskID); status != "" {
				updates["status"] = status
				if status == models.TaskStatusCancelled {
					completedAt := time.Now()
					updates["completed_at"] = &completedAt
				}
			}
			_ = UpdateTaskAndCurrentRun(taskID, updates)
			return
		}

		if _, err := createScanRunForTask(taskID); err != nil {
			fmt.Printf("createScanRunForTask failed for task %d: %v\n", taskID, err)
		}

		// 更新任务状态为运行中
		now := time.Now()
		_ = UpdateTaskAndCurrentRun(taskID, map[string]interface{}{
			"status":     "running",
			"started_at": &now,
			"phase":      "discovery",
		})

		// 根据扫描类型处理不同的目标
		// Web 扫描（URL 类型）：直接解析 URL，跳过 Nmap
		if targetType == "url" && scanType == "web" {
			handleWebURLScan(taskID, target, engine, webConfig)
			return
		}

		// 主机/全面扫描：解析 IP 列表
		ips, err := ParseTargetIPs(target, targetType)
		if err != nil || len(ips) == 0 {
			fmt.Printf("Failed to parse target IPs: %v\n", err)
			_ = UpdateTaskAndCurrentRun(taskID, map[string]interface{}{
				"status":  "failed",
				"message": "解析目标失败",
			})
			return
		}

		totalIPs := len(ips)
		_ = UpdateTaskAndCurrentRun(taskID, map[string]interface{}{
			"phase": "discovery",
			"target_snapshot": phase1JSONString(map[string]interface{}{
				"target_type":      targetType,
				"scan_type":        scanType,
				"input_target":     target,
				"expanded_targets": ips,
				"expanded_count":   totalIPs,
			}),
		})
		UpdateTaskProgress(taskID, 0, 0, totalIPs, "开始扫描...")

		var totalHighRisk, totalMediumRisk, totalLowRisk int

		// 根据扫描类型显示不同的提示信息
		var scanModeName string
		switch scanType {
		case "port":
			scanModeName = "端口扫描"
		case "host-vuln":
			scanModeName = "主机漏洞扫描"
		case "web":
			scanModeName = "Web漏洞扫描"
		default:
			scanModeName = "全面扫描"
		}
		fmt.Printf("Starting %s for task %d with %d targets\n", scanModeName, taskID, totalIPs)

		// 逐个扫描 IP
		for i, ip := range ips {
			scannedCount := i + 1

			// 检查任务状态（暂停/取消）
			if continueScan, msg := CheckTaskStatus(taskID); !continueScan {
				fmt.Printf("Task %d %s, stopping scan\n", taskID, msg)
				updates := map[string]interface{}{
					"status":  GetTaskStatus(taskID),
					"message": msg,
				}
				if status, _ := updates["status"].(string); status == models.TaskStatusCancelled {
					completedAt := time.Now()
					updates["completed_at"] = &completedAt
				}
				_ = UpdateTaskAndCurrentRun(taskID, updates)
				return
			}

			// 根据扫描类型计算进度
			nmapProgress, nucleiProgress := scanProgressWeights(scanType)

			progress := int((float64(scannedCount) / float64(totalIPs)) * float64(nmapProgress))
			UpdateTaskProgress(taskID, progress, scannedCount, totalIPs, fmt.Sprintf("正在扫描 %s (%d/%d)", ip, scannedCount, totalIPs))

			var nmapResult *NmapResult
			var nucleiResults []NucleiResult

			// 执行 Nmap 扫描
			nmapResult, err = engine.ExecuteNmap(ip)
			if err != nil {
				fmt.Printf("Nmap scan failed for %s: %v\n", ip, err)
				UpdateTaskProgress(taskID, progress, scannedCount, totalIPs, fmt.Sprintf("Nmap扫描失败: %s", ip))
				continue
			}
			fmt.Printf("Nmap result: host=%s, open_ports=%d\n", nmapResult.Host, len(nmapResult.Ports))

			if scanType == "host-vuln" {
				UpdateTaskProgress(taskID, progress, scannedCount, totalIPs, fmt.Sprintf("正在匹配 %s 服务版本漏洞...", ip))
				high, medium, low, err := matchHostVersionVulnerabilities(taskID, ip, nmapResult.Ports)
				if err != nil {
					fmt.Printf("Version vulnerability matching save failed for %s: %v\n", ip, err)
				} else {
					totalHighRisk += high
					totalMediumRisk += medium
					totalLowRisk += low
				}
			}

			// 根据扫描类型执行漏洞检测
			if scanType == "web" || scanType == "host-vuln" {
				_ = UpdateTaskAndCurrentRun(taskID, map[string]interface{}{"phase": "verification"})
				webTargets, hostTargets := discoverServiceTargets(ip, scanType, nmapResult.Ports)

				// 执行 Web 漏洞扫描（web 或 all 类型）
				if scanType == "web" && len(webTargets) > 0 {
					totalWebTargets := len(webTargets)
					for idx, targetURL := range webTargets {
						nucleiProgressBase := nmapProgress + int((float64(scannedCount)/float64(totalIPs))*float64(nucleiProgress)) + int((float64(idx+1)/float64(totalWebTargets))*20)
						UpdateTaskProgress(taskID, nucleiProgressBase, scannedCount, totalIPs, fmt.Sprintf("正在检测 %s Web漏洞 (%d/%d)...", targetURL, idx+1, totalWebTargets))

						targetResults, _ := engine.ExecuteNuclei(targetURL, webConfig)
						fmt.Printf("Nuclei Web results for %s: %d\n", targetURL, len(targetResults))

						// 将端口信息添加到 nuclei 结果中
						for i := range targetResults {
							parsedURL := targetURL
							if strings.HasPrefix(targetURL, "https://") {
								parsedURL = strings.TrimPrefix(targetURL, "https://")
							} else if strings.HasPrefix(targetURL, "http://") {
								parsedURL = strings.TrimPrefix(targetURL, "http://")
							}
							parts := strings.Split(parsedURL, ":")
							if len(parts) >= 2 {
								targetResults[i].Host = parts[0]
								fmt.Sscanf(parts[1], "%d", &targetResults[i].Port)
							}
						}

						nucleiResults = append(nucleiResults, targetResults...)
					}
				}

				// 执行主机漏洞扫描（host-vuln 或 all 类型）
				if scanType == "host-vuln" && len(hostTargets) > 0 {
					nucleiResults = append(nucleiResults, verifyHostServiceTargets(engine, taskID, ip, scannedCount, totalIPs, nmapProgress, nucleiProgress, hostTargets)...)
				}
			}

			// 如果没有发现任何可扫描的目标
			if len(nmapResult.Ports) == 0 {
				fmt.Printf("No open ports found on %s\n", ip)
			} else if scanType != "port" && len(nucleiResults) == 0 {
				fmt.Printf("No vulnerability scan targets found on %s\n", ip)
			}

			fmt.Printf("Total nuclei results for %s: %d\n", ip, len(nucleiResults))

			// 保存结果
			var saveProgress int
			saveProgress = nmapProgress + nucleiProgress + 10
			fmt.Printf("DEBUG: About to save results for task %d, nmapResult.Ports=%d\n", taskID, len(nmapResult.Ports))
			UpdateTaskProgress(taskID, saveProgress, scannedCount, totalIPs, fmt.Sprintf("正在保存 %s 结果...", ip))

			if err := engine.SaveScanResult(taskID, nmapResult, nucleiResults); err != nil {
				fmt.Printf("SaveScanResult failed for %s: %v\n", ip, err)
				UpdateTaskProgress(taskID, progress, scannedCount, totalIPs, fmt.Sprintf("保存结果失败: %s", ip))
				continue
			}
			fmt.Printf("DEBUG: SaveScanResult completed for task %d\n", taskID)

			// 统计漏洞
			for _, vuln := range nucleiResults {
				severity := strings.ToLower(vuln.Severity)
				if severity == "critical" || severity == "high" {
					totalHighRisk++
				} else if severity == "medium" {
					totalMediumRisk++
				} else {
					totalLowRisk++
				}
			}

			// 更新每个 IP 扫描完成后的进度
			if scannedCount < totalIPs {
				var nextProgress int
				nextProgress = nmapProgress + nucleiProgress
				UpdateTaskProgress(taskID, nextProgress, scannedCount, totalIPs, fmt.Sprintf("完成 %s，准备扫描下一个...", ip))
			}
		}

		// 全部完成，更新任务状态为完成
		completedAt := time.Now()
		_ = UpdateTaskAndCurrentRun(taskID, map[string]interface{}{
			"status":       "completed",
			"completed_at": &completedAt,
			"progress":     100,
			"scanned_ips":  totalIPs,
			"high_risk":    totalHighRisk,
			"medium_risk":  totalMediumRisk,
			"low_risk":     totalLowRisk,
			"message":      fmt.Sprintf("扫描完成，发现 %d 个高危，%d 个中危，%d 个低危漏洞", totalHighRisk, totalMediumRisk, totalLowRisk),
			"summary_snapshot": phase1JSONString(map[string]interface{}{
				"scan_type":    scanType,
				"scanned_ips":  totalIPs,
				"high_risk":    totalHighRisk,
				"medium_risk":  totalMediumRisk,
				"low_risk":     totalLowRisk,
				"completed_at": completedAt,
			}),
		})
	}()
}
