// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package security

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/jenvenson/ops-platform/internal/models"
)

// ============================================
// 模块三：漏洞匹配层 (Vulnerability Matching Module)
// 职责：基于服务指纹匹配 CVE 漏洞
// ============================================

// VulnMatcherConfig 漏洞匹配配置
type VulnMatcherConfig struct {
	AutoEnrich  bool // 是否自动从知识库补全漏洞信息
	StrictMatch bool // 是否严格匹配版本
}

// VulnMatchResult 漏洞匹配结果
type VulnMatchResult struct {
	CVEID         string  // CVE 编号
	CVSS          float64 // CVSS 分数
	Severity      string  // 严重程度
	Title         string  // 漏洞标题
	Description   string  // 漏洞描述
	Solution      string  // 解决方案
	VulnType      string  // 漏洞类型
	MatchedOn     string  // 匹配依据
	MatchMode     string  // 匹配模式: exact, version-range, fuzzy-product
	Confidence    string  // 置信度: high, medium, low
	ExploitPrereq string  // 利用前提
	Service       string  // 匹配的服务
	Version       string  // 匹配的版本
}

func inferExploitPrereq(cveID, title, description string) string {
	content := strings.ToLower(strings.Join([]string{cveID, title, description}, " "))

	switch {
	case strings.Contains(content, "high privileged attacker with network access"):
		return "需要高权限账号并具备网络访问能力，不属于匿名直接利用。"
	case strings.Contains(content, "privileged attacker with network access"):
		return "需要具备账号权限并可通过网络访问目标服务。"
	case strings.Contains(content, "local access"):
		return "需要本地访问权限。"
	case strings.Contains(content, "authenticated"):
		return "需要已认证访问权限。"
	default:
		return ""
	}
}

func hasStructuredVersionConstraint(vuln *models.VulnerabilityDatabase) bool {
	if vuln == nil {
		return false
	}
	return strings.TrimSpace(vuln.AffectedVersion) != "" ||
		strings.TrimSpace(vuln.VersionStartIncluding) != "" ||
		strings.TrimSpace(vuln.VersionStartExcluding) != "" ||
		strings.TrimSpace(vuln.VersionEndIncluding) != "" ||
		strings.TrimSpace(vuln.VersionEndExcluding) != ""
}

func matchModeRank(mode string) int {
	switch strings.TrimSpace(mode) {
	case "exact":
		return 3
	case "version-range":
		return 2
	case "fuzzy-product":
		return 1
	default:
		return 0
	}
}

