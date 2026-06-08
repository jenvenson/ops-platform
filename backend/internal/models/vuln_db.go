package models

import "time"

// VulnerabilityDatabase 漏洞知识库（离线本地数据库）
type VulnerabilityDatabase struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	CVEID         string    `json:"cve_id" gorm:"size:50;uniqueIndex;not null"` // CVE-2021-xxx
	CNVDID        string    `json:"cnvd_id" gorm:"size:50;index"`               // CNVD-2021-xxx
	CNNVDID       string    `json:"cnnvd_id" gorm:"size:50;index"`              // CNNVD-2021-xxx
	CNCVEID       string    `json:"cncve_id" gorm:"size:50"`                    // CNCVE-2021-xxx

	// 漏洞基本信息
	Title         string    `json:"title" gorm:"size:200;not null"`
	Description   string    `json:"description" gorm:"type:text"`
	VulnType      string    `json:"vuln_type" gorm:"size:50"`       // rce, xss, sqli, etc.
	Severity      string    `json:"severity" gorm:"size:20"`       // critical, high, medium, low
	CVSSScore     float64   `json:"cvss_score"`
	CVSSVector    string    `json:"cvss_vector" gorm:"size:200"`

	// 影响范围
	AffectedProduct  string `json:"affected_product" gorm:"size:200"`  // 受影响产品
	AffectedVersion  string `json:"affected_version" gorm:"size:200"` // 受影响版本
	AffectedCPE      string `json:"affected_cpe" gorm:"size:255;index"` // 受影响 CPE
	Vendor           string `json:"vendor" gorm:"size:100;index"`       // 厂商
	Product          string `json:"product" gorm:"size:100;index"`      // 产品
	VersionStartIncluding string `json:"version_start_including" gorm:"size:100"`
	VersionStartExcluding string `json:"version_start_excluding" gorm:"size:100"`
	VersionEndIncluding   string `json:"version_end_including" gorm:"size:100"`
	VersionEndExcluding   string `json:"version_end_excluding" gorm:"size:100"`
	RawConfigurations string `json:"raw_configurations" gorm:"type:longtext"` // NVD 原始受影响配置摘要

	// 修复信息
	Solution       string    `json:"solution" gorm:"type:text"`      // 修复建议
	PatchURL       string    `json:"patch_url" gorm:"size:500"`       // 补丁链接
	Workaround     string    `json:"workaround" gorm:"type:text"`    // 缓解措施

	// 参考信息
	References     string    `json:"references" gorm:"type:text"`     // 参考链接（逗号分隔）
	CWEID          string    `json:"cwe_id" gorm:"size:20"`           // CWE-xxx

	// 分类标签
	Tags           string    `json:"tags" gorm:"size:200"`            // 标签（逗号分隔）

	// 数据来源
	Source         string    `json:"source" gorm:"size:50"`          // nvd, cnnvd, cnvd, manual
	LastUpdated    time.Time `json:"last_updated"`

	CreatedAt      time.Time `json:"created_at"`
}

func (VulnerabilityDatabase) TableName() string {
	return "vulnerability_database"
}

// VulnEnrichment 漏洞增强信息结构
type VulnEnrichment struct {
	CNVDID     string  `json:"cnvd_id"`
	CNNVDID    string  `json:"cnnvd_id"`
	CNCVEID    string  `json:"cncve_id"`
	Title      string  `json:"title"`
	Severity   string  `json:"severity"`
	CVSSScore  float64 `json:"cvss_score"`
	Solution   string  `json:"solution"`
	VulnType   string  `json:"vuln_type"`
}

// VulnMappingData 漏洞映射数据结构（导出字段供外部访问）
type VulnMappingData struct {
	CnvdID     string
	CnnvdID    string
	CncveID    string
	Title      string
	Description string
	Solution   string
	VulnType   string
	Severity   string
	CvssScore  float64
}

