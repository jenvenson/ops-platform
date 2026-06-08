package monitor

import (
	"errors"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/edy/ops-platform/internal/cmdb"
	"github.com/edy/ops-platform/internal/database"
)

// CheckConfig 健康检查配置
type CheckConfig struct {
	Interval     time.Duration // 检查间隔，默认 1 分钟
	Timeout      time.Duration // 超时时间，默认 3 秒
	OfflineAfter time.Duration // 离线阈值，默认 5 分钟
	SSHPort      int           // SSH 端口，用于 TCP 检测
}

// ServerStatus 服务器状态
type ServerStatus struct {
	ServerID  uint      `json:"server_id"`
	Hostname  string    `json:"hostname"`
	IP        string    `json:"ip"`
	SSHPort   int       `json:"ssh_port"`
	Online    bool      `json:"online"`
	Latency   int       `json:"latency_ms"`
	LastCheck time.Time `json:"last_check"`
}

var (
	cfg = CheckConfig{
		Interval:     1 * time.Minute,
		Timeout:      3 * time.Second,
		OfflineAfter: 5 * time.Minute,
		SSHPort:      22,
	}

	errInvalidMonitorAddress = errors.New("invalid monitor address")
	numericAddressPattern    = regexp.MustCompile(`^\d+$`)
)

// CheckServerOnline 检测服务器是否在线
func CheckServerOnline(ip string, port int) (bool, int, error) {
	start := time.Now()
	if err := validateMonitorAddress(ip); err != nil {
		return false, 0, err
	}

	// 优先使用 TCP 端口检测（更可靠）
	if port > 0 {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), cfg.Timeout)
		if err == nil {
			conn.Close()
			return true, int(time.Since(start).Milliseconds()), nil
		}
	}

	// 备用：ICMP ping 检测
	latency, err := ping(ip)
	if err == nil {
		return true, latency, nil
	}

	return false, 0, err
}

func validateMonitorAddress(ip string) error {
	trimmed := strings.TrimSpace(ip)
	if trimmed == "" {
		return fmt.Errorf("%w: empty address", errInvalidMonitorAddress)
	}
	if net.ParseIP(trimmed) != nil {
		return nil
	}
	if numericAddressPattern.MatchString(trimmed) {
		return fmt.Errorf("%w: %s", errInvalidMonitorAddress, trimmed)
	}
	return nil
}

// ping 发送 ICMP ping
func ping(ip string) (int, error) {
	// 使用系统 ping 命令
	cmd := exec.Command("ping", "-c", "1", "-W", "3", ip)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("ping 失败: %s", string(output))
	}

	// 解析延迟 - 支持多种输出格式
	outputStr := string(output)
	var latency int
	// 尝试解析: rtt min/avg/max/mdev = 10.123/10.456/10.789/0.123 ms
	_, err = fmt.Sscanf(outputStr, "rtt min/avg/max/mdev = %d", &latency)
	if err != nil {
		// 尝试解析 Linux 格式: time=10.123 ms
		_, err = fmt.Sscanf(outputStr, " time=%d", &latency)
		if err != nil {
			// 尝试解析 macOS 格式
			_, err = fmt.Sscanf(outputStr, "%d bytes from", &latency)
			if err != nil {
				// 解析失败但 ping 成功，返回默认延迟
				latency = 1
			}
		}
	}
	return latency, nil
}

// CheckAllServers 检测所有服务器状态
func CheckAllServers() []ServerStatus {
	var servers []cmdb.Server
	if err := database.DB.Find(&servers).Error; err != nil {
		fmt.Printf("ERROR: 查询服务器列表失败: %v\n", err)
		return nil
	}

	results := make([]ServerStatus, 0, len(servers))
	now := time.Now()

	for _, s := range servers {
		if s.DeletedAt != nil {
			continue
		}

		online, latency, err := CheckServerOnline(s.IP, s.SSHPort)
		if err != nil {
			if !errors.Is(err, errInvalidMonitorAddress) {
				fmt.Printf("WARN: 检测服务器 %s (%s) 失败: %v\n", s.Hostname, s.IP, err)
			}
		}

		status := "online"
		if !online {
			status = "offline"
		}

		// 更新数据库
		updates := map[string]interface{}{
			"status":         status,
			"last_heartbeat": &now,
		}
		if online && latency > 0 {
			updates["cpu_usage"] = float64(latency) // 临时用 latency 字段存储延迟
		}
		database.DB.Model(&s).Updates(updates)

		results = append(results, ServerStatus{
			ServerID:  s.ID,
			Hostname:  s.Hostname,
			IP:        s.IP,
			SSHPort:   s.SSHPort,
			Online:    online,
			Latency:   latency,
			LastCheck: now,
		})

		if !online {
			fmt.Printf("WARN: 服务器离线: %s (%s)\n", s.Hostname, s.IP)
		}
	}

	return results
}

// GetServerStatus 获取服务器状态列表
func GetServerStatus() []ServerStatus {
	var servers []cmdb.Server
	if err := database.DB.Find(&servers).Error; err != nil {
		fmt.Printf("ERROR: 查询服务器列表失败: %v\n", err)
		return nil
	}

	results := make([]ServerStatus, 0, len(servers))
	now := time.Now()
	offlineThreshold := now.Add(-cfg.OfflineAfter)

	for _, s := range servers {
		if s.DeletedAt != nil {
			continue
		}

		// 优先使用后台健康检查回写的状态字段，避免把陈旧心跳误判为在线。
		online := strings.EqualFold(s.Status, "online")
		if s.Status == "" {
			online = s.LastHeartbeat != nil && !s.LastHeartbeat.Before(offlineThreshold)
		}

		results = append(results, ServerStatus{
			ServerID:  s.ID,
			Hostname:  s.Hostname,
			IP:        s.IP,
			SSHPort:   s.SSHPort,
			Online:    online,
			Latency:   int(s.CPUUsage), // 临时使用
			LastCheck: now,
		})
	}

	return results
}

// StartChecker 启动健康检查定时任务
func StartChecker() {
	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	fmt.Printf("INFO: 启动服务器健康检查，间隔=%v\n", cfg.Interval)

	for range ticker.C {
		results := CheckAllServers()
		onlineCount := 0
		for _, r := range results {
			if r.Online {
				onlineCount++
			}
		}
		fmt.Printf("INFO: 健康检查完成，在线 %d/%d 服务器\n", onlineCount, len(results))
	}
}

// SetConfig 设置配置
func SetConfig(c CheckConfig) {
	cfg = c
}
