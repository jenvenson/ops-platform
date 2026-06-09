package security

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jenvenson/ops-platform/internal/database"
	"github.com/jenvenson/ops-platform/internal/models"
	"gorm.io/gorm"
)

// AssetEnrichmentService 资产增强服务
type AssetEnrichmentService struct {
	db *gorm.DB
}

// NewAssetEnrichmentService 创建资产增强服务
func NewAssetEnrichmentService() *AssetEnrichmentService {
	return &AssetEnrichmentService{
		db: database.DB,
	}
}

// AssetAttributeRules 资产属性增强规则
type AssetAttributeRules struct {
	ServiceName string
	Version     string
	OSInfo      string
	Banner      string
	Importance  string
	AssetType   string
	Tags        []string
}

// EnrichAssetAttributes 根据服务信息丰富资产属性
func (s *AssetEnrichmentService) EnrichAssetAttributes(attrRules *AssetAttributeRules) *models.Asset {
	asset := &models.Asset{
		ServiceName: attrRules.ServiceName,
		Version:     attrRules.Version,
		OSInfo:      attrRules.OSInfo,
		Banner:      attrRules.Banner,
		Tags:        strings.Join(attrRules.Tags, ","),
	}

	// 根据服务名推断资产类型
	asset.AssetType = s.determineAssetType(attrRules.ServiceName, attrRules.Banner)

	// 根据服务重要性和风险推断重要性等级
	asset.Importance = s.determineImportance(attrRules.ServiceName, attrRules.Version, asset.AssetType)

	// 根据服务特征添加标签
	s.addTags(asset, attrRules)

	return asset
}

// determineAssetType 根据服务名和横幅确定资产类型
func (s *AssetEnrichmentService) determineAssetType(serviceName, banner string) string {
	serviceLower := strings.ToLower(serviceName)
	bannerLower := strings.ToLower(banner)

	// 根据服务名称判断
	switch serviceLower {
	case "ssh":
		return models.AssetTypeServer
	case "telnet", "rdp", "vnc":
		return models.AssetTypeServer
	case "http", "https", "http-proxy", "ssl", "websocket":
		return models.AssetTypeWeb
	case "mysql", "postgresql", "mongodb", "redis", "oracle", "mssql":
		return models.AssetTypeDatabase
	case "ftp", "snmp", "dhcp", "dns", "ntp", "ldap":
		return models.AssetTypeNetwork
	default:
		// 通过banner进行更精确的判断
		if bannerLower != "" {
			if strings.Contains(bannerLower, "apache") || strings.Contains(bannerLower, "nginx") || strings.Contains(bannerLower, "iis") {
				return models.AssetTypeWeb
			} else if strings.Contains(bannerLower, "mysql") || strings.Contains(bannerLower, "postgres") || strings.Contains(bannerLower, "redis") {
				return models.AssetTypeDatabase
			} else if strings.Contains(bannerLower, "openssh") || strings.Contains(bannerLower, "microsoft") || strings.Contains(bannerLower, "linux") {
				return models.AssetTypeServer
			} else if strings.Contains(bannerLower, "cisco") || strings.Contains(bannerLower, "juniper") || strings.Contains(bannerLower, "huawei") {
				return models.AssetTypeNetwork
			}
		}
		return models.AssetTypeOther
	}
}

// determineImportance 根据服务信息判断重要性等级
func (s *AssetEnrichmentService) determineImportance(serviceName, version, assetType string) string {
	serviceLower := strings.ToLower(serviceName)

	// 关键服务，高重要性
	criticalServices := []string{"ssh", "rdp", "mysql", "postgresql", "http", "https", "dns", "dhcp", "ldap"}
	for _, svc := range criticalServices {
		if serviceLower == svc {
			return models.AssetImportanceCritical
		}
	}

	// 如果是Web服务且版本较旧，重要性较高
	if assetType == models.AssetTypeWeb {
		oldVersions := []string{"iis 6", "iis 7", "apache 2.2", "nginx 0."}
		banner := fmt.Sprintf("%s %s", serviceName, version)
		for _, oldVer := range oldVersions {
			if strings.Contains(strings.ToLower(banner), strings.ToLower(oldVer)) {
				return models.AssetImportanceHigh
			}
		}
		return models.AssetImportanceHigh
	}

	// 数据库服务，重要性较高
	if assetType == models.AssetTypeDatabase {
		return models.AssetImportanceHigh
	}

	// 一般服务，中等重要性
	return models.AssetImportanceMedium
}