func confidenceRank(confidence string) int {
	switch strings.TrimSpace(confidence) {
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

// ServiceToProductMap 服务名到产品名的映射
var ServiceToProductMap = map[string][]string{
	"ssh":           {"openssh", "ssh"},
	"http":          {"apache", "nginx", "httpd", "iis", "tomcat"},
	"https":         {"apache", "nginx", "httpd", "iis", "tomcat"},
	"mysql":         {"mysql", "mariadb", "oracle:mysql", "oracle"},
	"redis":         {"redis"},
	"postgres":      {"postgresql", "postgres"},
	"mongodb":       {"mongodb"},
	"nginx":         {"nginx"},
	"apache":        {"apache", "httpd"},
	"httpd":         {"apache", "httpd"},
	"tomcat":        {"tomcat", "apache-tomcat"},
	"jetty":         {"jetty"},
	"elasticsearch": {"elasticsearch"},
	"kafka":         {"kafka"},
	"rabbitmq":      {"rabbitmq"},
	"zookeeper":     {"zookeeper"},
	"docker":        {"docker", "docker-engine"},
	"kubernetes":    {"kubernetes", "k8s"},
	"jenkins":       {"jenkins"},
	"gitlab":        {"gitlab"},
	"jira":          {"jira"},
	"confluence":    {"confluence"},
	"wordpress":     {"wordpress"},
	"drupal":        {"drupal"},
	"joomla":        {"joomla"},
	"oracle":        {"oracle-database", "oracle"},
	"mssql":         {"mssql", "sql-server"},
	"ftp":           {"vsftpd", "proftpd", "ftp"},
	"sftp":          {"openssh", "sftp"},
	"smtp":          {"postfix", "sendmail", "exim", "smtp"},
	"pop3":          {"dovecot", "courier"},
	"imap":          {"dovecot", "courier"},
	"dns":           {"bind", "named", "dns"},
	"ldap":          {"openldap", "ldap"},
	"vnc":           {"vnc", "tightvnc", "realvnc"},
	"rdp":           {"rdp", "xrdp", "windows-terminal"},
	"smb":           {"samba", "smb"},
}

// VulnMatcher 漏洞匹配器
type VulnMatcher struct {
	vulnDB *VulnDBService
	config *VulnMatcherConfig
}

// NewVulnMatcher 创建漏洞匹配器
func NewVulnMatcher() *VulnMatcher {
	matcher := &VulnMatcher{
		vulnDB: nil, // 延迟初始化，避免循环依赖
		config: &VulnMatcherConfig{
			AutoEnrich:  true,
			StrictMatch: false,
		},
	}
	return matcher
}

// getVulnDB 获取漏洞库服务
func (m *VulnMatcher) getVulnDB() *VulnDBService {
	if m.vulnDB == nil {
		m.vulnDB = NewVulnDBService()
	}
	return m.vulnDB
}

// MatchByFingerprint 根据服务指纹匹配漏洞（增强版）
func (m *VulnMatcher) MatchByFingerprint(portInfo *PortInfo) []VulnMatchResult {
	var results []VulnMatchResult

	// 1. 优先基于 CPE 做更精准的产品/版本匹配
	if portInfo.CPE != "" {
		cpeVulns := m.lookupVulnsByCPE(portInfo.CPE, portInfo.Version)
		results = append(results, cpeVulns...)
	}

	// 2. 从本地知识库查找服务/产品/版本匹配的漏洞
	vulns := m.lookupVulnsByNVDWithEvidence(portInfo)
	results = append(results, vulns...)

	// 3. 内置常见漏洞规则（补充知识库）
	builtinVulns := m.lookupBuiltinVulns(portInfo.Service, portInfo.Version)
	results = append(results, builtinVulns...)

	// 4. 去重
	results = m.deduplicateResults(results)

	// 5. 按 CVSS 评分排序
	results = m.sortBySeverity(results)

	return results
}

func (m *VulnMatcher) classifyMatch(vuln *models.VulnerabilityDatabase, portInfo *PortInfo, serviceFallback bool) (string, string) {
	if vuln == nil || portInfo == nil {
		return "", ""
	}

	runtimeCPE := strings.TrimSpace(portInfo.CPE)
	runtimeVersion := strings.TrimSpace(portInfo.Version)
	hasCPEEvidence := runtimeCPE != "" && strings.TrimSpace(vuln.AffectedCPE) != "" && m.cpeMatches(runtimeCPE, vuln.AffectedCPE)
	hasStructuredConstraint := hasStructuredVersionConstraint(vuln)
	hasVersionEvidence := runtimeVersion != "" && hasStructuredConstraint

	switch {
	case serviceFallback:
		return "fuzzy-product", "low"
	case hasCPEEvidence && !hasStructuredConstraint:
		return "exact", "high"
	case hasVersionEvidence:
		return "version-range", "high"
	case hasCPEEvidence:
		return "fuzzy-product", "medium"
	default:
		return "fuzzy-product", "medium"
	}
}

func annotateMatchedOn(matchedOn, matchMode string, portInfo *PortInfo) string {
	if portInfo == nil {
		return matchedOn
	}

	version := strings.TrimSpace(portInfo.Version)
	if version == "" {
		return matchedOn
	}

	if strings.Contains(matchedOn, "版本 ") {
		return matchedOn
	}

	if matchMode == "exact" || matchMode == "version-range" {
		return fmt.Sprintf("%s (版本: %s)", matchedOn, version)
	}

	return matchedOn
}

func (m *VulnMatcher) newVulnMatchResult(vuln models.VulnerabilityDatabase, portInfo *PortInfo, matchedOn string, serviceFallback bool) VulnMatchResult {
	matchMode, confidence := "", ""
	if portInfo != nil {
		matchMode, confidence = m.classifyMatch(&vuln, portInfo, serviceFallback)
		matchedOn = annotateMatchedOn(matchedOn, matchMode, portInfo)
	}

	service := ""
	version := ""
	if portInfo != nil {
		service = portInfo.Service
		version = portInfo.Version
	}

	return VulnMatchResult{
		CVEID:         vuln.CVEID,
		CVSS:          vuln.CVSSScore,
		Severity:      vuln.Severity,
		Title:         vuln.Title,
		Description:   vuln.Description,
		Solution:      vuln.Solution,
		VulnType:      vuln.VulnType,
		MatchedOn:     matchedOn,
		MatchMode:     matchMode,
		Confidence:    confidence,
		ExploitPrereq: inferExploitPrereq(vuln.CVEID, vuln.Title, vuln.Description),
		Service:       service,
		Version:       version,
	}
}

func (m *VulnMatcher) hasStrongProductEvidence(portInfo *PortInfo, term string) bool {
	term = strings.ToLower(strings.TrimSpace(term))
	if term == "" {
		return false
	}

	product := strings.ToLower(portInfo.Product)
	banner := strings.ToLower(portInfo.Banner)
	cpe := strings.ToLower(portInfo.CPE)
	service := strings.ToLower(portInfo.Service)

	// 对容易泛化的 Web 服务做收紧，必须有产品证据，不能只靠 http/https 服务名。
	if service == "http" || service == "https" {
		return strings.Contains(product, term) || strings.Contains(cpe, term)
	}

	if strings.Contains(product, term) || strings.Contains(banner, term) || strings.Contains(cpe, term) {
		return true
	}

	// 非 Web 服务保留有限的服务名回退能力，避免 mysql/redis/ssh 这类明确协议被误杀。
	return service == term
}

func (m *VulnMatcher) lookupVulnsByCPE(cpe, fallbackVersion string) []VulnMatchResult {
	var results []VulnMatchResult

	vendor, product, version := parseCPE(cpe)
	if product == "" {
		return results
	}
	if version == "" {
		version = fallbackVersion
	}

	searchTerms := []string{product}
	if vendor != "" {
		searchTerms = append(searchTerms, vendor+":"+product, vendor)
	}

	vulnDB := m.getVulnDB()
	if vulnDB == nil {
		return results
	}

	seen := make(map[string]bool)
	for _, term := range searchTerms {
		vulns, err := vulnDB.SearchByProduct(term, 20)
		if err != nil {
			continue
		}
		for _, vuln := range vulns {
			if !m.matchesVulnerabilityByFingerprint(&vuln, cpe, version) {
				continue
			}
			if seen[vuln.CVEID] {
				continue
			}
			seen[vuln.CVEID] = true
			result := m.newVulnMatchResult(vuln, &PortInfo{
				Service: product,
				Version: version,
				CPE:     cpe,
			}, fmt.Sprintf("CPE匹配: %s", cpe), false)
			result.Service = product
			result.Version = version
			results = append(results, result)
		}
	}

	return results
}

func parseCPE(cpe string) (vendor, product, version string) {
	cpe = strings.ToLower(strings.TrimSpace(cpe))
	if cpe == "" {
		return "", "", ""
	}

	// Legacy URI binding, e.g. cpe:/a:mysql:mysql:8.0.20
	if strings.HasPrefix(cpe, "cpe:/") {
		trimmed := strings.TrimPrefix(cpe, "cpe:/")
		parts := strings.Split(trimmed, ":")
		if len(parts) < 3 {
			return "", "", ""
		}

		vendor = strings.TrimSpace(parts[1])
		product = strings.TrimSpace(parts[2])
		if len(parts) >= 4 {
			version = strings.TrimSpace(parts[3])
		}
		return vendor, product, version
	}

	// Formatted string binding, e.g. cpe:2.3:a:oracle:mysql:*:*:*:*:*:*:*:*
	parts := strings.Split(cpe, ":")
	if len(parts) < 6 {
		return "", "", ""
	}
	return strings.ToLower(parts[3]), strings.ToLower(parts[4]), strings.ToLower(parts[5])
}

// lookupVulnsByNVD 从 NVD 知识库查找漏洞（增强版）
func (m *VulnMatcher) lookupVulnsByNVD(service, product, version string) []VulnMatchResult {
	var results []VulnMatchResult

	// 准备搜索关键字
	searchTerms := m.getSearchTerms(service, product, version)

	// 查询漏洞库
	vulnDB := m.getVulnDB()
	if vulnDB == nil {
		return results
	}

	for _, term := range searchTerms {
		// 根据产品名搜索
		vulns, err := vulnDB.SearchByProduct(term, 20)
		if err != nil {
			continue
		}
		for _, vuln := range vulns {
			if !matchesProductFamily(&vuln, term) {
				continue
			}
			// 版本匹配检查
			if version != "" && vuln.AffectedVersion != "" {
				if !m.isVersionInRange(version, vuln.AffectedVersion) {
					continue // 版本不在受影响范围内
				}
			}

			result := m.newVulnMatchResult(vuln, &PortInfo{
				Service: service,
				Product: product,
				Version: version,
			}, fmt.Sprintf("产品匹配: %s", term), false)
			result.Service = service
			result.Version = version
			results = append(results, result)
		}

		// 根据服务名搜索
		vulns = vulnDB.GetVulnByService(service, term, version)
		for _, vuln := range vulns {
			if !matchesProductFamily(&vuln, term) {
				continue
			}
			result := m.newVulnMatchResult(vuln, &PortInfo{
				Service: service,
				Product: product,
				Version: version,
			}, fmt.Sprintf("服务匹配: %s", service), true)
			result.Service = service
			result.Version = version
			results = append(results, result)
		}
	}

	return results
}

func (m *VulnMatcher) lookupVulnsByNVDWithEvidence(portInfo *PortInfo) []VulnMatchResult {
	var results []VulnMatchResult

	searchTerms := m.getSearchTermsForPortInfo(portInfo)
	vulnDB := m.getVulnDB()
	if vulnDB == nil {
		return results
	}

	for _, term := range searchTerms {
		strongEvidence := m.hasStrongProductEvidence(portInfo, term)
		if !strongEvidence {
			continue
		}

		vulns, err := vulnDB.SearchByProduct(term, 20)
		if err != nil {
			continue
		}
		for _, vuln := range vulns {
			productFamilyMatch := matchesProductFamily(&vuln, term)
			fingerprintMatch := m.matchesVulnerabilityByFingerprint(&vuln, portInfo.CPE, portInfo.Version)
			if !productFamilyMatch {
				continue
			}
			if !fingerprintMatch {
				continue
			}

			results = append(results, m.newVulnMatchResult(vuln, portInfo, fmt.Sprintf("产品匹配: %s", term), false))
		}

		if m.shouldSkipServiceFallback(portInfo) {
			continue
		}

		vulns = vulnDB.GetVulnByService(portInfo.Service, term, portInfo.Version)
		for _, vuln := range vulns {
			if !matchesProductFamily(&vuln, term) {
				continue
			}
			if !m.matchesVulnerabilityByFingerprint(&vuln, portInfo.CPE, portInfo.Version) {
				continue
			}

			results = append(results, m.newVulnMatchResult(vuln, portInfo, fmt.Sprintf("服务匹配: %s", portInfo.Service), true))
		}
	}

	return results
}

func matchesProductFamily(vuln *models.VulnerabilityDatabase, term string) bool {
	if vuln == nil {
		return false
	}

	term = normalizeProductToken(term)
	if term == "" {
		return false
	}

	candidates := []string{
		vuln.Product,
		vuln.Vendor,
		vuln.AffectedProduct,
		vuln.AffectedCPE,
	}

	for _, candidate := range candidates {
		for _, token := range extractComparableProductTokens(candidate) {
			if token == term {
				return true
			}
		}
	}

	return false
}

func extractComparableProductTokens(raw string) []string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return nil
	}

	seen := make(map[string]bool)
	var tokens []string

	if strings.HasPrefix(raw, "cpe:") {
		if vendor, product, _ := parseCPE(raw); product != "" {
			candidates := []string{product}
			if vendor != "" {
				candidates = append(candidates, vendor, vendor+":"+product)
			}
			for _, candidate := range candidates {
				token := normalizeProductToken(candidate)
				if token == "" || seen[token] {
					continue
				}
				seen[token] = true
				tokens = append(tokens, token)
			}
			return tokens
		}
	}

	for _, part := range strings.FieldsFunc(raw, func(r rune) bool {
		return r == ':' || r == '/' || r == '|' || r == ' ' || r == '\t' || r == '\n'
	}) {
		token := normalizeProductToken(part)
		if token == "" || seen[token] {
			continue
		}
		seen[token] = true
		tokens = append(tokens, token)
	}
	return tokens
}

