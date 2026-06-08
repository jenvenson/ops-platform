# 安全扫描引擎架构设计

## 总体架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                     安全扫描任务调度器                            │
│                 SecurityScanCoordinator                          │
└─────────────────────────────────────────────────────────────────┘
                                 │
         ┌───────────────────────┼───────────────────────┐
         ▼                       ▼                       ▼
┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐
│  模块一:        │   │  模块二:        │   │  模块四:        │
│  高速探测层     │   │  深度指纹层     │   │  Web漏洞扫描    │
│ DiscoveryModule │──▶│ FingerprintModule│◀──│ WebScanner      │
└─────────────────┘   └─────────────────┘   └─────────────────┘
         │                       │                       │
         │       开放端口列表     │                       │
         │   (TCP Ports)         │  服务指纹             │
         │                       │  (Service Fingerprint)│
         │                       ▼                       │
         │              ┌─────────────────┐              │
         │              │  模块三:        │              │
         └──────────────▶│  漏洞匹配层    │◀─────────────┘
                        │ VulnMatcher    │
                        └─────────────────┘
                                 │
                                 ▼
                        ┌─────────────────┐
                        │  模块五:        │
                        │  报告生成层     │
                        │ ReportGenerator │
                        └─────────────────┘
```

## 模块详细设计

### 模块一：高速探测层 (DiscoveryModule)

**职责**: 快速发现存活主机和开放端口

```
输入: 目标 IP/CIDR/域名
     并发数限制 (默认 1000)
     超时设置 (默认 3s)

处理:
  1. 并发控制 - 使用 worker pool
  2. TCP SYN 扫描 (RustScan/Nmap)
  3. 端口状态过滤 (只保留 open)
  4. 存活检测 (ICMP Ping)

输出: DiscoveryResult {
    IP: string
    Ports: []int           // 开放端口列表
    Hostname: string
    OS: string
    Latency: time.Duration
}
```

### 模块二：深度指纹识别层 (FingerprintModule)

**职责**: 对开放端口进行深入服务识别

```
输入: 目标 IP + 开放端口列表
     版本检测强度 (1-9, 默认 7)

处理:
  1. 针对每个开放端口发送探针
  2. 解析响应，提取:
     - 服务名称 (ssh, http, mysql...)
     - 产品名称 (OpenSSH, Apache, Nginx...)
     - 版本号 (8.9p1, 2.4.41...)
     - CPE 标识
  3. 协议栈指纹 (HTTP Server header, SSL/TLS info)

输出: FingerprintResult {
    Port: int
    Protocol: string
    Service: string
    Product: string
    Version: string
    CPE: string
    Banner: string
    TechStack: []string
}
```

### 模块三：漏洞匹配层 (VulnMatcher)

**职责**: 基于指纹匹配 CVE 漏洞

```
输入: 服务指纹列表
     漏洞知识库

处理:
  1. 服务 → CVE 映射查询
  2. 版本范围匹配:
     - CPE 匹配 (cpe:/a:openbsd:openssh:8.9)
     - 版本号比较 (semver)
  3. 风险排序 (CVSS)

输出: Vulnerability {
    CVEID: string
    CVSS: float64
    Severity: string  // critical/high/medium/low
    Title: string
    Description: string
    Solution: string
    MatchedOn: string  // "OpenSSH 8.9p1"
}
```

### 模块四：Web漏洞扫描层 (WebScanner)

**职责**: 针对 HTTP/HTTPS 服务进行 Web 漏洞检测

```
输入: Web 资产列表 (IP:Port + http/https)

处理:
  1. 目标发现:
     - 提取 WAF/框架指纹
     - 发现管理接口、API 端点
  2. 漏洞检测 (Nuclei):
     - cves/          - CVE 漏洞模板
     - vulnerabilities/ - 通用漏洞
     - technologies/   - 技术特定漏洞
     - exposed-panels/ - 暴露的管理面板
     - files/          - 敏感文件泄露
  3. 自定义扫描策略

输出: WebVulnerability {
    URL: string
    TemplateID: string
    CVE: []string
    Severity: string
    Description: string
    Raw: string  // 原始响应
}
```

### 模块五：报告生成层 (ReportGenerator)

**职责**: 生成可视化安全报告

```
输入: 资产列表 + 漏洞列表

处理:
  1. 风险计算
  2. 统计汇总
  3. 图表生成
  4. HTML/Markdown 渲染

输出: SecurityReport {
    Summary: ReportSummary
    Assets: []Asset
    Vulns: []Vulnerability
    WebVulns: []WebVulnerability
    Charts: map[string]string  // base64 编码的图表
}
```

## 数据模型设计

```go
// 扫描任务配置
type ScanConfig struct {
    Target      string        // 目标 IP/CIDR/域名
    TargetType  string        // ip/cidr/domain/url
    Mode        string        // quick/normal/deep
    WebScan     bool          // 是否启用 Web 扫描
    Concurrent  int           // 并发数
    Timeout     time.Duration // 超时时间
    Ports       []int         // 指定端口范围
    Tags        []string      // 扫描标签
}

// 扫描任务状态
type ScanTask struct {
    ID           uint
    Config       ScanConfig
    Status       string        // pending/running/completed/failed
    Progress     int           // 0-100
    Phase        string        // discovery/fingerprinting/vuln-matching/reporting
    AssetsFound  int           // 发现的资产数
    VulnsFound   int           // 发现的漏洞数
    StartedAt    time.Time
    CompletedAt  time.Time
}

// 资产模型
type Asset struct {
    ID        uint
    TaskID    uint
    IP        string
    Hostname  string
    Port      int
    Protocol  string
    Service   string
    Product   string
    Version   string
    CPE       string
    Banner    string
    OS        string
    VulnCount int
}

// 漏洞模型
type Vulnerability struct {
    ID          uint
    TaskID      uint
    AssetID     uint
    CVEID       string
    CVSS        float64
    Severity    string
    Title       string
    Description string
    Solution    string
    Source      string        // fingerprint/nuclei/manual
    MatchedOn   string        // 匹配依据
}
```

## 扫描模式

| 模式 | 描述 | 适用场景 |
|------|------|---------|
| **quick** | 快速扫描，仅发现端口和服务 | 初步资产盘点 |
| **normal** | 标准扫描，完整服务指纹 + Nuclei CVE | 常规安全评估 |
| **deep** | 深度扫描，完整指纹 + Web 扫描 + PoC 验证 | 详细渗透测试 |

## 配置示例

```yaml
scanner:
  # 全局并发限制
  max_concurrent: 1000
  # 文件描述符限制
  ulimit: 65535
  # 默认超时
  default_timeout: 3s
  # 扫描模式
  default_mode: normal

  # 模块配置
  discovery:
    enabled: true
    ports: [1-65535]
    batch_size: 1000

  fingerprint:
    enabled: true
    version_intensity: 7
    timeout: 5s

  web_scan:
    enabled: true
    nuclei_tags:
      - cves
      - vulnerabilities
      - technologies
      - exposed-panels
    timeout: 30s

  vuln_matching:
    enabled: true
    auto_enrich: true  # 自动从 NVD 获取 CVE 详情
```
