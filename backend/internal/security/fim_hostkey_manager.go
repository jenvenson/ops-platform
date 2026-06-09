package security

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/jenvenson/ops-platform/internal/database"
	"github.com/jenvenson/ops-platform/internal/models"
	"golang.org/x/crypto/ssh"
)

// StrictPostKeyCallback 严格模式的主机密钥验证
type StrictPostKeyCallback struct {
	serverID   *uint
	policyID   *uint
	snapshotID *uint
}

// NewStrictPostKeyCallback 创建严格模式的HostKeyCallback
func NewStrictPostKeyCallback(serverID, policyID, snapshotID *uint) *StrictPostKeyCallback {
	return &StrictPostKeyCallback{
		serverID:   serverID,
		policyID:   policyID,
		snapshotID: snapshotID,
	}
}

// VerifyHostKey 验证主机密钥（严格模式：只允许白名单中的主机）
func (s *StrictPostKeyCallback) VerifyHostKey(hostname string, remote net.Addr, key ssh.PublicKey) error {
	// 提取主机和端口
	host, port := s.parseHostPort(hostname, remote)

	// 查询白名单
	var knownHost models.FIMKnownHost
	result := database.DB.Where(
		"hostname = ? AND port = ? AND key_type = ? AND is_enabled = ?",
		host, port, key.Type(), true,
	).Limit(1).Find(&knownHost)

	if result.Error != nil {
		return s.handleUnknownHost(host, port, key)
	}

	if result.RowsAffected == 0 {
		// 主机密钥不存在于白名单
		return s.handleUnknownHost(host, port, key)
	}

	// 验证密钥是否匹配
	expectedKey := strings.TrimSpace(knownHost.PublicKey)
	presentedKey := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key)))

	if expectedKey != presentedKey {
		// 密钥不匹配（可能是MITM攻击或密钥已更新）
		return s.handleKeyMismatch(knownHost, key)
	}

	// 密钥匹配，验证成功
	s.recordSuccessLog(host, port, knownHost)
	s.updateHostUsage(knownHost.ID)

	return nil
}

// handleUnknownHost 处理未知主机
func (s *StrictPostKeyCallback) handleUnknownHost(host string, port int, key ssh.PublicKey) error {
	// 记录连接尝试日志
	connectionLog := models.FIMSSHConnectionLog{
		Hostname:             host,
		Port:                 port,
		Result:               "key_not_found",
		ErrorMessage:         "Host key not found in whitelist",
		PresentedKeyType:     ptrString(key.Type()),
		PresentedFingerprint: ptrString(ssh.FingerprintSHA256(key)),
		ServerID:             s.serverID,
		PolicyID:             s.policyID,
		SnapshotID:           s.snapshotID,
	}
	database.DB.Create(&connectionLog)

	// 发送告警（记录日志）
	log.Printf("[SECURITY ALERT] Unknown host attempted connection: %s:%d, Key: %s, Fingerprint: %s",
		host, port, key.Type(), ssh.FingerprintSHA256(key))

	// 严格模式：拒绝连接
	return fmt.Errorf("host key not found in whitelist for %s:%d. Please add it in the FIM Known Hosts management interface first", host, port)
}

// handleKeyMismatch 处理密钥不匹配
func (s *StrictPostKeyCallback) handleKeyMismatch(knownHost models.FIMKnownHost, newKey ssh.PublicKey) error {
	// 记录连接尝试日志
	connectionLog := models.FIMSSHConnectionLog{
		Hostname:             knownHost.Hostname,
		Port:                 knownHost.Port,
		Result:               "key_mismatch",
		ErrorMessage:         "Host key does not match the one in whitelist",
		PresentedKeyType:     ptrString(newKey.Type()),
		PresentedFingerprint: ptrString(ssh.FingerprintSHA256(newKey)),
		ExpectedFingerprint:  &knownHost.FingerprintSHA256,
		ServerID:             s.serverID,
		PolicyID:             s.policyID,
		SnapshotID:           s.snapshotID,
	}
	database.DB.Create(&connectionLog)

	// 发送告警（记录日志）
	log.Printf("[SECURITY ALERT] Host key mismatch for %s:%d. Expected: %s, Got: %s",
		knownHost.Hostname, knownHost.Port,
		knownHost.FingerprintSHA256,
		ssh.FingerprintSHA256(newKey))

	// 记录历史
	history := models.FIMKnownHostsHistory{
		HostID:         knownHost.ID,
		Action:         "key_changed",
		OldKeyType:     &knownHost.KeyType,
		OldPublicKey:   &knownHost.PublicKey,
		OldFingerprint: &knownHost.FingerprintSHA256,
		NewKeyType:     ptrString(newKey.Type()),
		NewPublicKey:   ptrString(string(ssh.MarshalAuthorizedKey(newKey))),
		NewFingerprint: ptrString(ssh.FingerprintSHA256(newKey)),
		OperatedBy:     "system",
		Reason:         "Key mismatch detected during SSH connection attempt",
	}
	database.DB.Create(&history)

	// 严格模式：拒绝连接
	return fmt.Errorf("host key mismatch for %s:%d. Expected: %s, Got: %s. This could indicate a MITM attack or the host key was changed",
		knownHost.Hostname, knownHost.Port,
		knownHost.FingerprintSHA256,
		ssh.FingerprintSHA256(newKey))
}

// recordSuccessLog 记录成功连接日志
func (s *StrictPostKeyCallback) recordSuccessLog(host string, port int, knownHost models.FIMKnownHost) {
	connectionLog := models.FIMSSHConnectionLog{
		Hostname:            host,
		Port:                port,
		Result:              "success",
		ExpectedFingerprint: &knownHost.FingerprintSHA256,
		ServerID:            s.serverID,
		PolicyID:            s.policyID,
		SnapshotID:          s.snapshotID,
	}
	database.DB.Create(&connectionLog)
}

// updateHostUsage 更新主机使用统计
func (s *StrictPostKeyCallback) updateHostUsage(hostID uint) {
	now := time.Now()
	database.DB.Model(&models.FIMKnownHost{}).Where("id = ?", hostID).Updates(map[string]interface{}{
		"last_used_at": &now,
		"use_count":    "use_count + 1",
	})
}

// parseHostPort 解析主机和端口
func (s *StrictPostKeyCallback) parseHostPort(hostname string, remote net.Addr) (string, int) {
	host := hostname
	port := 22

	// 尝试从hostname解析端口（格式：[host]:port 或 host:port）
	if strings.Contains(hostname, ":") && !strings.Contains(hostname, "::") {
		// IPv6地址格式：[::1]:22 或 IPv4格式：host:22
		if strings.HasPrefix(hostname, "[") {
			// IPv6格式：[::1]:22
			closingBracket := strings.Index(hostname, "]")
			if closingBracket > 0 && closingBracket < len(hostname)-2 {
				host = hostname[1:closingBracket]
				fmt.Sscanf(hostname[closingBracket+2:], "%d", &port)
			}
		} else {
			// IPv4格式：host:22
			parts := strings.Split(hostname, ":")
			if len(parts) == 2 {
				host = parts[0]
				fmt.Sscanf(parts[1], "%d", &port)
			}
		}
	}

	// 尝试从remote地址解析端口
	if remote != nil {
		if _, portStr, err := net.SplitHostPort(remote.String()); err == nil {
			fmt.Sscanf(portStr, "%d", &port)
		}
	}

	return host, port
}

// 辅助函数：将字符串转为指针
func ptrString(s string) *string {
	return &s
}