// CommonVulnMapping 常见漏洞映射表（内存缓存，加速查询）
var CommonVulnMapping = map[string]VulnMappingData{
	// === Log4j 系列 ===
	"CVE-2021-44228": {
		CnvdID:    "CNVD-2021-73702",
		CnnvdID:   "CNNVD-202111-1822",
		Title:     "Apache Log4j 远程代码执行漏洞",
		Solution:  "升级 Apache Log4j 至 2.17.0 或更高版本，或应用官方缓解措施",
		VulnType:  "rce",
		Severity:  "critical",
		CvssScore: 10.0,
	},
	"CVE-2021-45046": {
		CnvdID:    "CNVD-2021-78570",
		CnnvdID:   "CNNVD-202112-1843",
		Title:     "Apache Log4j 远程代码执行漏洞（绕过）",
		Solution:  "升级 Apache Log4j 至 2.17.0 或更高版本",
		VulnType:  "rce",
		Severity:  "critical",
		CvssScore: 9.0,
	},
	"CVE-2021-45105": {
		CnvdID:    "CNVD-2021-81936",
		CnnvdID:   "CNNVD-202112-1881",
		Title:     "Apache Log4j 拒绝服务漏洞",
		Solution:  "升级 Apache Log4j 至 2.17.0 或更高版本",
		VulnType:  "dos",
		Severity:  "medium",
		CvssScore: 5.9,
	},

	// === Spring 系列 ===
	"CVE-2022-22965": {
		CnvdID:    "CNVD-2022-24934",
		CnnvdID:   "CNNVD-202203-2310",
		Title:     "Spring Framework 远程代码执行漏洞",
		Solution:  "升级 Spring Framework 至 5.3.18+ / 5.2.20+，或使用 JDK 11+",
		VulnType:  "rce",
		Severity:  "critical",
		CvssScore: 9.8,
	},
	"CVE-2022-22947": {
		CnvdID:    "CNVD-2022-42734",
		CnnvdID:   "CNNVD-202205-1390",
		Title:     "Spring Cloud Gateway 远程代码执行漏洞",
		Solution:  "升级 Spring Cloud Gateway 至 3.1.3+ / 3.0.7+ / 2.2.6.1+",
		VulnType:  "rce",
		Severity:  "high",
		CvssScore: 8.3,
	},
	"CVE-2022-31690": {
		CnvdID:    "CNVD-2022-73256",
		CnnvdID:   "CNNVD-202207-1178",
		Title:     "Spring Security 权限提升漏洞",
		Solution:  "升级 Spring Security 至 5.7.x / 5.8.x 或更高版本",
		VulnType:  "auth-bypass",
		Severity:  "high",
		CvssScore: 8.1,
	},

	// === Apache 系列 ===
	"CVE-2021-42013": {
		CnvdID:    "CNVD-2021-81905",
		CnnvdID:   "CNNVD-202112-2122",
		Title:     "Apache HTTP Server 路径遍历漏洞",
		Solution:  "升级 Apache HTTP Server 至 2.4.51 或更高版本",
		VulnType:  "lfi",
		Severity:  "high",
		CvssScore: 8.8,
	},
	"CVE-2021-41773": {
		CnvdID:    "CNVD-2021-79209",
		CnnvdID:   "CNNVD-202111-1889",
		Title:     "Apache HTTP Server 路径遍历漏洞",
		Solution:  "升级 Apache HTTP Server 至 2.4.51 或更高版本",
		VulnType:  "lfi",
		Severity:  "high",
		CvssScore: 8.6,
	},
	"CVE-2021-36161": {
		CnvdID:    "CNVD-2021-69852",
		CnnvdID:   "CNNVD-202107-1413",
		Title:     "Apache HTTP Server 范围绕过漏洞",
		Solution:  "升级 Apache HTTP Server 至 2.4.48 或更高版本",
		VulnType:  "bypass",
		Severity:  "medium",
		CvssScore: 7.5,
	},

	// === Nginx ===
	"CVE-2021-23017": {
		CnvdID:    "CNVD-2021-30191",
		CnnvdID:   "CNNVD-202105-1188",
		Title:     "Nginx 解析漏洞",
		Solution:  "升级 Nginx 至 1.20.1 / 1.21.0 或更高版本",
		VulnType:  "rce",
		Severity:  "high",
		CvssScore: 9.8,
	},

	// === OpenSSL ===
	"CVE-2022-0778": {
		CnvdID:    "CNVD-2022-23307",
		CnnvdID:   "CNNVD-202203-1396",
		Title:     "OpenSSL 拒绝服务漏洞",
		Solution:  "升级 OpenSSL 至 1.0.2zd / 1.1.1n / 3.0.2 或更高版本",
		VulnType:  "dos",
		Severity:  "medium",
		CvssScore: 5.3,
	},
	"CVE-2021-3711": {
		CnvdID:    "CNVD-2021-56732",
		CnnvdID:   "CNNVD-202108-1394",
		Title:     "OpenSSL 缓冲区溢出漏洞",
		Solution:  "升级 OpenSSL 至 1.1.1l / 3.0.0 或更高版本",
		VulnType:  "buffer-overflow",
		Severity:  "high",
		CvssScore: 7.4,
	},

	// === PHP ===
	"CVE-2021-21708": {
		CnvdID:    "CNVD-2021-26186",
		CnnvdID:   "CNNVD-202105-1111",
		Title:     "PHP 远程代码执行漏洞",
		Solution:  "升级 PHP 至 7.4.28 / 8.0.16 或更高版本",
		VulnType:  "rce",
		Severity:  "high",
		CvssScore: 8.1,
	},

	// === ThinkPHP ===
	"CVE-2022-25474": {
		CnvdID:    "CNVD-2022-29227",
		CnnvdID:   "CNNVD-202203-2320",
		Title:     "ThinkPHP 远程代码执行漏洞",
		Solution:  "升级 ThinkPHP 至 6.0.13+ 或应用官方修复补丁",
		VulnType:  "rce",
		Severity:  "critical",
		CvssScore: 10.0,
	},

	// === MySQL ===
	"CVE-2021-22569": {
		CnvdID:    "CNVD-2021-69612",
		CnnvdID:   "CNNVD-202112-1615",
		Title:     "MySQL 权限提升漏洞",
		Solution:  "升级 MySQL 至 8.0.28 / 5.7.37 或更高版本",
		VulnType:  "privilege-escalation",
		Severity:  "high",
		CvssScore: 8.8,
	},

	// === Redis ===
	"CVE-2022-31137": {
		CnvdID:    "CNVD-2022-57368",
		CnnvdID:   "CNNVD-202207-1234",
		Title:     "Redis 远程代码执行漏洞",
		Solution:  "升级 Redis 至 6.2.9 / 7.0.1 或更高版本",
		VulnType:  "rce",
		Severity:  "high",
		CvssScore: 8.6,
	},

	// === Jenkins ===
	"CVE-2022-27209": {
		CnvdID:    "CNVD-2022-27215",
		CnnvdID:   "CNNVD-202203-1312",
		Title:     "Jenkins 远程代码执行漏洞",
		Solution:  "升级 Jenkins 至 2.356+ 或应用安全更新",
		VulnType:  "rce",
		Severity:  "high",
		CvssScore: 8.8,
	},

	// === Docker / Kubernetes ===
	"CVE-2022-0185": {
		CnvdID:    "CNVD-2022-06798",
		CnnvdID:   "CNNVD-202201-1102",
		Title:     "Linux Kernel 容器逃逸漏洞",
		Solution:  "升级 Linux Kernel 至 5.4.173+ / 5.10.93+ / 5.15.16+",
		VulnType:  "container-escape",
		Severity:  "high",
		CvssScore: 8.8,
	},

	// === GitLab ===
	"CVE-2021-22205": {
		CnvdID:    "CNVD-2021-96709",
		CnnvdID:   "CNNVD-202111-1821",
		Title:     "GitLab 远程代码执行漏洞",
		Solution:  "升级 GitLab 至 14.6.2 / 14.5.4 / 13.12.9 或更高版本",
		VulnType:  "rce",
		Severity:  "critical",
		CvssScore: 10.0,
	},
	"CVE-2022-2185": {
		CnvdID:    "CNVD-2022-38877",
		CnnvdID:   "CNNVD-202206-1395",
		Title:     "GitLab 远程代码执行漏洞",
		Solution:  "升级 GitLab 至 14.10.5 / 15.0.4 / 15.1.1 或更高版本",
		VulnType:  "rce",
		Severity:  "critical",
		CvssScore: 9.6,
	},

	// === Jira ===
	"CVE-2022-26135": {
		CnvdID:    "CNVD-2022-42703",
		CnnvdID:   "CNNVD-202206-1311",
		Title:     "Atlassian Jira 权限绕过漏洞",
		Solution:  "升级 Jira 至 8.5.21 / 8.13.13 / 8.16.1 / 8.20.9 / 8.22.3 或更高版本",
		VulnType:  "auth-bypass",
		Severity:  "high",
		CvssScore: 8.5,
	},

	// === WordPress ===
	"CVE-2021-29447": {
		CnvdID:    "CNVD-2021-38579",
		CnnvdID:   "CNNVD-202104-1217",
		Title:     "WordPress 路径遍历漏洞",
		Solution:  "升级 WordPress 至 5.7.2 或更高版本",
		VulnType:  "lfi",
		Severity:  "medium",
		CvssScore: 7.5,
	},
	"CVE-2022-21662": {
		CnvdID:    "CNVD-2022-23808",
		CnnvdID:   "CNNVD-202201-1091",
		Title:     "WordPress 注入漏洞",
		Solution:  "升级 WordPress 至 5.8.4 / 5.7.6 / 5.6.8 / 5.5.9 或更高版本",
		VulnType:  "sql-injection",
		Severity:  "high",
		CvssScore: 8.0,
	},

	// === F5 BIG-IP ===
	"CVE-2022-1388": {
		CnvdID:    "CNVD-2022-27488",
		CnnvdID:   "CNNVD-202205-1424",
		Title:     "F5 BIG-IP 远程代码执行漏洞",
		Solution:  "升级 F5 BIG-IP 至 16.1.2.1 / 15.1.5.1 / 14.1.4.6 / 13.1.5 或应用 hotfix",
		VulnType:  "rce",
		Severity:  "critical",
		CvssScore: 9.8,
	},

	// === Citrix ADC ===
	"CVE-2022-27510": {
		CnvdID:    "CNVD-2022-27346",
		CnnvdID:   "CNNVD-202203-1023",
		Title:     "Citrix ADC 路径遍历漏洞",
		Solution:  "升级 Citrix ADC 至 13.0-88.12 / 13.1-49.13 或更高版本",
		VulnType:  "lfi",
		Severity:  "high",
		CvssScore: 8.6,
	},

	// === 通⽤ Web 漏洞 ===
	"CVE-2017-12629": {
		CnvdID:    "CNVD-2017-28269",
		CnnvdID:   "CNNVD-201711-1059",
		Title:     "Apache Solr XXE 漏洞",
		Solution:  "升级 Apache Solr 至 7.0.0 或更高版本",
		VulnType:  "xxe",
		Severity:  "high",
		CvssScore: 8.1,
	},
	"CVE-2018-7600": {
		CnvdID:    "CNVD-2018-06793",
		CnnvdID:   "CNNVD-201803-1059",
		Title:     "Drupal Core 远程代码执行漏洞",
		Solution:  "升级 Drupal Core 至 7.58 / 8.3.9 / 8.4.6 / 8.5.1 或更高版本",
		VulnType:  "rce",
		Severity:  "critical",
		CvssScore: 9.8,
	},
	// === Consul ===
	// Consul未授权访问漏洞（原理扫描，无CVE编号）
	"CONSUL-UNAUTHORIZED": {
		Title:      "Consul未授权访问漏洞",
		Description: "Consul是HashiCorp公司推出的开源工具，用于实现分布式系统的服务发现与配置。Consul默认配置下缺少访问控制，导致攻击者可以获取敏感信息。",
		Solution:    "请增加Consul的授权管理控制，启用ACL功能",
		VulnType:   "info-leak",
		Severity:   "medium",
		CvssScore:  6.5,
	},

	// === MySQL 2024 系列 (Oracle CPU Jul 2024) ===
	"CVE-2024-21171": {
		CnvdID:     "CNVD-2024-34920",
		CnnvdID:    "CNNVD-202407-1714",
		CncveID:    "CNCVE-202421171",
		Title:      "Oracle MySQL 安全漏洞",
		Description: "Oracle MySQL的MySQL Server存在安全漏洞。攻击者利用该漏洞可以导致MySQL Server挂起或频繁重复崩溃。",
		Solution:   "升级MySQL至官方安全版本，参考: https://www.oracle.com/security-alerts/cpujul2024.html",
		VulnType:   "dos",
		Severity:   "medium",
		CvssScore:  6.5,
	},
	"CVE-2024-21177": {
		CnvdID:     "CNVD-2024-34923",
		CnnvdID:    "CNNVD-202407-1718",
		CncveID:    "CNCVE-202421177",
		Title:      "Oracle MySQL 安全漏洞",
		Description: "Oracle MySQL的MySQL Server存在安全漏洞。攻击者利用该漏洞可以导致MySQL Server挂起或频繁重复崩溃。",
		Solution:   "升级MySQL至官方安全版本，参考: https://www.oracle.com/security-alerts/cpujul2024.html",
		VulnType:   "dos",
		Severity:   "medium",
		CvssScore:  6.5,
	},
	"CVE-2024-21166": {
		CnvdID:     "CNVD-2024-34749",
		CnnvdID:    "CNNVD-202407-1710",
		CncveID:    "CNCVE-202421166",
		Title:      "Oracle MySQL 安全漏洞",
		Description: "Oracle MySQL的MySQL Server存在安全漏洞。攻击者利用该漏洞可以创建、删除或修改关键数据。",
		Solution:   "升级MySQL至官方安全版本，参考: https://www.oracle.com/security-alerts/cpujul2024.html",
		VulnType:   "data-manipulation",
		Severity:   "medium",
		CvssScore:  5.9,
	},
	"CVE-2024-21163": {
		CnvdID:     "CNVD-2024-34919",
		CnnvdID:    "CNNVD-202407-1709",
		CncveID:    "CNCVE-202421163",
		Title:      "Oracle MySQL 安全漏洞",
		Description: "Oracle MySQL的MySQL Server存在安全漏洞。攻击者利用该漏洞可以获得对数据的更新、插入或删除权限。",
		Solution:   "升级MySQL至官方安全版本，参考: https://www.oracle.com/security-alerts/cpujul2024.html",
		VulnType:   "data-manipulation",
		Severity:   "medium",
		CvssScore:  5.5,
	},
	"CVE-2024-21165": {
		CnvdID:     "CNVD-2024-34748",
		CnnvdID:    "CNNVD-202407-1708",
		CncveID:    "CNCVE-202421165",
		Title:      "Oracle MySQL 安全漏洞",
		Description: "Oracle MySQL的MySQL Server存在安全漏洞。攻击者利用该漏洞可以获得对数据的更新、插入或删除权限。",
		Solution:   "升级MySQL至官方安全版本，参考: https://www.oracle.com/security-alerts/cpujul2024.html",
		VulnType:   "data-manipulation",
		Severity:   "medium",
		CvssScore:  5.5,
	},
	"CVE-2024-21135": {
		CnvdID:     "CNVD-2024-34715",
		CnnvdID:    "CNNVD-202407-1682",
		CncveID:    "CNCVE-202421135",
		Title:      "Oracle MySQL 安全漏洞",
		Description: "Oracle MySQL的MySQL Server存在安全漏洞。攻击者利用该漏洞可以导致MySQL Server挂起或频繁重复崩溃。",
		Solution:   "升级MySQL至官方安全版本，参考: https://www.oracle.com/security-alerts/cpujul2024.html",
		VulnType:   "dos",
		Severity:   "medium",
		CvssScore:  4.9,
	},
	"CVE-2024-21162": {
		CnvdID:     "CNVD-2024-34750",
		CnnvdID:    "CNNVD-202407-1711",
		CncveID:    "CNCVE-202421162",
		Title:      "Oracle MySQL 安全漏洞",
		Description: "Oracle MySQL的MySQL Server存在安全漏洞。攻击者利用该漏洞可以获得对数据的更新、插入或删除权限。",
		Solution:   "升级MySQL至官方安全版本，参考: https://www.oracle.com/security-alerts/cpujul2024.html",
		VulnType:   "data-manipulation",
		Severity:   "medium",
		CvssScore:  5.3,
	},
	"CVE-2024-21142": {
		CnvdID:     "CNVD-2024-34717",
		CnnvdID:    "CNNVD-202407-1683",
		CncveID:    "CNCVE-202421142",
		Title:      "Oracle MySQL 安全漏洞",
		Description: "Oracle MySQL的MySQL Server存在安全漏洞。攻击者利用该漏洞可以获得对数据的更新、插入或删除权限。",
		Solution:   "升级MySQL至官方安全版本，参考: https://www.oracle.com/security-alerts/cpujul2024.html",
		VulnType:   "data-manipulation",
		Severity:   "medium",
		CvssScore:  5.3,
	},
	"CVE-2024-21173": {
		CnvdID:     "CNVD-2024-34922",
		CnnvdID:    "CNNVD-202407-1716",
		CncveID:    "CNCVE-202421173",
		Title:      "Oracle MySQL 安全漏洞",
		Description: "Oracle MySQL的MySQL Server存在安全漏洞。攻击者利用该漏洞可以导致MySQL Server挂起或频繁重复崩溃。",
		Solution:   "升级MySQL至官方安全版本，参考: https://www.oracle.com/security-alerts/cpujul2024.html",
		VulnType:   "dos",
		Severity:   "medium",
		CvssScore:  5.3,
	},
	"CVE-2024-21179": {
		CnvdID:     "CNVD-2024-34921",
		CnnvdID:    "CNNVD-202407-1720",
		CncveID:    "CNCVE-202421179",
		Title:      "Oracle MySQL 安全漏洞",
		Description: "Oracle MySQL的MySQL Server存在安全漏洞。攻击者利用该漏洞可以导致MySQL Server挂起或频繁重复崩溃。",
		Solution:   "升级MySQL至官方安全版本，参考: https://www.oracle.com/security-alerts/cpujul2024.html",
		VulnType:   "dos",
		Severity:   "medium",
		CvssScore:  5.3,
	},
	"CVE-2024-21125": {
		CnvdID:     "CNVD-2024-34714",
		CnnvdID:    "CNNVD-202407-1681",
		CncveID:    "CNCVE-202421125",
		Title:      "Oracle MySQL 安全漏洞",
		Description: "Oracle MySQL的MySQL Server存在安全漏洞。攻击者利用该漏洞可以获得对数据的更新、插入或删除权限。",
		Solution:   "升级MySQL至官方安全版本，参考: https://www.oracle.com/security-alerts/cpujul2024.html",
		VulnType:   "data-manipulation",
		Severity:   "medium",
		CvssScore:  4.9,
	},
	"CVE-2024-21127": {
		CnvdID:     "CNVD-2024-34716",
		CnnvdID:    "CNNVD-202407-1680",
		CncveID:    "CNCVE-202421127",
		Title:      "Oracle MySQL 安全漏洞",
		Description: "Oracle MySQL的MySQL Server存在安全漏洞。攻击者利用该漏洞可以获得对数据的更新、插入或删除权限。",
		Solution:   "升级MySQL至官方安全版本，参考: https://www.oracle.com/security-alerts/cpujul2024.html",
		VulnType:   "data-manipulation",
		Severity:   "medium",
		CvssScore:  4.9,
	},
	"CVE-2024-21129": {
		CnvdID:     "CNVD-2024-34718",
		CnnvdID:    "CNNVD-202407-1679",
		CncveID:    "CNCVE-202421129",
		Title:      "Oracle MySQL 安全漏洞",
		Description: "Oracle MySQL的MySQL Server存在安全漏洞。攻击者利用该漏洞可以获得对数据的更新、插入或删除权限。",
		Solution:   "升级MySQL至官方安全版本，参考: https://www.oracle.com/security-alerts/cpujul2024.html",
		VulnType:   "data-manipulation",
		Severity:   "medium",
		CvssScore:  4.9,
	},
	"CVE-2024-21130": {
		CnvdID:     "CNVD-2024-34719",
		CnnvdID:    "CNNVD-202407-1678",
		CncveID:    "CNCVE-202421130",
		Title:      "Oracle MySQL 安全漏洞",
		Description: "Oracle MySQL的MySQL Server存在安全漏洞。攻击者利用该漏洞可以获得对数据的更新、插入或删除权限。",
		Solution:   "升级MySQL至官方安全版本，参考: https://www.oracle.com/security-alerts/cpujul2024.html",
		VulnType:   "data-manipulation",
		Severity:   "medium",
		CvssScore:  4.9,
	},

	// === MySQL 2024 系列 (Oracle CPU Apr 2024) ===
	"CVE-2024-0727": {
		CnvdID:     "CNVD-2024-11398",
		CnnvdID:    "CNNVD-202403-1923",
		CncveID:    "CNCVE-20240727",
		Title:      "Oracle MySQL 安全漏洞",
		Description: "Oracle MySQL的MySQL Server存在安全漏洞。攻击者利用该漏洞可以获得对数据的更新、插入或删除权限。",
		Solution:   "升级MySQL至官方安全版本，参考: https://www.oracle.com/security-alerts/cpuapr2024.html",
		VulnType:   "data-manipulation",
		Severity:   "medium",
		CvssScore:  5.3,
	},
	"CVE-2023-6129": {
		CnvdID:     "CNVD-2024-3822",
		CnnvdID:    "CNNVD-202401-736",
		CncveID:    "CNCVE-20236129",
		Title:      "Oracle MySQL 安全漏洞",
		Description: "Oracle MySQL的MySQL Server的Server: Packaging (OpenSSL)组件存在安全漏洞。该漏洞源于POLY1305 MAC算法存在安全问题。",
		Solution:   "升级MySQL至官方安全版本，参考: https://www.oracle.com/security-alerts/cpuapr2024.html",
		VulnType:   "crypto-bypass",
		Severity:   "medium",
		CvssScore:  6.5,
	},
	"CVE-2023-5678": {
		CnvdID:     "CNVD-2024-3823",
		CnnvdID:    "CNNVD-202401-737",
		CncveID:    "CNCVE-20235678",
		Title:      "Oracle MySQL 安全漏洞",
		Description: "Oracle MySQL的MySQL Server存在安全漏洞。攻击者利用该漏洞可以获得对数据的更新、插入或删除权限。",
		Solution:   "升级MySQL至官方安全版本，参考: https://www.oracle.com/security-alerts/cpuapr2024.html",
		VulnType:   "data-manipulation",
		Severity:   "medium",
		CvssScore:  5.3,
	},
	"CVE-2024-20994": {
		CnvdID:     "CNVD-2024-19011",
		CnnvdID:    "CNNVD-202404-2241",
		CncveID:    "CNCVE-202420994",
		Title:      "Oracle MySQL 安全漏洞",
		Description: "Oracle MySQL的MySQL Server存在安全漏洞。攻击者利用该漏洞可以获得对数据的更新、插入或删除权限。",
		Solution:   "升级MySQL至官方安全版本，参考: https://www.oracle.com/security-alerts/cpuapr2024.html",
		VulnType:   "data-manipulation",
		Severity:   "medium",
		CvssScore:  5.3,
	},
	"CVE-2024-21096": {
		CnvdID:     "CNVD-2024-34925",
		CnnvdID:    "CNNVD-202404-2221",
		CncveID:    "CNCVE-202421096",
		Title:      "Oracle MySQL 安全漏洞",
		Description: "Oracle MySQL的MySQL Server存在安全漏洞。攻击者利用该漏洞可以获得对数据的更新、插入或删除权限。",
		Solution:   "升级MySQL至官方安全版本，参考: https://www.oracle.com/security-alerts/cpuapr2024.html",
		VulnType:   "data-manipulation",
		Severity:   "medium",
		CvssScore:  4.9,
	},
	"CVE-2024-20998": {
		CnvdID:     "CNVD-2024-20814",
		CnnvdID:    "CNNVD-202404-2228",
		CncveID:    "CNCVE-202420998",
		Title:      "Oracle MySQL 安全漏洞",
		Description: "Oracle MySQL的MySQL Server存在安全漏洞。攻击者利用该漏洞导致MySQL服务器挂起或频繁重复崩溃。",
		Solution:   "升级MySQL至官方安全版本，参考: https://www.oracle.com/security-alerts/cpuapr2024.html",
		VulnType:   "dos",
		Severity:   "medium",
		CvssScore:  4.9,
	},
	"CVE-2024-21009": {
		CnvdID:     "CNVD-2024-19017",
		CnnvdID:    "CNNVD-202404-2226",
		CncveID:    "CNCVE-202421009",
		Title:      "Oracle MySQL 安全漏洞",
		Description: "Oracle MySQL的MySQL Server存在安全漏洞。攻击者利用该漏洞导致MySQL服务器挂起或频繁重复崩溃。",
		Solution:   "升级MySQL至官方安全版本，参考: https://www.oracle.com/security-alerts/cpuapr2024.html",
		VulnType:   "dos",
		Severity:   "medium",
		CvssScore:  4.9,
	},
	"CVE-2024-21054": {
		CnvdID:     "CNVD-2024-34924",
		CnnvdID:    "CNNVD-202404-2242",
		CncveID:    "CNCVE-202421054",
		Title:      "Oracle MySQL 安全漏洞",
		Description: "Oracle MySQL的MySQL Server存在安全漏洞。攻击者利用该漏洞可以获得对数据的更新、插入或删除权限。",
		Solution:   "升级MySQL至官方安全版本，参考: https://www.oracle.com/security-alerts/cpuapr2024.html",
		VulnType:   "data-manipulation",
		Severity:   "medium",
		CvssScore:  4.9,
	},
	"CVE-2024-21060": {
		CnvdID:     "CNVD-2024-19016",
		CnnvdID:    "CNNVD-202404-2243",
		CncveID:    "CNCVE-202421060",
		Title:      "Oracle MySQL 安全漏洞",
		Description: "Oracle MySQL的MySQL Server存在安全漏洞。攻击者利用该漏洞可以获得对数据的更新、插入或删除权限。",
		Solution:   "升级MySQL至官方安全版本，参考: https://www.oracle.com/security-alerts/cpuapr2024.html",
		VulnType:   "data-manipulation",
		Severity:   "medium",
		CvssScore:  4.9,
	},
	"CVE-2024-21087": {
		CnvdID:     "CNVD-2024-19015",
		CnnvdID:    "CNNVD-202404-2245",
		CncveID:    "CNCVE-202421087",
		Title:      "Oracle MySQL 安全漏洞",
		Description: "Oracle MySQL的MySQL Server存在安全漏洞。攻击者利用该漏洞可以获得对数据的更新、插入或删除权限。",
		Solution:   "升级MySQL至官方安全版本，参考: https://www.oracle.com/security-alerts/cpuapr2024.html",
		VulnType:   "data-manipulation",
		Severity:   "medium",
		CvssScore:  4.9,
	},
	"CVE-2024-21047": {
		CnvdID:     "CNVD-2024-34926",
		CnnvdID:    "CNNVD-202404-2227",
		CncveID:    "CNCVE-202421047",
		Title:      "Oracle MySQL 安全漏洞",
		Description: "Oracle MySQL的MySQL Server存在安全漏洞。攻击者利用该漏洞可以获得对数据的更新、插入或删除权限。",
		Solution:   "升级MySQL至官方安全版本，参考: https://www.oracle.com/security-alerts/cpuapr2024.html",
		VulnType:   "data-manipulation",
		Severity:   "medium",
		CvssScore:  4.9,
	},
	"CVE-2024-21069": {
		CnvdID:     "CNVD-2024-19012",
		CnnvdID:    "CNNVD-202404-2238",
		CncveID:    "CNCVE-202421069",
		Title:      "Oracle MySQL 安全漏洞",
		Description: "Oracle MySQL的MySQL Server存在安全漏洞。攻击者利用该漏洞可以获得对数据的更新、插入或删除权限。",
		Solution:   "升级MySQL至官方安全版本，参考: https://www.oracle.com/security-alerts/cpuapr2024.html",
		VulnType:   "data-manipulation",
		Severity:   "medium",
		CvssScore:  4.9,
	},
	"CVE-2024-20996": {
		CnvdID:     "CNVD-2024-19013",
		CnnvdID:    "CNNVD-202404-2239",
		CncveID:    "CNCVE-202420996",
		Title:      "Oracle MySQL 安全漏洞",
		Description: "Oracle MySQL的MySQL Server存在安全漏洞。攻击者利用该漏洞可以获得对数据的更新、插入或删除权限。",
		Solution:   "升级MySQL至官方安全版本，参考: https://www.oracle.com/security-alerts/cpuapr2024.html",
		VulnType:   "data-manipulation",
		Severity:   "medium",
		CvssScore:  4.9,
	},

	// === 远端WWW服务支持TRACE请求 ===
	// 通用Web服务器漏洞，无特定CVE
	"": {
		Title:      "远端WWW服务支持TRACE请求",
		Description: "远端WWW服务支持TRACE请求。RFC 2616介绍了TRACE请求，该请求典型地用于测试HTTP协议实现。攻击者利用TRACE请求，结合其它浏览器端漏洞，有可能进行跨站脚本攻击，获取敏感信息。",
		Solution:   "管理员应禁用WWW服务对TRACE请求的支持。Apache: 使用Mod_Rewrite阻止TRACE请求；Nginx: 配置server块禁止TRACE方法",
		VulnType:   "xss",
		Severity:   "medium",
		CvssScore:  5.3,
	},
}
