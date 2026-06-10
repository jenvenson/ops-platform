// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package security

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/jenvenson/ops-platform/internal/database"
	"github.com/jenvenson/ops-platform/internal/models"
)

// NVD API 配置
const (
	NVDApiBaseURL     = "https://services.nvd.nist.gov/rest/json/cves/2.0"
	NVDRequestTimeout = 30 * time.Second
	BatchInsertSize   = 1000
	MaxRetries        = 3
	RateLimitDelay    = 6 * time.Second // NVD API 要求每秒不超过5个请求
	NVDUserAgent      = "ops-platform-dev/1.0"
	MaxNVDDateRange   = 120 * 24 * time.Hour
)

// NVD API 响应结构
type NVDCVEResponse struct {
	ResultsPerPage  int          `json:"resultsPerPage"`
	StartIndex      int          `json:"startIndex"`
	TotalResults    int          `json:"totalResults"`
	Vulnerabilities []NVDCVEItem `json:"vulnerabilities"`
}

type NVDCVEItem struct {
	CVE NVDCVEInfo `json:"cve"`
}

type NVDTime struct {
	time.Time
}

func (t *NVDTime) UnmarshalJSON(data []byte) error {
	raw := strings.Trim(string(data), "\"")
	if raw == "" || raw == "null" {
		t.Time = time.Time{}
		return nil
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05.000",
		"2006-01-02T15:04:05",
	}

	for _, layout := range layouts {
		parsed, err := time.Parse(layout, raw)
		if err == nil {
			t.Time = parsed
			return nil
		}
	}

	return fmt.Errorf("unsupported NVD time format: %s", raw)
}

type NVDCVEInfo struct {
	ID             string             `json:"id"`
	Descriptions   []CVEDescription   `json:"descriptions"`
	Metrics        CVEMetrics         `json:"metrics"`
	Weaknesses     []CVEWeakness      `json:"weaknesses"`
	Configurations []CVEConfiguration `json:"configurations"`
	References     []CVEReference     `json:"references"`
	Published      NVDTime            `json:"published"`
	LastModified   NVDTime            `json:"lastModified"`
}

type CVEDescription struct {
	Lang  string `json:"lang"`
	Value string `json:"value"`
}

type CVEMetrics struct {
	CVSSMetricV31 []CVSSV31 `json:"cvssMetricV31"`
	CVSSMetricV30 []CVSSV30 `json:"cvssMetricV30"`
	CVSSMetricV2  []CVSSV2  `json:"cvssMetricV2"`
}

type CVSSV31 struct {
	CVSSData CVSSData `json:"cvssData"`
}

type CVSSV30 struct {
	CVSSData CVSSData `json:"cvssData"`
}

type CVSSV2 struct {
	CVSSData CVSSDataV2 `json:"cvssData"`
}

type CVSSData struct {
	BaseScore             float64 `json:"baseScore"`
	Severity              string  `json:"baseSeverity"`
	AttackVector          string  `json:"attackVector"`
	AttackComplexity      string  `json:"attackComplexity"`
	PrivilegesRequired    string  `json:"privilegesRequired"`
	UserInteraction       string  `json:"userInteraction"`
	Scope                 string  `json:"scope"`
	ConfidentialityImpact string  `json:"confidentialityImpact"`
	IntegrityImpact       string  `json:"integrityImpact"`
	AvailabilityImpact    string  `json:"availabilityImpact"`
	VectorString          string  `json:"vectorString"`
}

type CVSSDataV2 struct {
	BaseScore    float64 `json:"baseScore"`
	Severity     string  `json:"baseSeverity"`
	AV           string  `json:"AV"`
	AC           string  `json:"AC"`
	AU           string  `json:"AU"`
	C            string  `json:"C"`
	I            string  `json:"I"`
	A            string  `json:"A"`
	VectorString string  `json:"vectorString"`
}

type CVEWeakness struct {
	Description []CWEDescription `json:"description"`
}

type CWEDescription struct {
	Lang  string `json:"lang"`
	Value string `json:"value"`
}

type CVEConfiguration struct {
	Nodes []CVENode `json:"nodes"`
}

type CVENode struct {
	CPEMatch []CPEMatch `json:"cpeMatch"`
}

