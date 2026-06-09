package security

import (
	"fmt"
	"time"

	"github.com/jenvenson/ops-platform/internal/database"
	"github.com/jenvenson/ops-platform/internal/models"
)

// ============================================
// 扫描协调器 (Scan Coordinator)
// 职责：协调各模块执行完整的扫描流程
// ============================================

// ScanMode 扫描模式
type ScanMode string

const (
	ModeQuick  ScanMode = "quick"  // 快速扫描
	ModeNormal ScanMode = "normal" // 标准扫描
	ModeDeep   ScanMode = "deep"   // 深度扫描
)

// ScanCoordinator 扫描协调器
type ScanCoordinator struct {
	discoveryModule *DiscoveryModule
	vulnMatcher     *VulnMatcher
	webScanner      *WebScanner
}

// NewScanCoordinator 创建扫描协调器
func NewScanCoordinator() *ScanCoordinator {
	return &ScanCoordinator{
		discoveryModule: NewDiscoveryModule(),
		vulnMatcher:     NewVulnMatcher(),
		webScanner:      NewWebScanner(),
	}
}

// Execute 执行完整扫描流程
func (c *ScanCoordinator) Execute(taskID uint, target string, mode ScanMode, enableWebScan bool) error {
	fmt.Printf("Starting coordinated scan for task %d, target: %s, mode: %s\n", taskID, target, mode)

	// 初始化漏洞知识库
	vulnDB := NewVulnDBService()
	if err := vulnDB.InitVulnDB(); err != nil {
		fmt.Printf("Warning: Failed to initialize vuln DB: %v\n", err)
	}

	// 第一步：高速探测
	fmt.Printf("[Phase 1/4] Starting discovery...\n")
	discoveryConfig := NewDiscoveryConfig(target)
	discoveryResult, err := c.discoveryModule.Execute(discoveryConfig)
	if err != nil {
		return fmt.Errorf("discovery failed: %v", err)
	}

	openPorts := discoveryResult.FilterOpenPorts()
	fmt.Printf("[Phase 1] Discovery complete: %d open ports found\n", len(openPorts))

	// 保存资产到数据库
	var assetIDs map[int]uint
	var assetID uint
	if len(openPorts) > 0 {
		assetIDs = make(map[int]uint)
		for _, port := range openPorts {
			asset := models.SecurityAsset{
				TaskID:      taskID,
				IP:          discoveryResult.IP,
				Port:        port.PortID,
				Protocol:    port.Protocol,
				ServiceName: port.Service,
				Version:    port.Version,
				Banner:      port.Banner,
			}
			if err := database.DB.Create(&asset).Error; err != nil {
				fmt.Printf("Failed to create asset: %v\n", err)
			} else {
				assetIDs[port.PortID] = asset.ID
				assetID = asset.ID
			}
		}
	}

	// 第二步：指纹识别（已在探测阶段通过 Nmap -sV 完成）

	// 第三步：漏洞匹配
	fmt.Printf("[Phase 3/4] Starting vulnerability matching...\n")
	var matchedVulns []models.SecurityVulnerability

	if mode != ModeQuick {
		for _, port := range openPorts {
			vulns := c.vulnMatcher.MatchByFingerprint(&port)
			for _, vuln := range vulns {
				portAssetID := assetID
				if aid, ok := assetIDs[port.PortID]; ok {
					portAssetID = aid
				}
				vulnModel := vuln.ToVulnerability(discoveryResult.IP, port.PortID, taskID, portAssetID)
				matchedVulns = append(matchedVulns, vulnModel)
			}
		}
	}
	fmt.Printf("[Phase 3] Vulnerability matching complete: %d vulnerabilities found\n", len(matchedVulns))

	// 第四步：Web 漏洞扫描
	var webVulns []models.SecurityVulnerability
	if enableWebScan && mode != ModeQuick {
		fmt.Printf("[Phase 4/4] Starting web vulnerability scan...\n")
		webPorts := discoveryResult.GetWebPorts()
		if len(webPorts) > 0 {
			webResults, err := c.webScanner.ScanWebPorts(discoveryResult.IP, webPorts, nil)
			if err != nil {
				fmt.Printf("Web scan failed: %v\n", err)
			} else {
				for _, webResult := range webResults {
					portAssetID := assetID
					if aid, ok := assetIDs[webResult.Port]; ok {
						portAssetID = aid
					}
					vulnModel := webResult.ToVulnerability(taskID, portAssetID)
					webVulns = append(webVulns, vulnModel)
				}
			}
		}
		fmt.Printf("[Phase 4] Web scan complete: %d vulnerabilities found\n", len(webVulns))
	}

	// 保存所有漏洞到数据库
	allVulns := append(matchedVulns, webVulns...)
	for _, vuln := range allVulns {
		if err := database.DB.Create(&vuln).Error; err != nil {
			fmt.Printf("Failed to create vulnerability: %v\n", err)
		}
	}

	// 统计漏洞
	var highRisk, mediumRisk, lowRisk int
	for _, vuln := range allVulns {
		switch vuln.Severity {
		case "critical", "high":
			highRisk++
		case "medium":
			mediumRisk++
		default:
			lowRisk++
		}
	}

	// 更新任务状态
	completedAt := time.Now()
	updates := map[string]interface{}{
		"status":      "completed",
		"completed_at": &completedAt,
		"progress":   100,
		"scanned_ips": 1,
		"high_risk":  highRisk,
		"medium_risk": mediumRisk,
		"low_risk":   lowRisk,
		"message":    fmt.Sprintf("扫描完成，发现 %d 个高危，%d 个中危，%d 个低危漏洞", highRisk, mediumRisk, lowRisk),
	}
	database.DB.Model(&models.SecurityScanTask{}).Where("id = ?", taskID).Updates(updates)

	fmt.Printf("Scan complete for task %d: %d total vulnerabilities (%d high, %d medium, %d low)\n",
		taskID, len(allVulns), highRisk, mediumRisk, lowRisk)

	return nil
}

// ExecuteAsync 异步执行扫描
func (c *ScanCoordinator) ExecuteAsync(taskID uint, target string, mode ScanMode, enableWebScan bool) {
	go func() {
		if err := c.Execute(taskID, target, mode, enableWebScan); err != nil {
			fmt.Printf("Scan failed for task %d: %v\n", taskID, err)
			now := time.Now()
			database.DB.Model(&models.SecurityScanTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
				"status":  "failed",
				"message": err.Error(),
				"completed_at": &now,
			})
		}
	}()
}