func normalizeProductToken(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return ""
	}

	replacer := strings.NewReplacer("_", "", "-", "", ".", "", ":", "", "/", "", " ", "")
	return replacer.Replace(raw)
}

func (m *VulnMatcher) getSearchTermsForPortInfo(portInfo *PortInfo) []string {
	if portInfo == nil {
		return nil
	}

	service := strings.ToLower(strings.TrimSpace(portInfo.Service))
	product := strings.ToLower(strings.TrimSpace(portInfo.Product))
	cpe := strings.ToLower(strings.TrimSpace(portInfo.CPE))
	terms := make(map[string]bool)

	if cpe != "" {
		if vendor, cpeProduct, _ := parseCPE(cpe); cpeProduct != "" {
			terms[cpeProduct] = true
			if vendor != "" {
				terms[vendor] = true
				terms[vendor+":"+cpeProduct] = true
			}
		}
	}

	if service == "mysql" || strings.Contains(product, "mysql") || strings.Contains(cpe, ":mysql:") {
		terms["mysql"] = true
		terms["oracle"] = true
		terms["oracle:mysql"] = true
	}

	if (service == "http" || service == "https") && (product != "" || cpe != "") {
		if vendor, cpeProduct, _ := parseCPE(cpe); cpeProduct != "" {
			terms[cpeProduct] = true
			if vendor != "" {
				terms[vendor] = true
				terms[vendor+":"+cpeProduct] = true
			}
		}

		for _, token := range tokenizeProductEvidence(product) {
			terms[token] = true
		}

		var result []string
		for term := range terms {
			if term != "" {
				result = append(result, term)
			}
		}
		return result
	}

	if len(terms) > 0 {
		for _, term := range m.getSearchTerms(portInfo.Service, portInfo.Product, portInfo.Version) {
			terms[term] = true
		}
		var result []string
		for term := range terms {
			if term != "" {
				result = append(result, term)
			}
		}
		return result
	}

	return m.getSearchTerms(portInfo.Service, portInfo.Product, portInfo.Version)
}