type CPEMatch struct {
	Vulnerable            bool   `json:"vulnerable"`
	Criteria              string `json:"criteria"`
	VersionStartIncluding string `json:"versionStartIncluding"`
	VersionStartExcluding string `json:"versionStartExcluding"`
	VersionEndIncluding   string `json:"versionEndIncluding"`
	VersionEndExcluding   string `json:"versionEndExcluding"`
}

type CVEReference struct {
	URL    string   `json:"url"`
	Source string   `json:"source"`
	Tags   []string `json:"tags"`
}

// SyncTask 同步任务记录
type SyncTask struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	Source       string     `json:"source" gorm:"size:64"` // nvd, cnvd, cnnvd, nvd-pub
	Status       string     `json:"status" gorm:"size:20"` // running, completed, failed
	TotalCount   int        `json:"total_count"`
	SuccessCount int        `json:"success_count"`
	FailCount    int        `json:"fail_count"`
	StartTime    time.Time  `json:"start_time"`
	EndTime      *time.Time `json:"end_time"`
	ErrorMessage string     `json:"error_message" gorm:"type:text"`
	CreatedAt    time.Time  `json:"created_at"`
}

func (SyncTask) TableName() string {
	return "vuln_sync_tasks"
}

// VulnDBService 漏洞知识库服务（重构版）
type VulnDBService struct {
	db           *gorm.DB
	localCache   sync.Map // 本地缓存，加速查询
	productCache sync.Map // 产品名缓存，用于快速匹配
	rateLimiter  *RateLimiter
	httpClient   *http.Client
	mu           sync.Mutex
}

// RateLimiter 限流器
type RateLimiter struct {
	tokens     int
	maxTokens  int
	refillRate time.Duration
	lastRefill time.Time
	mu         sync.Mutex
}

func NewRateLimiter(maxTokens int, refillRate time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

func (r *RateLimiter) Acquire() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(r.lastRefill)
	if elapsed >= r.refillRate {
		r.tokens = r.maxTokens
		r.lastRefill = now
	}

	if r.tokens > 0 {
		r.tokens--
		return true
	}
	return false
}

// NewVulnDBService 创建漏洞知识库服务
func NewVulnDBService() *VulnDBService {
	httpClient := &http.Client{
		Timeout: NVDRequestTimeout,
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
			ForceAttemptHTTP2:     false,
			MaxIdleConns:          10,
			IdleConnTimeout:       30 * time.Second,
			TLSHandshakeTimeout:   15 * time.Second,
			ResponseHeaderTimeout: 20 * time.Second,
		},
	}

	return &VulnDBService{
		db:          database.DB,
		httpClient:  httpClient,
		rateLimiter: NewRateLimiter(1, RateLimitDelay),
	}
}