// addTags 为资产添加标签
func (s *AssetEnrichmentService) addTags(asset *models.Asset, attrRules *AssetAttributeRules) {
	var tags []string

	// 根据服务类型添加标签
	switch asset.AssetType {
	case models.AssetTypeWeb:
		tags = append(tags, "web", "http", "application")
	case models.AssetTypeDatabase:
		tags = append(tags, "database", "storage", "persistence")
	case models.AssetTypeServer:
		tags = append(tags, "server", "infrastructure")
	case models.AssetTypeNetwork:
		tags = append(tags, "network", "infrastructure", "device")
	}

	// 根据服务名称添加标签
	serviceTags := map[string]string{
		"ssh":      "remote-access,secure-shell",
		"http":     "web,http,application",
		"https":    "web,http,application,encrypted",
		"mysql":    "database,sql,persistence",
		"redis":    "cache,database,in-memory",
		"nginx":    "web-server,proxy,reverse-proxy",
		"apache":   "web-server,httpd",
		"dns":      "domain,domain-name-system",
		"ldap":     "directory,authentication",
		"snmp":     "monitoring,management",
	}

	if serviceTag, exists := serviceTags[strings.ToLower(asset.ServiceName)]; exists {
		tags = append(tags, strings.Split(serviceTag, ",")...)
	}

	// 如果有特定版本信息，添加版本相关标签
	if asset.Version != "" {
		tags = append(tags, fmt.Sprintf("version-%s", strings.ReplaceAll(asset.Version, ".", "-")))
	}

	// 合并外部提供的标签
	tags = append(tags, attrRules.Tags...)

	// 去重并设置标签
	uniqueTags := s.removeDuplicateTags(tags)
	asset.Tags = strings.Join(uniqueTags, ",")
}

// removeDuplicateTags 去除重复标签
func (s *AssetEnrichmentService) removeDuplicateTags(tags []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag != "" && !seen[tag] {
			seen[tag] = true
			result = append(result, tag)
		}
	}
	return result
}

// UpdateAssetVulnerabilityCounts 更新资产关联漏洞数统计
func (s *AssetEnrichmentService) UpdateAssetVulnerabilityCounts(assetID uint) error {
	// 查询该资产关联的漏洞数量统计
	var counts struct {
		CriticalNum int
		HighNum     int
		MediumNum   int
		LowNum      int
		TotalNum    int
	}

	query := s.db.Table("security_vulnerabilities").
		Select(`
			SUM(CASE WHEN severity = 'critical' THEN 1 ELSE 0 END) as critical_num,
			SUM(CASE WHEN severity = 'high' THEN 1 ELSE 0 END) as high_num,
			SUM(CASE WHEN severity = 'medium' THEN 1 ELSE 0 END) as medium_num,
			SUM(CASE WHEN severity = 'low' THEN 1 ELSE 0 END) as low_num,
			COUNT(*) as total_num
		`).
		Where("asset_id = ?", assetID).
		Scan(&counts)

	if query.Error != nil {
		return query.Error
	}

	// 创建或更新漏洞数统计记录
	countRecord := models.AssetVulnCount{
		AssetID:     assetID,
		CriticalNum: counts.CriticalNum,
		HighNum:     counts.HighNum,
		MediumNum:   counts.MediumNum,
		LowNum:      counts.LowNum,
		TotalNum:    counts.TotalNum,
	}

	// 尝试更新，如果不存在则创建
	var existingCount models.AssetVulnCount
	err := s.db.Where("asset_id = ?", assetID).First(&existingCount).Error
	if err == gorm.ErrRecordNotFound {
		// 记录不存在，创建新记录
		return s.db.Create(&countRecord).Error
	} else if err != nil {
		return err
	}

	// 记录存在，更新
	return s.db.Model(&existingCount).Updates(countRecord).Error
}

// BulkUpdateAssetVulnerabilityCounts 批量更新资产漏洞统计
func (s *AssetEnrichmentService) BulkUpdateAssetVulnerabilityCounts(assetIDs []uint) error {
	for _, assetID := range assetIDs {
		if err := s.UpdateAssetVulnerabilityCounts(assetID); err != nil {
			return err
		}
	}
	return nil
}