func tokenizeProductEvidence(product string) []string {
	splitter := regexp.MustCompile(`[^a-z0-9]+`)
	rawTokens := splitter.Split(strings.ToLower(strings.TrimSpace(product)), -1)
	stopwords := map[string]bool{
		"http": true, "https": true, "server": true, "compatible": true,
		"object": true, "store": true, "service": true,
	}

	var tokens []string
	for _, token := range rawTokens {
		if len(token) < 3 || stopwords[token] {
			continue
		}
		tokens = append(tokens, token)
	}
	return tokens
}

func (m *VulnMatcher) shouldSkipServiceFallback(portInfo *PortInfo) bool {
	if portInfo == nil {
		return false
	}

	service := strings.ToLower(strings.TrimSpace(portInfo.Service))
	if service != "http" && service != "https" {
		return false
	}

	return strings.TrimSpace(portInfo.Product) != "" || strings.TrimSpace(portInfo.CPE) != ""
}

func (m *VulnMatcher) matchesVulnerabilityByFingerprint(vuln *models.VulnerabilityDatabase, runtimeCPE, runtimeVersion string) bool {
	if vuln == nil {
		return false
	}

	runtimeCPE = strings.ToLower(strings.TrimSpace(runtimeCPE))
	runtimeVersion = strings.TrimSpace(runtimeVersion)
	if runtimeVersion == "" && runtimeCPE != "" {
		_, _, runtimeVersion = parseCPE(runtimeCPE)
	}

	if vuln.AffectedCPE != "" && runtimeCPE != "" {
		if !m.cpeMatches(runtimeCPE, vuln.AffectedCPE) {
			return false
		}
	}

	return m.versionMatches(runtimeVersion, vuln)
}