func (s *VulnDBService) fetchNVDPage(url string) (*NVDCVEResponse, error) {
	var lastErr error

	for attempt := 1; attempt <= MaxRetries; attempt++ {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", NVDUserAgent)
		req.Close = true

		resp, err := s.httpClient.Do(req)
		if err != nil {
			lastErr = err
		} else {
			body, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			if readErr != nil {
				lastErr = readErr
			} else if resp.StatusCode != http.StatusOK {
				lastErr = fmt.Errorf("NVD API returned status: %d", resp.StatusCode)
			} else {
				var nvdResp NVDCVEResponse
				if err := json.Unmarshal(body, &nvdResp); err != nil {
					lastErr = err
				} else {
					return &nvdResp, nil
				}
			}
		}

		if attempt < MaxRetries {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}

	if fallbackResp, err := s.fetchNVDPageWithWget(url); err == nil {
		return fallbackResp, nil
	} else if lastErr == nil {
		lastErr = err
	}

	return nil, lastErr
}

func (s *VulnDBService) fetchNVDPageWithWget(url string) (*NVDCVEResponse, error) {
	cmd := exec.Command("wget", "-qO-", url)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var nvdResp NVDCVEResponse
	if err := json.Unmarshal(output, &nvdResp); err != nil {
		return nil, err
	}

	return &nvdResp, nil
}

// InitVulnDB 初始化漏洞知识库
func (s *VulnDBService) InitVulnDB() error {
	// 自动创建表
	if err := s.db.AutoMigrate(&models.VulnerabilityDatabase{}, &SyncTask{}); err != nil {
		return err
	}

	// 创建索引以加速查询
	s.createIndexes()

	// 加载到本地缓存
	return s.loadToCache()
}

// createIndexes 创建数据库索引
func (s *VulnDBService) createIndexes() {
	s.createIndexIfMissing("vulnerability_database", "idx_vuln_cve_id", "cve_id")
	s.createIndexIfMissing("vulnerability_database", "idx_vuln_severity", "severity")
	s.createIndexIfMissing("vulnerability_database", "idx_vuln_cvss_score", "cvss_score")
	s.createIndexIfMissing("vulnerability_database", "idx_vuln_source", "source")
	s.createIndexIfMissing("vulnerability_database", "idx_vuln_product", "affected_product")
	s.createIndexIfMissing("vulnerability_database", "idx_vuln_affected_cpe", "affected_cpe")
	s.createIndexIfMissing("vulnerability_database", "idx_vuln_vendor", "vendor")
	s.createIndexIfMissing("vulnerability_database", "idx_vuln_product_name", "product")
}

func (s *VulnDBService) createIndexIfMissing(tableName, indexName, columnName string) {
	var count int64
	s.db.Raw(
		`SELECT COUNT(1)
		 FROM information_schema.statistics
		 WHERE table_schema = DATABASE()
		   AND table_name = ?
		   AND index_name = ?`,
		tableName, indexName,
	).Scan(&count)
	if count > 0 {
		return
	}
	s.db.Exec(fmt.Sprintf("CREATE INDEX %s ON %s(%s)", indexName, tableName, columnName))
}

// loadToCache 加载到本地缓存
func (s *VulnDBService) loadToCache() error {
	var vulns []models.VulnerabilityDatabase
	if err := s.db.Find(&vulns).Error; err != nil {
		return err
	}

	s.localCache = sync.Map{}
	s.productCache = sync.Map{}

	for _, vuln := range vulns {
		if s.shouldExcludeFromMatching(vuln.Source) {
			continue
		}

		s.localCache.Store(vuln.CVEID, vuln)

		// 建立产品名索引
		if vuln.AffectedProduct != "" {
			product := strings.ToLower(vuln.AffectedProduct)
			if existing, ok := s.productCache.Load(product); ok {
				list := existing.([]models.VulnerabilityDatabase)
				s.productCache.Store(product, append(list, vuln))
			} else {
				s.productCache.Store(product, []models.VulnerabilityDatabase{vuln})
			}
		}
	}
	return nil
}

func (s *VulnDBService) shouldExcludeFromMatching(source string) bool {
	return strings.EqualFold(strings.TrimSpace(source), "dev-test")
}

// SyncFromNVD 从 NVD API 同步漏洞数据
func (s *VulnDBService) SyncFromNVD(startIndex, resultsPerPage int) (int, error) {
	// 等待限流
	for !s.rateLimiter.Acquire() {
		time.Sleep(100 * time.Millisecond)
	}

	url := fmt.Sprintf("%s?startIndex=%d&resultsPerPage=%d", NVDApiBaseURL, startIndex, resultsPerPage)

	nvdResp, err := s.fetchNVDPage(url)
	if err != nil {
		return 0, err
	}

	// 转换并插入数据库
	inserted := 0
	for _, item := range nvdResp.Vulnerabilities {
		vuln := s.convertNVDCVE(&item)
		if vuln == nil {
			continue
		}

		// 查找已存在的记录
		var existing models.VulnerabilityDatabase
		result := s.db.Where("cve_id = ?", vuln.CVEID).First(&existing)

		if result.Error == gorm.ErrRecordNotFound {
			if err := s.db.Create(vuln).Error; err == nil {
				inserted++
			}
		} else if result.Error == nil {
			// 更新已存在的记录
			s.db.Model(&existing).Updates(vuln)
		}
	}

	// 刷新缓存
	s.loadToCache()

	return inserted, nil
}

// convertNVDCVE 将 NVD API 响应转换为数据库模型
func (s *VulnDBService) convertNVDCVE(item *NVDCVEItem) *models.VulnerabilityDatabase {
	cve := &item.CVE
	if cve == nil {
		return nil
	}

	vuln := &models.VulnerabilityDatabase{
		CVEID:       cve.ID,
		Source:      "nvd",
		LastUpdated: cve.LastModified.Time,
		CreatedAt:   cve.Published.Time,
	}

	// 提取英文描述
	for _, desc := range cve.Descriptions {
		if desc.Lang == "en" {
			vuln.Description = desc.Value
			break
		}
	}
	if vuln.Description == "" && len(cve.Descriptions) > 0 {
		vuln.Description = cve.Descriptions[0].Value
	}

	// 提取 CVSS 评分
	vuln.CVSSScore, vuln.Severity, vuln.CVSSVector = s.extractCVSS(&cve.Metrics)

	// 提取 CWE
	for _, w := range cve.Weaknesses {
		for _, d := range w.Description {
			if strings.HasPrefix(d.Value, "CWE-") {
				vuln.CWEID = d.Value
				break
			}
		}
		if vuln.CWEID != "" {
			break
		}
	}

	// 提取受影响的产品和版本
	if len(cve.Configurations) > 0 {
		targets := s.extractAffectedTargets(&cve.Configurations)
		if len(targets) > 0 {
			primary := targets[0]
			vuln.AffectedProduct = primary.LegacyProduct
			vuln.AffectedVersion = primary.LegacyVersion
			vuln.AffectedCPE = primary.Criteria
			vuln.Vendor = primary.Vendor
			vuln.Product = primary.Product
			vuln.VersionStartIncluding = primary.VersionStartIncluding
			vuln.VersionStartExcluding = primary.VersionStartExcluding
			vuln.VersionEndIncluding = primary.VersionEndIncluding
			vuln.VersionEndExcluding = primary.VersionEndExcluding
			vuln.RawConfigurations = s.marshalAffectedTargets(targets)
		}
	}

	// 提取漏洞类型
	vuln.VulnType = s.inferVulnType(vuln.CWEID, vuln.Description)

	// 提取修复建议和参考链接
	for _, ref := range cve.References {
		if vuln.PatchURL == "" {
			for _, tag := range ref.Tags {
				if tag == "Patch" || tag == "Vendor Advisory" {
					vuln.PatchURL = ref.URL
					break
				}
			}
		}
		if vuln.References == "" {
			vuln.References = ref.URL
		} else if len(vuln.References) < 1000 {
			vuln.References += "\n" + ref.URL
		}
	}

	// 生成默认标题
	if vuln.Title == "" {
		vuln.Title = cve.ID + " Vulnerability"
	}

	return vuln
}

type affectedTarget struct {
	Criteria              string `json:"criteria"`
	Vendor                string `json:"vendor"`
	Product               string `json:"product"`
	Version               string `json:"version"`
	VersionStartIncluding string `json:"version_start_including"`
	VersionStartExcluding string `json:"version_start_excluding"`
	VersionEndIncluding   string `json:"version_end_including"`
	VersionEndExcluding   string `json:"version_end_excluding"`
	LegacyProduct         string `json:"legacy_product"`
	LegacyVersion         string `json:"legacy_version"`
}

// extractAffectedTargets 提取受影响的产品和版本约束
func (s *VulnDBService) extractAffectedTargets(configs *[]CVEConfiguration) []affectedTarget {
	var results []affectedTarget

	for _, config := range *configs {
		for _, node := range config.Nodes {
			for _, match := range node.CPEMatch {
				if match.Vulnerable {
					criteria := match.Criteria
					// 解析 CPE 格式: cpe:2.3:a:apache:log4j:2.0:*:*:*:*:*:*:*
					parts := strings.Split(criteria, ":")
					if len(parts) >= 6 {
						vendor := parts[3]
						product := parts[4]
						version := parts[5]
						legacyVersion := s.buildLegacyVersionRange(&match, version)
						results = append(results, affectedTarget{
							Criteria:              criteria,
							Vendor:                vendor,
							Product:               product,
							Version:               version,
							VersionStartIncluding: match.VersionStartIncluding,
							VersionStartExcluding: match.VersionStartExcluding,
							VersionEndIncluding:   match.VersionEndIncluding,
							VersionEndExcluding:   match.VersionEndExcluding,
							LegacyProduct:         vendor + ":" + product,
							LegacyVersion:         legacyVersion,
						})
					}
				}
			}
		}
	}
	return results
}

func (s *VulnDBService) buildLegacyVersionRange(match *CPEMatch, fallbackVersion string) string {
	var parts []string
	if match.VersionStartIncluding != "" {
		parts = append(parts, ">="+match.VersionStartIncluding)
	}
	if match.VersionStartExcluding != "" {
		parts = append(parts, ">"+match.VersionStartExcluding)
	}
	if match.VersionEndIncluding != "" {
		parts = append(parts, "<="+match.VersionEndIncluding)
	}
	if match.VersionEndExcluding != "" {
		parts = append(parts, "<"+match.VersionEndExcluding)
	}
	if len(parts) > 0 {
		return strings.Join(parts, " ")
	}
	if fallbackVersion != "" && fallbackVersion != "*" {
		return fallbackVersion
	}
	return ""
}

func (s *VulnDBService) upsertNVDBatch(items []NVDCVEItem) int {
	inserted := 0
	for _, item := range items {
		vuln := s.convertNVDCVE(&item)
		if vuln == nil {
			continue
		}

		var existing models.VulnerabilityDatabase
		result := s.db.Where("cve_id = ?", vuln.CVEID).First(&existing)
		if result.Error == gorm.ErrRecordNotFound {
			if err := s.db.Create(vuln).Error; err == nil {
				inserted++
			}
			continue
		}
		if result.Error == nil {
			s.db.Model(&existing).Updates(vuln)
		}
	}
	return inserted
}

func (s *VulnDBService) syncNVDByURLBuilder(source string, buildURL func(startIndex, resultsPerPage int) string, maxPages int) (int, error) {
	startIndex := 0
	resultsPerPage := 100
	totalInserted := 0
	pageCount := 0

	syncTask := &SyncTask{
		Source:    source,
		Status:    "running",
		StartTime: time.Now(),
	}
	s.db.Create(syncTask)

	for {
		url := buildURL(startIndex, resultsPerPage)
		nvdResp, err := s.fetchNVDPage(url)
		if err != nil {
			syncTask.Status = "failed"
			syncTask.ErrorMessage = err.Error()
			endTime := time.Now()
			syncTask.EndTime = &endTime
			s.db.Save(syncTask)
			return totalInserted, err
		}

		inserted := s.upsertNVDBatch(nvdResp.Vulnerabilities)
		totalInserted += inserted
		startIndex += resultsPerPage
		pageCount++

		syncTask.TotalCount = nvdResp.TotalResults
		syncTask.SuccessCount = totalInserted
		s.db.Save(syncTask)

		if len(nvdResp.Vulnerabilities) < resultsPerPage || startIndex >= nvdResp.TotalResults {
			break
		}
		if maxPages > 0 && pageCount >= maxPages {
			break
		}

		time.Sleep(RateLimitDelay)
	}

	syncTask.Status = "completed"
	endTime := time.Now()
	syncTask.EndTime = &endTime
	s.db.Save(syncTask)
	s.loadToCache()

	return totalInserted, nil
}

func (s *VulnDBService) SyncPublishedRangeNVD(startTime, endTime time.Time) (int, error) {
	totalInserted := 0
	windowStart := startTime.UTC()
	finalEnd := endTime.UTC()

	for !windowStart.After(finalEnd) {
		windowEnd := windowStart.Add(MaxNVDDateRange - time.Millisecond)
		if windowEnd.After(finalEnd) {
			windowEnd = finalEnd
		}

		source := "nvd-pub"
		if windowStart.Year() == windowEnd.Year() {
			source = fmt.Sprintf(
				"nvd-pub-%d-%02d%02d-%02d%02d",
				windowStart.Year(),
				windowStart.Month(),
				windowStart.Day(),
				windowEnd.Month(),
				windowEnd.Day(),
			)
		}

		pubStart := url.QueryEscape(formatNVDAPITime(windowStart))
		pubEnd := url.QueryEscape(formatNVDAPITime(windowEnd))
		inserted, err := s.syncNVDByURLBuilder(source, func(startIndex, resultsPerPage int) string {
			return fmt.Sprintf(
				"%s?pubStartDate=%s&pubEndDate=%s&startIndex=%d&resultsPerPage=%d",
				NVDApiBaseURL,
				pubStart,
				pubEnd,
				startIndex,
				resultsPerPage,
			)
		}, 0)
		totalInserted += inserted
		if err != nil {
			return totalInserted, err
		}

		windowStart = windowEnd.Add(time.Millisecond)
	}

	return totalInserted, nil
}

func (s *VulnDBService) SyncCVEIDs(ids []string) (int, error) {
	cleanIDs := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		cleanIDs = append(cleanIDs, id)
	}
	if len(cleanIDs) == 0 {
		return 0, nil
	}

	syncTask := &SyncTask{
		Source:     "nvd-cve",
		Status:     "running",
		TotalCount: len(cleanIDs),
		StartTime:  time.Now(),
	}
	s.db.Create(syncTask)

	totalInserted := 0
	var failures []string

	for _, cveID := range cleanIDs {
		u := fmt.Sprintf("%s?cveId=%s", NVDApiBaseURL, url.QueryEscape(cveID))
		nvdResp, err := s.fetchNVDPage(u)
		if err != nil {
			syncTask.FailCount++
			failures = append(failures, fmt.Sprintf("%s: %v", cveID, err))
			s.db.Save(syncTask)
			continue
		}

		inserted := s.upsertNVDBatch(nvdResp.Vulnerabilities)
		totalInserted += inserted
		syncTask.SuccessCount += inserted
		s.db.Save(syncTask)

		time.Sleep(RateLimitDelay)
	}

	if len(failures) > 0 {
		syncTask.Status = "failed"
		syncTask.ErrorMessage = strings.Join(failures, "; ")
	} else {
		syncTask.Status = "completed"
	}
	endTime := time.Now()
	syncTask.EndTime = &endTime
	s.db.Save(syncTask)
	s.loadToCache()

	if len(failures) > 0 {
		return totalInserted, fmt.Errorf("failed to sync %d CVEs", len(failures))
	}

	return totalInserted, nil
}