// AggregateAssetStatistics 聚合资产统计数据
func (s *AssetEnrichmentService) AggregateAssetStatistics() (*models.ScanStatistics, error) {
	var stats models.ScanStatistics

	// 获取任务统计
	var tasks struct {
		Total      int64
		Running    int64
		Completed  int64
	}

	s.db.Model(&models.SecurityScanTask{}).Count(&tasks.Total)
	s.db.Model(&models.SecurityScanTask{}).Where("status = ?", "running").Count(&tasks.Running)
	s.db.Model(&models.SecurityScanTask{}).Where("status = ?", "completed").Count(&tasks.Completed)

	// 获取资产统计
	s.db.Model(&models.Asset{}).Count(&stats.TotalAssets)

	// 获取漏洞统计
	var vulnCounts struct {
		Total   int64
		High    int64
		Medium  int64
		Low     int64
	}

	s.db.Model(&models.SecurityVulnerability{}).Count(&vulnCounts.Total)
	s.db.Model(&models.SecurityVulnerability{}).Where("severity = ?", "high").Count(&vulnCounts.High)
	s.db.Model(&models.SecurityVulnerability{}).Where("severity = ?", "medium").Count(&vulnCounts.Medium)
	s.db.Model(&models.SecurityVulnerability{}).Where("severity = ?", "low").Count(&vulnCounts.Low)

	stats.TotalTasks = tasks.Total
	stats.RunningTasks = tasks.Running
	stats.CompletedTasks = tasks.Completed
	stats.TotalVulnerabilities = vulnCounts.Total
	stats.HighRiskCount = vulnCounts.High
	stats.MediumRiskCount = vulnCounts.Medium
	stats.LowRiskCount = vulnCounts.Low

	return &stats, nil
}

// AssetGroupManager 资产分组管理器
type AssetGroupManager struct {
	db *gorm.DB
}

// NewAssetGroupManager 创建资产分组管理器
func NewAssetGroupManager() *AssetGroupManager {
	return &AssetGroupManager{
		db: database.DB,
	}
}

// GroupAssetsByCriteria 根据不同条件对资产进行分组
func (gm *AssetGroupManager) GroupAssetsByCriteria(criteria string, value string) ([]models.Asset, error) {
	var assets []models.Asset

	switch criteria {
	case "type":
		err := gm.db.Where("asset_type = ?", value).Find(&assets).Error
		return assets, err
	case "importance":
		err := gm.db.Where("importance = ?", value).Find(&assets).Error
		return assets, err
	case "department":
		err := gm.db.Where("department = ?", value).Find(&assets).Error
		return assets, err
	case "tag":
		err := gm.db.Where("tags LIKE ?", "%"+value+"%").Find(&assets).Error
		return assets, err
	default:
		err := gm.db.Where("asset_group = ?", value).Find(&assets).Error
		return assets, err
	}
}

// CreateAssetGroupByPattern 根据正则表达式模式创建资产组
func (gm *AssetGroupManager) CreateAssetGroupByPattern(groupName, ipPattern string, assetType, importance string) error {
	// 使用事务确保一致性
	tx := gm.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 编译正则表达式
	re, err := regexp.Compile(ipPattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern: %v", err)
	}

	// 查找匹配的资产
	var assets []models.Asset
	if err := tx.Find(&assets).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 更新匹配的资产
	updatedCount := 0
	for _, asset := range assets {
		if re.MatchString(asset.IP) {
			updates := make(map[string]interface{})

			if groupName != "" {
				updates["asset_group"] = groupName
			}
			if assetType != "" {
				updates["asset_type"] = assetType
			}
			if importance != "" {
				updates["importance"] = importance
			}

			// 更新资产信息
			if err := tx.Model(&asset).Updates(updates).Error; err != nil {
				tx.Rollback()
				return err
			}
			updatedCount++
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	fmt.Printf("Successfully grouped %d assets under '%s'\n", updatedCount, groupName)
	return nil
}

// AssetRiskCalculator 资产风险计算器
type AssetRiskCalculator struct {
	db *gorm.DB
}

// NewAssetRiskCalculator 创建资产风险计算器
func NewAssetRiskCalculator() *AssetRiskCalculator {
	return &AssetRiskCalculator{
		db: database.DB,
	}
}

// CalculateAssetRiskScore 计算资产风险评分
func (rc *AssetRiskCalculator) CalculateAssetRiskScore(assetID uint) (float64, error) {
	var asset models.Asset
	if err := rc.db.First(&asset, assetID).Error; err != nil {
		return 0, err
	}

	var vulnCounts models.AssetVulnCount
	if err := rc.db.Where("asset_id = ?", assetID).First(&vulnCounts).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// 没有漏洞记录，返回基础风险评分
			return rc.calculateBaseRiskScore(&asset), nil
		}
		return 0, err
	}

	// 基础风险评分
	baseScore := rc.calculateBaseRiskScore(&asset)

	// 漏洞加权风险评分
	vulnScore := float64(vulnCounts.CriticalNum*10 + vulnCounts.HighNum*7 + vulnCounts.MediumNum*4 + vulnCounts.LowNum*1)

	// 最终风险评分
	finalScore := baseScore + vulnScore

	// 风险评分上限为100
	if finalScore > 100 {
		finalScore = 100
	}

	return finalScore, nil
}