func (m *VulnMatcher) cpeMatches(runtimeCPE, vulnCPE string) bool {
	rVendor, rProduct, _ := parseCPE(runtimeCPE)
	vVendor, vProduct, _ := parseCPE(strings.ToLower(vulnCPE))

	if vVendor != "" && rVendor != "" && rVendor != vVendor {
		// Nmap often reports legacy mysql CPE as cpe:/a:mysql:mysql:x,
		// while NVD uses cpe:2.3:a:oracle:mysql:*...
		mysqlVendorCompatible := rProduct == "mysql" && vProduct == "mysql" &&
			((rVendor == "mysql" && vVendor == "oracle") || (rVendor == "oracle" && vVendor == "mysql"))
		nginxVendorCompatible := rProduct == "nginx" && vProduct == "nginx" &&
			((rVendor == "nginx" && vVendor == "f5") || (rVendor == "f5" && vVendor == "nginx"))
		if !(mysqlVendorCompatible || nginxVendorCompatible) {
			return false
		}
	}
	if vVendor != "" && rVendor == "" {
		return false
	}
	if vProduct != "" && rProduct != vProduct {
		return false
	}
	return true
}

func (m *VulnMatcher) versionMatches(runtimeVersion string, vuln *models.VulnerabilityDatabase) bool {
	if vuln == nil {
		return false
	}
	if runtimeVersion == "" {
		return vuln.AffectedVersion == "" &&
			vuln.VersionStartIncluding == "" &&
			vuln.VersionStartExcluding == "" &&
			vuln.VersionEndIncluding == "" &&
			vuln.VersionEndExcluding == ""
	}

	if vuln.VersionStartIncluding != "" && !m.isVersionGreaterOrEqual(runtimeVersion, vuln.VersionStartIncluding) {
		return false
	}
	if vuln.VersionStartExcluding != "" && !m.isVersionGreaterThan(runtimeVersion, vuln.VersionStartExcluding) {
		return false
	}
	if vuln.VersionEndIncluding != "" && !m.isVersionLessOrEqual(runtimeVersion, vuln.VersionEndIncluding) {
		return false
	}
	if vuln.VersionEndExcluding != "" && !m.isVersionLessThan(runtimeVersion, vuln.VersionEndExcluding) {
		return false
	}

	if vuln.VersionStartIncluding != "" || vuln.VersionStartExcluding != "" || vuln.VersionEndIncluding != "" || vuln.VersionEndExcluding != "" {
		return true
	}

	if vuln.AffectedVersion != "" {
		return m.isVersionInRange(runtimeVersion, vuln.AffectedVersion)
	}

	return true
}

// getSearchTerms 获取搜索关键字列表
func (m *VulnMatcher) getSearchTerms(service, product, version string) []string {
	terms := make(map[string]bool)

	// 添加服务名
	if service != "" {
		terms[strings.ToLower(service)] = true

		// 添加服务名对应的产品名
		if products, ok := ServiceToProductMap[strings.ToLower(service)]; ok {
			for _, p := range products {
				terms[p] = true
			}
		}
	}

	// 添加产品名
	if product != "" {
		terms[strings.ToLower(product)] = true
	}

	// 添加常见产品名
	lowerService := strings.ToLower(service)
	if mapped, ok := ServiceToProductMap[lowerService]; ok {
		for _, p := range mapped {
			terms[p] = true
		}
	}

	// 转换为切片
	var result []string
	for term := range terms {
		result = append(result, term)
	}

	return result
}