func (s *VulnDBService) marshalAffectedTargets(targets []affectedTarget) string {
	payload, err := json.Marshal(targets)
	if err != nil {
		return ""
	}
	return string(payload)
}

// extractCVSS 提取 CVSS 评分
func (s *VulnDBService) extractCVSS(metrics *CVEMetrics) (float64, string, string) {
	// 优先使用 V3.1
	if len(metrics.CVSSMetricV31) > 0 {
		data := metrics.CVSSMetricV31[0].CVSSData
		return data.BaseScore, data.Severity, data.VectorString
	}

	// 其次 V3.0
	if len(metrics.CVSSMetricV30) > 0 {
		data := metrics.CVSSMetricV30[0].CVSSData
		return data.BaseScore, data.Severity, data.VectorString
	}

	// 最后 V2.0
	if len(metrics.CVSSMetricV2) > 0 {
		data := metrics.CVSSMetricV2[0].CVSSData
		severity := "medium"
		if data.BaseScore >= 9.0 {
			severity = "critical"
		} else if data.BaseScore >= 7.0 {
			severity = "high"
		} else if data.BaseScore >= 4.0 {
			severity = "medium"
		} else {
			severity = "low"
		}
		return data.BaseScore, severity, data.VectorString
	}

	return 0, "unknown", ""
}

