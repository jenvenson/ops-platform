package security

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ============================================
// 模块一：高速探测层 (Discovery Module)
// 职责：快速发现存活主机和开放端口
// ============================================

// DiscoveryConfig 探测层配置
type DiscoveryConfig struct {
	Target            string        // 目标 IP/CIDR/域名
	Ports             []int         // 端口范围，默认 1-65535
	Concurrent        int           // 并发数，默认 1000
	Timeout           time.Duration // 超时时间，默认 3s
	BatchSize         int           // 批处理大小，默认 1000
	VersionIntensity  int           // 版本检测强度，默认 7
}

// DiscoveryResult 探测结果
type DiscoveryResult struct {
	IP        string       // 目标 IP
	Hostname  string       // 主机名
	Ports     []PortInfo   // 开放端口列表
	OS        string       // 操作系统信息
	Latency   time.Duration // 延迟
	ScannedAt time.Time
}

// PortInfo 端口信息
type PortInfo struct {
	PortID    int    `json:"portid"`
	Protocol  string `json:"protocol"`
	State     string `json:"state"` // open/closed/filtered
	Service   string `json:"service,omitempty"`  // 服务名
	Product   string `json:"product,omitempty"`  // 产品名
	Version   string `json:"version,omitempty"`  // 版本
	CPE       string `json:"cpe,omitempty"`      // CPE 标识
	Banner    string `json:"banner,omitempty"`   // 完整 banner
}

// NewDiscoveryConfig 创建默认配置
func NewDiscoveryConfig(target string) *DiscoveryConfig {
	return &DiscoveryConfig{
		Target:            target,
		Ports:             []int{}, // 空表示全端口
		Concurrent:        1000,
		Timeout:           3 * time.Second,
		BatchSize:         1000,
		VersionIntensity:  7,
	}
}

// DiscoveryModule 探测模块
type DiscoveryModule struct{}

// NewDiscoveryModule 创建探测模块
func NewDiscoveryModule() *DiscoveryModule {
	return &DiscoveryModule{}
}

// Execute 执行端口探测
// 使用 Nmap 进行端口扫描和服务版本检测
func (m *DiscoveryModule) Execute(config *DiscoveryConfig) (*DiscoveryResult, error) {
	result := &DiscoveryResult{
		IP:        config.Target,
		Ports:     []PortInfo{},
		ScannedAt: time.Now(),
	}

	// 构建端口范围
	portRange := "1-65535"
	if len(config.Ports) > 0 {
		portRange = buildPortRange(config.Ports)
	}

	// 使用 Nmap 进行扫描（更快且更可靠）
	cmd := exec.Command("nmap",
		"-sT",              // TCP connect 扫描（更可靠）
		"-sV",              // 服务版本检测
		"-p", portRange,    // 端口范围
		"-T4",              // 快速扫描
		"--version-intensity", fmt.Sprintf("%d", config.VersionIntensity),
		"-oX", "-",         // XML 输出格式到 stdout
		config.Target,
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("Nmap stderr: %s\n", stderr.String())
		return nil, fmt.Errorf("nmap execution failed: %v", err)
	}

	// 解析 XML 输出
	m.parseXMLOutput(stdout.String(), result)

	fmt.Printf("Discovery result: ip=%s, open_ports=%d\n", result.IP, len(result.Ports))
	for _, port := range result.Ports {
		fmt.Printf("  - Port %d/%s: %s %s (state=%s)\n",
			port.PortID, port.Protocol, port.Service, port.Version, port.State)
	}

	return result, nil
}

// parseXMLOutput 解析 Nmap XML 输出
func (m *DiscoveryModule) parseXMLOutput(output string, result *DiscoveryResult) {
	// 移除 ANSI 转义码
	output = stripANSI(output)

	var nmapOutput NmapXMLOutput
	if err := xml.Unmarshal([]byte(output), &nmapOutput); err != nil {
		fmt.Printf("DEBUG XML parse error: %v\n", err)
		return
	}

	for _, host := range nmapOutput.Hosts {
		if result.IP == "" {
			result.IP = host.Address.Addr
		}

		for _, port := range host.Ports.Ports {
			// 只处理开放端口
			if !strings.Contains(strings.ToLower(port.State.State), "open") {
				continue
			}

			// 获取 CPE (可能在多个 cpe 子元素中)
			var cpe string
			for _, c := range port.Service.CPEs {
				if c != "" {
					cpe = c
					break
				}
			}

			portInfo := PortInfo{
				PortID:   port.PortID,
				Protocol: port.Protocol,
				State:    port.State.State,
				Service:  port.Service.Name,
				Product:  port.Service.Product,
				Version: port.Service.Version,
				CPE:     cpe,
			}

			// 构建 banner
			portInfo.Banner = buildBanner(portInfo.Service, portInfo.Product, portInfo.Version)

			// 如果服务名为空，使用端口默认服务
			if portInfo.Service == "" {
				portInfo.Service = detectService(portInfo.PortID)
			}

			fmt.Printf("DEBUG parseXMLOutput: port=%d, service=%s, version=%s, cpe=%s\n",
				portInfo.PortID, portInfo.Service, portInfo.Version, cpe)

			result.Ports = append(result.Ports, portInfo)
		}
	}
}

// buildPortRange 构建端口范围字符串
func buildPortRange(ports []int) string {
	if len(ports) == 0 {
		return "1-65535"
	}

	// 简化处理：转为逗号分隔
	var portStrs []string
	for _, p := range ports {
		portStrs = append(portStrs, fmt.Sprintf("%d", p))
	}
	return strings.Join(portStrs, ",")
}

// DiscoverMultiple 对多个目标执行探测
func (m *DiscoveryModule) DiscoverMultiple(targets []string, config *DiscoveryConfig) ([]*DiscoveryResult, error) {
	var results []*DiscoveryResult

	for _, target := range targets {
		config.Target = target
		result, err := m.Execute(config)
		if err != nil {
			fmt.Printf("Discovery failed for %s: %v\n", target, err)
			continue
		}
		results = append(results, result)
	}

	return results, nil
}

// FilterOpenPorts 只保留开放端口
func (r *DiscoveryResult) FilterOpenPorts() []PortInfo {
	var openPorts []PortInfo
	for _, port := range r.Ports {
		if strings.ToLower(port.State) == "open" {
			openPorts = append(openPorts, port)
		}
	}
	return openPorts
}

// GetWebPorts 获取 Web 服务端口 (HTTP/HTTPS)
func (r *DiscoveryResult) GetWebPorts() []PortInfo {
	webPorts := []PortInfo{}
	for _, port := range r.Ports {
		service := strings.ToLower(port.Service)
		if service == "http" || service == "https" ||
		   service == "http-proxy" || service == "https-alt" ||
		   port.PortID == 80 || port.PortID == 443 ||
		   port.PortID == 8080 || port.PortID == 8443 {
			webPorts = append(webPorts, port)
		}
	}
	return webPorts
}