// isVersionInRange 检查版本是否在受影响范围内
func (m *VulnMatcher) isVersionInRange(currentVersion, affectedRange string) bool {
	// 简单版本范围匹配
	// 格式: "1.0 to 2.0" 或 ">=1.0 <3.0" 或 "<=2.0"
	affectedRange = strings.ToLower(affectedRange)

	// 处理 "to" 格式
	if strings.Contains(affectedRange, " to ") {
		parts := strings.Split(affectedRange, " to ")
		if len(parts) == 2 {
			start := strings.TrimSpace(parts[0])
			end := strings.TrimSpace(parts[1])
			return m.isVersionGreaterOrEqual(currentVersion, start) &&
				m.isVersionLessOrEqual(currentVersion, end)
		}
	}

	// 处理 >= < 格式
	hasGreaterEqual := strings.Contains(affectedRange, ">=")
	hasLessEqual := strings.Contains(affectedRange, "<=")
	hasGreater := strings.Contains(affectedRange, ">")
	hasLess := strings.Contains(affectedRange, "<")

	// 简单处理：检查版本是否在范围内
	if hasGreaterEqual && hasLessEqual {
		// >=x <=y
		re := regexp.MustCompile(`>=(\S+)\s*<=(\S+)`)
		if matches := re.FindStringSubmatch(affectedRange); len(matches) == 3 {
			return m.isVersionGreaterOrEqual(currentVersion, matches[1]) &&
				m.isVersionLessOrEqual(currentVersion, matches[2])
		}
	}

	if hasGreater && hasLess {
		// >x <y
		re := regexp.MustCompile(`>(\S+)\s*<(\S+)`)
		if matches := re.FindStringSubmatch(affectedRange); len(matches) == 3 {
			return m.isVersionGreaterThan(currentVersion, matches[1]) &&
				m.isVersionLessThan(currentVersion, matches[2])
		}
	}

	return false
}

// isVersionGreaterOrEqual 检查 version >= target
func (m *VulnMatcher) isVersionGreaterOrEqual(version, target string) bool {
	return !m.isVersionLessThan(version, target)
}

// isVersionLessOrEqual 检查 version <= target
func (m *VulnMatcher) isVersionLessOrEqual(version, target string) bool {
	return !m.isVersionGreaterThan(version, target)
}

// deduplicateResults 去重
func (m *VulnMatcher) deduplicateResults(results []VulnMatchResult) []VulnMatchResult {
	uniqueByCVE := make(map[string]VulnMatchResult)

	for _, result := range results {
		existing, ok := uniqueByCVE[result.CVEID]
		if !ok {
			uniqueByCVE[result.CVEID] = result
			continue
		}

		existingModeRank := matchModeRank(existing.MatchMode)
		currentModeRank := matchModeRank(result.MatchMode)
		if currentModeRank > existingModeRank {
			uniqueByCVE[result.CVEID] = result
			continue
		}
		if currentModeRank < existingModeRank {
			continue
		}

		existingConfidenceRank := confidenceRank(existing.Confidence)
		currentConfidenceRank := confidenceRank(result.Confidence)
		if currentConfidenceRank > existingConfidenceRank {
			uniqueByCVE[result.CVEID] = result
			continue
		}
		if currentConfidenceRank < existingConfidenceRank {
			continue
		}

		if result.CVSS > existing.CVSS {
			uniqueByCVE[result.CVEID] = result
		}
	}

	unique := make([]VulnMatchResult, 0, len(uniqueByCVE))
	for _, result := range uniqueByCVE {
		unique = append(unique, result)
	}
	return unique
}