// inferVulnType 推断漏洞类型
func (s *VulnDBService) inferVulnType(cweID, description string) string {
	// 从 CWE 推断
	if cweID != "" {
		cweMap := map[string]string{
			"CWE-79":  "xss",
			"CWE-89":  "sql-injection",
			"CWE-78":  "os-command-injection",
			"CWE-94":  "code-injection",
			"CWE-287": "authentication-bypass",
			"CWE-862": "authorization-bypass",
			"CWE-918": "ssrf",
			"CWE-611": "xxe",
			"CWE-22":  "path-traversal",
			"CWE-502": "deserialization",
			"CWE-400": "dos",
			"CWE-200": "information-disclosure",
			"CWE-295": "certificate-validation",
			"CWE-434": "file-upload",
			"CWE-306": "missing-authentication",
		}
		if v, ok := cweMap[cweID]; ok {
			return v
		}
	}

	// 从描述关键词推断
	descLower := strings.ToLower(description)
	keywords := map[string][]string{
		"sql injection":               {"sql-injection", "sqli"},
		"cross-site script":           {"xss"},
		"remote code execution":       {"rce", "code-execution"},
		"command injection":           {"command-injection", "rce"},
		"path traversal":              {"path-traversal", "lfi"},
		"server-side request forgery": {"ssrf"},
		"xml external entity":         {"xxe"},
		"denial of service":           {"dos"},
		"information disclosure":      {"information-disclosure"},
		"authentication bypass":       {"authentication-bypass"},
		"privilege escalation":        {"privilege-escalation"},
		"buffer overflow":             {"buffer-overflow"},
		"deserialization":             {"deserialization"},
	}

	for keyword, types := range keywords {
		if strings.Contains(descLower, keyword) {
			return types[0]
		}
	}

	return "unknown"
}