// calculateBaseRiskScore 计算基础风险评分
func (rc *AssetRiskCalculator) calculateBaseRiskScore(asset *models.Asset) float64 {
	score := 0.0

	// 根据重要性等级调整基础评分
	switch asset.Importance {
	case models.AssetImportanceCritical:
		score += 25.0
	case models.AssetImportanceHigh:
		score += 15.0
	case models.AssetImportanceMedium:
		score += 8.0
	case models.AssetImportanceLow:
		score += 3.0
	}

	// 根据资产类型调整评分
	switch asset.AssetType {
	case models.AssetTypeServer:
		score += 10.0
	case models.AssetTypeDatabase:
		score += 12.0
	case models.AssetTypeWeb:
		score += 8.0
	case models.AssetTypeNetwork:
		score += 6.0
	}

	// 根据开放端口数量调整（假设开放端口越多风险越高）
	if asset.Port > 0 {
		if asset.Port <= 100 {
			score += 2.0
		} else if asset.Port <= 1000 {
			score += 5.0
		} else {
			score += 8.0
		}
	}

	// 基础分值为5分
	return score + 5.0
}

// UpdateAssetRiskScores 更新所有资产的风险评分
func (rc *AssetRiskCalculator) UpdateAssetRiskScores() error {
	var assets []models.Asset
	if err := rc.db.Find(&assets).Error; err != nil {
		return err
	}

	for _, asset := range assets {
		score, err := rc.CalculateAssetRiskScore(asset.ID)
		if err != nil {
			continue // 跳过计算失败的资产
		}

		// 更新资产风险评分（这里可以扩展数据库模型来保存风险评分）
		// 由于当前模型没有风险评分字段，暂时仅作演示
		fmt.Printf("Asset %s:%d Risk Score: %.2f\n", asset.IP, asset.Port, score)
	}

	return nil
}

// SyncAssetsFromScanResults 将扫描结果同步到资产中心
func (s *AssetEnrichmentService) SyncAssetsFromScanResults(taskID uint) error {
	// 获取扫描任务中的所有资产
	var scanAssets []models.SecurityAsset
	if err := s.db.Where("task_id = ?", taskID).Find(&scanAssets).Error; err != nil {
		return err
	}

	for _, scanAsset := range scanAssets {
		// 检查资产是否已存在于统一资产中心
		var existingAsset models.Asset
		err := s.db.Where("ip = ? AND port = ?", scanAsset.IP, scanAsset.Port).First(&existingAsset).Error

		if err == gorm.ErrRecordNotFound {
			// 资产不存在，创建新的资产记录
			newAsset := models.Asset{
				IP:          scanAsset.IP,
				Port:        scanAsset.Port,
				Protocol:    scanAsset.Protocol,
				ServiceName: scanAsset.ServiceName,
				Version:     scanAsset.Version,
				OSInfo:      scanAsset.OSInfo,
				Banner:      scanAsset.Banner,
				FirstSeen:   time.Now(),
				LastSeen:    time.Now(),
			}

			// 使用资产增强服务来丰富资产属性
			attrRules := &AssetAttributeRules{
				ServiceName: scanAsset.ServiceName,
				Version:     scanAsset.Version,
				OSInfo:      scanAsset.OSInfo,
				Banner:      scanAsset.Banner,
			}

			enrichedAsset := s.EnrichAssetAttributes(attrRules)

			// 合并增强后的属性
			newAsset.AssetType = enrichedAsset.AssetType
			newAsset.Importance = enrichedAsset.Importance
			newAsset.Tags = enrichedAsset.Tags

			if err := s.db.Create(&newAsset).Error; err != nil {
				return err
			}
		} else if err == nil {
			// 资产已存在，更新信息
			updates := map[string]interface{}{
				"protocol":     scanAsset.Protocol,
				"service_name": scanAsset.ServiceName,
				"version":      scanAsset.Version,
				"os_info":      scanAsset.OSInfo,
				"banner":       scanAsset.Banner,
				"last_seen":    time.Now(),
			}

			if err := s.db.Model(&existingAsset).Updates(updates).Error; err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}