// sortBySeverity 按严重程度排序
func (m *VulnMatcher) sortBySeverity(results []VulnMatchResult) []VulnMatchResult {
	severityOrder := map[string]int{
		"critical": 4,
		"high":     3,
		"medium":   2,
		"low":      1,
		"unknown":  0,
	}

	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			si := severityOrder[results[i].Severity]
			sj := severityOrder[results[j].Severity]
			mi := matchModeRank(results[i].MatchMode)
			mj := matchModeRank(results[j].MatchMode)
			ci := confidenceRank(results[i].Confidence)
			cj := confidenceRank(results[j].Confidence)
			if si < sj ||
				(si == sj && mi < mj) ||
				(si == sj && mi == mj && ci < cj) ||
				(si == sj && mi == mj && ci == cj && results[i].CVSS < results[j].CVSS) {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return results
}

// lookupVulnsByService 从知识库查找漏洞
func (m *VulnMatcher) lookupVulnsByService(service, version string) []VulnMatchResult {
	var results []VulnMatchResult

	// 从 CommonVulnMapping 查找匹配的漏洞
	for cveID, vuln := range models.CommonVulnMapping {
		// 检查是否匹配当前服务
		if strings.Contains(strings.ToLower(vuln.Title), strings.ToLower(service)) ||
			strings.Contains(strings.ToLower(vuln.Title), "openssh") && service == "ssh" ||
			strings.Contains(strings.ToLower(vuln.Title), "apache") && (service == "http" || service == "apache") {

			results = append(results, VulnMatchResult{
				CVEID:       cveID,
				CVSS:        vuln.CvssScore,
				Severity:    vuln.Severity,
				Title:       vuln.Title,
				Description: vuln.Description,
				Solution:    vuln.Solution,
				VulnType:    vuln.VulnType,
				MatchedOn:   fmt.Sprintf("服务匹配: %s", service),
				MatchMode:   "fuzzy-product",
				Confidence:  "low",
				Service:     service,
				Version:     version,
			})
		}
	}

	return results
}

// lookupBuiltinVulns 内置漏洞规则
func (m *VulnMatcher) lookupBuiltinVulns(service, version string) []VulnMatchResult {
	var results []VulnMatchResult

	// OpenSSH 版本漏洞规则
	if strings.ToLower(service) == "ssh" {
		// CVE-2023-28531: OpenSSH < 9.3
		if m.isVersionLessThan(version, "9.3") {
			results = append(results, VulnMatchResult{
				CVEID:       "CVE-2023-28531",
				CVSS:        5.3,
				Severity:    "medium",
				Title:       "OpenSSH ssh-add 智能卡密钥约束漏洞",
				Description: "ssh-add 在 OpenSSH 9.3 之前版本中添加智能卡密钥时未按预期添加每跳目标约束",
				Solution:    "升级 OpenSSH 至 9.3 或更高版本",
				VulnType:    "auth-bypass",
				MatchedOn:   fmt.Sprintf("版本 %s < 9.3", version),
				MatchMode:   "version-range",
				Confidence:  "high",
				Service:     "ssh",
				Version:     version,
			})
		}

		// CVE-2023-48795: Terrapin 攻击 (OpenSSH < 9.6)
		if m.isVersionLessThan(version, "9.6") {
			results = append(results, VulnMatchResult{
				CVEID:       "CVE-2023-48795",
				CVSS:        5.9,
				Severity:    "medium",
				Title:       "OpenSSH Terrapin 攻击漏洞",
				Description: "OpenSSH 存在前缀截断攻击漏洞，攻击者可能利用此漏洞降级算法和功能",
				Solution:    "升级 OpenSSH 至 9.6 或更高版本",
				VulnType:    "auth-bypass",
				MatchedOn:   fmt.Sprintf("版本 %s < 9.6", version),
				MatchMode:   "version-range",
				Confidence:  "high",
				Service:     "ssh",
				Version:     version,
			})
		}

		// CVE-2023-38408: OpenSSH < 9.3p1
		if m.isVersionLessThan(version, "9.3") && m.parseVersionPatch(version) < 1 {
			results = append(results, VulnMatchResult{
				CVEID:       "CVE-2023-38408",
				CVSS:        9.8,
				Severity:    "critical",
				Title:       "OpenSSH 远程代码执行漏洞",
				Description: "OpenSSH sshd 存在远程代码执行漏洞，攻击者可利用此漏洞获取系统权限",
				Solution:    "升级 OpenSSH 至 9.3p1 或更高版本",
				VulnType:    "rce",
				MatchedOn:   fmt.Sprintf("版本 %s < 9.3p1", version),
				MatchMode:   "version-range",
				Confidence:  "high",
				Service:     "ssh",
				Version:     version,
			})
		}
	}

	// Nginx 版本漏洞规则
	if strings.ToLower(service) == "nginx" {
		// CVE-2021-23017
		if m.isVersionLessThan(version, "1.20.1") {
			results = append(results, VulnMatchResult{
				CVEID:       "CVE-2021-23017",
				CVSS:        9.8,
				Severity:    "critical",
				Title:       "Nginx 解析漏洞",
				Description: "Nginx 存在解析漏洞，可能导致远程代码执行",
				Solution:    "升级 Nginx 至 1.20.1 或 1.21.0 或更高版本",
				VulnType:    "rce",
				MatchedOn:   fmt.Sprintf("版本 %s < 1.20.1", version),
				MatchMode:   "version-range",
				Confidence:  "high",
				Service:     "nginx",
				Version:     version,
			})
		}
	}

	return results
}

// isVersionLessThan 检查 version < target
func (m *VulnMatcher) isVersionLessThan(version, target string) bool {
	v1 := m.parseVersion(version)
	v2 := m.parseVersion(target)

	for i := 0; i < len(v1) || i < len(v2); i++ {
		n1 := 0
		n2 := 0
		if i < len(v1) {
			n1 = v1[i]
		}
		if i < len(v2) {
			n2 = v2[i]
		}
		if n1 < n2 {
			return true
		}
		if n1 > n2 {
			return false
		}
	}
	return false
}

// isVersionGreaterThan 检查 version > target
func (m *VulnMatcher) isVersionGreaterThan(version, target string) bool {
	v1 := m.parseVersion(version)
	v2 := m.parseVersion(target)

	for i := 0; i < len(v1) || i < len(v2); i++ {
		n1 := 0
		n2 := 0
		if i < len(v1) {
			n1 = v1[i]
		}
		if i < len(v2) {
			n2 = v2[i]
		}
		if n1 > n2 {
			return true
		}
		if n1 < n2 {
			return false
		}
	}
	return false
}

// parseVersion 解析版本号为整数数组
func (m *VulnMatcher) parseVersion(version string) []int {
	// 清理版本字符串，移除后缀如 "p1", "Ubuntu", 等
	re := regexp.MustCompile(`[^\d.]`)
	cleaned := re.ReplaceAllString(version, " ")

	// 移除空格
	cleaned = strings.TrimSpace(cleaned)

	// 按点分割
	parts := strings.Split(cleaned, ".")
	var nums []int
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		var num int
		fmt.Sscanf(part, "%d", &num)
		nums = append(nums, num)
	}

	return nums
}