// SyncRecentNVD 同步最近一段时间的 NVD 数据
func (s *VulnDBService) SyncRecentNVD(days int) (int, error) {
	now := time.Now().UTC()
	startTime := now.AddDate(0, 0, -days)
	lastModStart := url.QueryEscape(formatNVDAPITime(startTime))
	lastModEnd := url.QueryEscape(formatNVDAPITime(now))
	return s.syncNVDByURLBuilder("nvd", func(startIndex, resultsPerPage int) string {
		return fmt.Sprintf(
			"%s?lastModStartDate=%s&lastModEndDate=%s&startIndex=%d&resultsPerPage=%d",
			NVDApiBaseURL,
			lastModStart,
			lastModEnd,
			startIndex,
			resultsPerPage,
		)
	}, 0)
}

func formatNVDAPITime(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000Z")
}

// FullSyncNVD 全量同步 NVD 数据（首次使用）
func (s *VulnDBService) FullSyncNVD() (int, error) {
	totalInserted := 0
	startYear := 2000
	endYear := time.Now().UTC().Year()
	for year := startYear; year <= endYear; year++ {
		rangeStart := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
		rangeEnd := time.Date(year, time.December, 31, 23, 59, 59, 0, time.UTC)
		inserted, err := s.SyncPublishedRangeNVD(rangeStart, rangeEnd)
		totalInserted += inserted
		if err != nil {
			return totalInserted, err
		}
	}
	return totalInserted, nil
}

// SearchByProduct 根据产品名搜索漏洞
func (s *VulnDBService) SearchByProduct(product string, limit int) ([]models.VulnerabilityDatabase, error) {
	productLower := strings.ToLower(product)

	var results []models.VulnerabilityDatabase

	// 先从产品缓存中查找
	if cached, ok := s.productCache.Load(productLower); ok {
		results = cached.([]models.VulnerabilityDatabase)
		if limit > 0 && len(results) > limit {
			results = results[:limit]
		}
		return results, nil
	}

	// 数据库查询
	query := s.db.Where(
		"LOWER(affected_product) LIKE ? OR LOWER(product) LIKE ? OR LOWER(vendor) LIKE ? OR LOWER(affected_cpe) LIKE ?",
		"%"+productLower+"%",
		"%"+productLower+"%",
		"%"+productLower+"%",
		"%"+productLower+"%",
	).Where("LOWER(source) <> ?", "dev-test")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&results).Error; err != nil {
		return nil, err
	}

	return results, nil
}

// GetVulnByService 根据服务名和产品版本匹配漏洞
func (s *VulnDBService) GetVulnByService(serviceName, productName, version string) []models.VulnerabilityDatabase {
	var results []models.VulnerabilityDatabase

	// 查询条件：产品名匹配
	searchTerms := []string{}
	if productName != "" {
		searchTerms = append(searchTerms, strings.ToLower(productName))
	}
	if serviceName != "" {
		searchTerms = append(searchTerms, strings.ToLower(serviceName))
	}

	for _, term := range searchTerms {
		var vulns []models.VulnerabilityDatabase
		s.db.Where(
			"LOWER(affected_product) LIKE ? OR LOWER(product) LIKE ? OR LOWER(vendor) LIKE ? OR LOWER(affected_cpe) LIKE ?",
			"%"+term+"%",
			"%"+term+"%",
			"%"+term+"%",
			"%"+term+"%",
		).
			Where("LOWER(source) <> ?", "dev-test").
			Where("severity IN ?", []string{"critical", "high", "medium"}).
			Order("cvss_score DESC").
			Limit(50).
			Find(&vulns)

		results = append(results, vulns...)
	}

	// 去重
	seen := make(map[string]bool)
	var uniqueResults []models.VulnerabilityDatabase
	for _, v := range results {
		if !seen[v.CVEID] {
			seen[v.CVEID] = true
			uniqueResults = append(uniqueResults, v)
		}
	}

	return uniqueResults
}