// parseVersionPatch 解析 patch 版本号 (如 8.9p1 -> 1)
func (m *VulnMatcher) parseVersionPatch(version string) int {
	// 查找 p 后面的数字
	re := regexp.MustCompile(`p(\d+)`)
	matches := re.FindStringSubmatch(version)
	if len(matches) >= 2 {
		var patch int
		fmt.Sscanf(matches[1], "%d", &patch)
		return patch
	}
	return 0
}

// EnrichWithVulnDB 使用知识库补全漏洞信息
func (m *VulnMatcher) EnrichWithVulnDB(result *VulnMatchResult) {
	if m.vulnDB == nil || result.CVEID == "" {
		return
	}

	enrichment := m.vulnDB.EnrichVulnerability(result.CVEID)
	if enrichment == nil {
		return
	}

	if result.Title == "" {
		result.Title = enrichment.Title
	}
	if result.Solution == "" {
		result.Solution = enrichment.Solution
	}
	if result.VulnType == "" {
		result.VulnType = enrichment.VulnType
	}
	if result.CVSS == 0 {
		result.CVSS = enrichment.CVSSScore
	}
	if result.Severity == "" {
		result.Severity = enrichment.Severity
	}
}

// MatchMultiplePorts 对多个端口进行漏洞匹配
func (m *VulnMatcher) MatchMultiplePorts(ports []PortInfo) map[int][]VulnMatchResult {
	results := make(map[int][]VulnMatchResult)

	for _, port := range ports {
		if strings.ToLower(port.State) != "open" {
			continue
		}

		vulns := m.MatchByFingerprint(&port)
		if len(vulns) > 0 {
			results[port.PortID] = vulns
		}
	}

	return results
}

// ToVulnerability 转换为数据库模型
func (r *VulnMatchResult) ToVulnerability(ip string, port int, taskID uint, assetID uint) models.SecurityVulnerability {
	// 确定优先级
	priority := "优先级低"
	if r.Severity == "critical" || r.Severity == "high" {
		priority = "优先级高"
	} else if r.Severity == "medium" {
		priority = "优先级中"
	}

	vuln := models.SecurityVulnerability{
		AssetID:       assetID,
		IP:            ip,
		Port:          port,
		Protocol:      "tcp",
		Severity:      strings.ToLower(r.Severity),
		CVSSScore:     r.CVSS,
		CVEID:         r.CVEID,
		Title:         r.Title,
		Description:   r.Description,
		VulnType:      r.VulnType,
		Solution:      r.Solution,
		MatchedOn:     r.MatchedOn,
		ExploitPrereq: r.ExploitPrereq,
		Scanner:       "vuln-matcher",
		VulnURL:       fmt.Sprintf("%s:%d", ip, port),
		FindingSource: "host-version-match",
		FindingFamily: "vulnerability",
		Confidence:    r.Confidence,
		PrimaryCVEID:  primaryCVEIDFromString(r.CVEID),
		MatchMode:     r.MatchMode,
		Priority:      priority,
		FalsePositive: false,
	}
	assignVulnerabilityTaskTracking(&vuln, taskID)
	return vuln
}