// GetSyncTasks 获取同步任务列表
func (s *VulnDBService) GetSyncTasks(limit int) ([]SyncTask, error) {
	var tasks []SyncTask
	query := s.db.Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

// GetLastSyncTime 获取最后同步时间
func (s *VulnDBService) GetLastSyncTime() *time.Time {
	var task SyncTask
	if err := s.db.Where("source LIKE ?", "nvd%").Where("status = ?", "completed").
		Order("end_time DESC").First(&task).Error; err != nil {
		return nil
	}
	return task.EndTime
}

// 保留原有的方法以兼容
func (s *VulnDBService) LookupCVE(cveID string) *models.VulnerabilityDatabase {
	cveID = strings.ToUpper(cveID)

	if v, ok := s.localCache.Load(cveID); ok {
		vuln := v.(models.VulnerabilityDatabase)
		return &vuln
	}

	var vuln models.VulnerabilityDatabase
	if err := s.db.Where("cve_id = ?", cveID).First(&vuln).Error; err == nil {
		s.localCache.Store(cveID, vuln)
		return &vuln
	}

	return nil
}

func (s *VulnDBService) SearchVulns(keyword string, limit int) ([]models.VulnerabilityDatabase, error) {
	var vulns []models.VulnerabilityDatabase
	query := s.db.Where("cve_id LIKE ? OR title LIKE ? OR description LIKE ?",
		"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&vulns).Error; err != nil {
		return nil, err
	}
	return vulns, nil
}

func (s *VulnDBService) GetStats() (map[string]int64, error) {
	stats := make(map[string]int64)

	var total int64
	if err := s.db.Model(&models.VulnerabilityDatabase{}).Count(&total).Error; err != nil {
		return nil, err
	}
	stats["total"] = total

	var critical, high, medium, low int64
	s.db.Model(&models.VulnerabilityDatabase{}).Where("severity = ?", "critical").Count(&critical)
	s.db.Model(&models.VulnerabilityDatabase{}).Where("severity = ?", "high").Count(&high)
	s.db.Model(&models.VulnerabilityDatabase{}).Where("severity = ?", "medium").Count(&medium)
	s.db.Model(&models.VulnerabilityDatabase{}).Where("severity = ?", "low").Count(&low)

	stats["critical"] = critical
	stats["high"] = high
	stats["medium"] = medium
	stats["low"] = low

	// 本周新增
	weekAgo := time.Now().AddDate(0, 0, -7)
	var weekCount int64
	s.db.Model(&models.VulnerabilityDatabase{}).Where("created_at > ?", weekAgo).Count(&weekCount)
	stats["this_week"] = weekCount

	return stats, nil
}

func (s *VulnDBService) EnrichVulnerability(cveID string) *models.VulnEnrichment {
	cveID = strings.ToUpper(cveID)
	if !strings.HasPrefix(cveID, "CVE-") {
		cveID = "CVE-" + cveID
	}

	vuln := s.LookupCVE(cveID)
	if vuln == nil {
		return nil
	}

	return &models.VulnEnrichment{
		CNVDID:    vuln.CNVDID,
		CNNVDID:   vuln.CNNVDID,
		CNCVEID:   vuln.CNCVEID,
		Title:     vuln.Title,
		Severity:  vuln.Severity,
		CVSSScore: vuln.CVSSScore,
		Solution:  vuln.Solution,
		VulnType:  vuln.VulnType,
	}
}

// SyncFromCSV 保留原有方法
func (s *VulnDBService) SyncFromCSV(csvData string) (int, error) {
	lines := strings.Split(csvData, "\n")
	var inserted int

	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) < 5 {
			continue
		}

		cveID := strings.TrimSpace(parts[0])
		if cveID == "" || !strings.HasPrefix(cveID, "CVE-") {
			continue
		}

		var vuln models.VulnerabilityDatabase
		s.db.Where("cve_id = ?", cveID).First(&vuln)

		if vuln.ID == 0 {
			vuln = models.VulnerabilityDatabase{
				CVEID:       cveID,
				CNVDID:      strings.TrimSpace(parts[1]),
				CNNVDID:     strings.TrimSpace(parts[2]),
				Title:       strings.TrimSpace(parts[3]),
				Severity:    strings.TrimSpace(parts[4]),
				Source:      "import",
				LastUpdated: time.Now(),
			}
			if len(parts) > 5 {
				vuln.Description = strings.TrimSpace(parts[5])
			}
			if len(parts) > 6 {
				vuln.VulnType = strings.TrimSpace(parts[6])
			}
			if len(parts) > 7 {
				vuln.CVSSScore, _ = strconv.ParseFloat(strings.TrimSpace(parts[7]), 64)
			}
			if len(parts) > 8 {
				vuln.Solution = strings.TrimSpace(parts[8])
			}

			s.db.Create(&vuln)
			inserted++
		}
	}

	s.loadToCache()
	return inserted, nil
}