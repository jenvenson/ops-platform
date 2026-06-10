// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package security

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jenvenson/ops-platform/internal/database"
	"github.com/jenvenson/ops-platform/internal/models"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/ssh"
)

// strPtr returns a pointer to the given string
func strPtr(s string) *string {
	return &s
}

// getOperatorName 获取操作人名称（优先使用真实姓名，否则使用用户名）
func getOperatorName(c *gin.Context) string {
	if realName := strings.TrimSpace(c.GetString("real_name")); realName != "" {
		return realName
	}
	return c.GetString("username")
}

// ListKnownHosts 获取已知主机列表
func ListKnownHosts(c *gin.Context) {
	hostname := c.Query("hostname")
	status := c.Query("status")
	keyType := c.Query("key_type")

	query := database.DB.Model(&models.FIMKnownHost{})

	if hostname != "" {
		query = query.Where("hostname LIKE ?", "%"+hostname+"%")
	}
	if status == "enabled" {
		query = query.Where("is_enabled = ?", true)
	} else if status == "disabled" {
		query = query.Where("is_enabled = ?", false)
	}
	if keyType != "" {
		query = query.Where("key_type = ?", keyType)
	}

	var hosts []models.FIMKnownHost
	if err := query.Order("added_at DESC").Find(&hosts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query known hosts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  hosts,
		"total": len(hosts),
	})
}

// GetKnownHost 获取单个已知主机详情
func GetKnownHost(c *gin.Context) {
	id := c.Param("id")

	var host models.FIMKnownHost
	if err := database.DB.First(&host, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "host not found"})
		return
	}

	// 获取使用历史
	var logs []models.FIMSSHConnectionLog
	database.DB.Where("hostname = ? AND port = ?", host.Hostname, host.Port).
		Order("attempted_at DESC").
		Limit(10).
		Find(&logs)

	// 获取变更历史
	var history []models.FIMKnownHostsHistory
	database.DB.Where("host_id = ?", host.ID).
		Order("operated_at DESC").
		Limit(10).
		Find(&history)

	c.JSON(http.StatusOK, gin.H{
		"host":    host,
		"logs":    logs,
		"history": history,
	})
}

// AddKnownHost 添加已知主机
func AddKnownHost(c *gin.Context) {
	var req struct {
		Hostname    string   `json:"hostname" binding:"required"`
		Port        int      `json:"port" binding:"required,min=1,max=65535"`
		KeyType     string   `json:"key_type" binding:"required"`
		PublicKey   string   `json:"public_key" binding:"required"`
		ServerID    *uint    `json:"server_id"`
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证公钥格式
	key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(req.PublicKey))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid public key format: " + err.Error()})
		return
	}

	// 检查是否已存在
	var existing models.FIMKnownHost
	if err := database.DB.Where("hostname = ? AND port = ? AND key_type = ?",
		req.Hostname, req.Port, req.KeyType).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "host key already exists"})
		return
	}

	// 创建记录
	operator := getOperatorName(c)
	now := time.Now()

	tagsStr := "[]"
	if len(req.Tags) > 0 {
		tagsBytes, _ := json.Marshal(req.Tags)
		tagsStr = string(tagsBytes)
	}

	host := models.FIMKnownHost{
		Hostname:           req.Hostname,
		Port:               req.Port,
		KeyType:            req.KeyType,
		PublicKey:          strings.TrimSpace(req.PublicKey),
		FingerprintSHA256:  ssh.FingerprintSHA256(key),
		ServerID:           req.ServerID,
		Description:        req.Description,
		Tags:               tagsStr,
		VerificationStatus: "verified",
		VerifiedBy:         operator,
		VerifiedAt:         &now,
		AddedBy:            operator,
		AddedAt:            now,
		IsEnabled:          true,
	}

	if err := database.DB.Create(&host).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create known host"})
		return
	}

	// 记录历史
	history := models.FIMKnownHostsHistory{
		HostID:         host.ID,
		Action:         "added",
		NewKeyType:     &host.KeyType,
		NewPublicKey:   &host.PublicKey,
		NewFingerprint: &host.FingerprintSHA256,
		OperatedBy:     operator,
		IPAddress:      c.ClientIP(),
	}
	database.DB.Create(&history)

	c.JSON(http.StatusCreated, host)
}

// UpdateKnownHost 更新已知主机
func UpdateKnownHost(c *gin.Context) {
	id := c.Param("id")

	var host models.FIMKnownHost
	if err := database.DB.First(&host, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "host not found"})
		return
	}

	var req struct {
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
		IsEnabled   *bool    `json:"is_enabled"`
		PublicKey   string   `json:"public_key"` // 如果提供，表示更新密钥
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	operator := getOperatorName(c)
	oldKey := host.PublicKey
	oldFingerprint := host.FingerprintSHA256

	updates := map[string]interface{}{
		"updated_by": operator,
	}

	if req.Description != "" {
		updates["description"] = req.Description
	}
	if len(req.Tags) > 0 {
		tagsBytes, _ := json.Marshal(req.Tags)
		updates["tags"] = string(tagsBytes)
	}
	if req.IsEnabled != nil {
		updates["is_enabled"] = *req.IsEnabled
	}

	// 如果提供了新密钥
	if req.PublicKey != "" {
		key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(req.PublicKey))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid public key format"})
			return
		}

		updates["public_key"] = strings.TrimSpace(req.PublicKey)
		updates["fingerprint_sha256"] = ssh.FingerprintSHA256(key)
		newKeyType := key.Type()
		updates["key_type"] = newKeyType
		updates["verification_status"] = "verified"
		updates["verified_by"] = operator
		now := time.Now()
		updates["verified_at"] = &now

		// 记录历史（密钥变更）
		history := models.FIMKnownHostsHistory{
			HostID:          host.ID,
			Action:          "key_changed",
			OldKeyType:      &host.KeyType,
			OldPublicKey:    &oldKey,
			OldFingerprint:  &oldFingerprint,
			NewKeyType:      &newKeyType,
			NewPublicKey:    &req.PublicKey,
			NewFingerprint:  strPtr(ssh.FingerprintSHA256(key)),
			OperatedBy:      operator,
			IPAddress:       c.ClientIP(),
		}
		database.DB.Create(&history)
	}

	// 更新数据库
	if err := database.DB.Model(&host).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update host"})
		return
	}

	// 记录历史
	history := models.FIMKnownHostsHistory{
		HostID:          host.ID,
		Action:          "updated",
		OldKeyType:      &host.KeyType,
		OldPublicKey:    &oldKey,
		OldFingerprint:  &oldFingerprint,
		OperatedBy:      operator,
		IPAddress:       c.ClientIP(),
	}
	database.DB.Create(&history)

	// 重新查询获取最新数据
	database.DB.First(&host, id)

	c.JSON(http.StatusOK, host)
}

// DeleteKnownHost 删除已知主机
func DeleteKnownHost(c *gin.Context) {
	id := c.Param("id")

	var host models.FIMKnownHost
	if err := database.DB.First(&host, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "host not found"})
		return
	}

	operator := getOperatorName(c)

	// 记录历史
	history := models.FIMKnownHostsHistory{
		HostID:          host.ID,
		Action:          "deleted",
		OldKeyType:      &host.KeyType,
		OldPublicKey:    &host.PublicKey,
		OldFingerprint:  &host.FingerprintSHA256,
		OperatedBy:      operator,
		IPAddress:       c.ClientIP(),
	}
	database.DB.Create(&history)

	// 删除
	if err := database.DB.Delete(&host).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete host"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ImportKnownHosts 从known_hosts格式导入
func ImportKnownHosts(c *gin.Context) {
	var req struct {
		Content string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	operator := getOperatorName(c)
	lines := strings.Split(req.Content, "\n")

	imported := 0
	skipped := 0
	errors := []string{}

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 解析known_hosts格式
		// 格式: [hostname]:port keytype publickey
		// 或: hostname keytype publickey
		parts := strings.Fields(line)
		if len(parts) < 3 {
			errors = append(errors, fmt.Sprintf("Line %d: invalid format", i+1))
			skipped++
			continue
		}

		hostPort := parts[0]
		var host string
		var port int

		// 解析主机和端口
		host = hostPort
		port = 22 // 默认端口
		if strings.HasPrefix(hostPort, "[") {
			// 格式: [hostname]:port
			closingBracket := strings.Index(hostPort, "]")
			if closingBracket > 0 && closingBracket < len(hostPort)-2 {
				host = hostPort[1:closingBracket]
				if n, _ := fmt.Sscanf(hostPort[closingBracket+2:], "%d", &port); n != 1 {
					port = 22 // 解析失败使用默认端口
				}
			}
		} else if strings.Contains(hostPort, ":") && !strings.Contains(hostPort, "::") {
			// 格式: hostname:port
			splitParts := strings.SplitN(hostPort, ":", 2)
			host = splitParts[0]
			if len(splitParts) > 1 {
				if n, _ := fmt.Sscanf(splitParts[1], "%d", &port); n != 1 {
					port = 22 // 解析失败使用默认端口
				}
			}
		}

		keyType := parts[1]
		publicKey := strings.Join(parts[2:], " ")

		// 验证公钥
		key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(publicKey))
		if err != nil {
			errors = append(errors, fmt.Sprintf("Line %d: invalid key - %s", i+1, err.Error()))
			skipped++
			continue
		}

		// 检查是否已存在
		var existing models.FIMKnownHost
		err = database.DB.Where("hostname = ? AND port = ? AND key_type = ?",
			host, port, keyType).First(&existing).Error
		if err == nil {
			skipped++
			continue
		}

		// 创建记录
		now := time.Now()
		newHost := models.FIMKnownHost{
			Hostname:           host,
			Port:               port,
			KeyType:            keyType,
			PublicKey:          publicKey,
			FingerprintSHA256:  ssh.FingerprintSHA256(key),
			VerificationStatus: "verified",
			VerifiedBy:         operator,
			VerifiedAt:         &now,
			AddedBy:            operator,
			AddedAt:            now,
			IsEnabled:          true,
		}

		if err := database.DB.Create(&newHost).Error; err != nil {
			errors = append(errors, fmt.Sprintf("Line %d: failed to import - %s", i+1, err.Error()))
			skipped++
			continue
		}

		imported++
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"imported": imported,
		"skipped":  skipped,
		"errors":   errors,
	})
}

// ExportKnownHosts 导出为known_hosts格式
func ExportKnownHosts(c *gin.Context) {
	var hosts []models.FIMKnownHost
	if err := database.DB.Where("is_enabled = ?", true).Find(&hosts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query hosts"})
		return
	}

	var lines []string
	for _, host := range hosts {
		var hostPort string
		if host.Port == 22 {
			hostPort = host.Hostname
		} else {
			hostPort = fmt.Sprintf("[%s]:%d", host.Hostname, host.Port)
		}
		lines = append(lines, fmt.Sprintf("%s %s %s", hostPort, host.KeyType, host.PublicKey))
	}

	content := strings.Join(lines, "\n")

	c.Header("Content-Disposition", "attachment; filename=known_hosts")
	c.Data(http.StatusOK, "text/plain", []byte(content))
}

// GetConnectionLogs 获取连接尝试日志
func GetConnectionLogs(c *gin.Context) {
	hostname := c.Query("hostname")
	result := c.Query("result")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))

	query := database.DB.Model(&models.FIMSSHConnectionLog{})

	if hostname != "" {
		query = query.Where("hostname LIKE ?", "%"+hostname+"%")
	}
	if result != "" {
		query = query.Where("result = ?", result)
	}

	var logs []models.FIMSSHConnectionLog
	if err := query.Order("attempted_at DESC").Limit(limit).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  logs,
		"total": len(logs),
	})
}

// BatchAddKnownHosts 批量添加主机密钥
func BatchAddKnownHosts(c *gin.Context) {
	var req struct {
		Hosts []struct {
			Hostname    string   `json:"hostname" binding:"required"`
			Port        int      `json:"port" binding:"required"`
			KeyType     string   `json:"key_type" binding:"required"`
			PublicKey   string   `json:"public_key" binding:"required"`
			Description string   `json:"description"`
			Tags        []string `json:"tags"`
		} `json:"hosts" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	operator := getOperatorName(c)
	now := time.Now()

	added := 0
	skipped := 0
	errors := []string{}

	for i, hostReq := range req.Hosts {
		// 验证公钥
		key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(hostReq.PublicKey))
		if err != nil {
			errors = append(errors, fmt.Sprintf("Host %d: invalid key - %s", i+1, err.Error()))
			skipped++
			continue
		}

		// 检查是否已存在
		var existing models.FIMKnownHost
		err = database.DB.Where("hostname = ? AND port = ? AND key_type = ?",
			hostReq.Hostname, hostReq.Port, hostReq.KeyType).First(&existing).Error
		if err == nil {
			skipped++
			continue
		}

		// 创建记录
		tagsStr := "[]"
		if len(hostReq.Tags) > 0 {
			tagsBytes, _ := json.Marshal(hostReq.Tags)
			tagsStr = string(tagsBytes)
		}

		newHost := models.FIMKnownHost{
			Hostname:           hostReq.Hostname,
			Port:               hostReq.Port,
			KeyType:            hostReq.KeyType,
			PublicKey:          strings.TrimSpace(hostReq.PublicKey),
			FingerprintSHA256:  ssh.FingerprintSHA256(key),
			Description:        hostReq.Description,
			Tags:               tagsStr,
			VerificationStatus: "verified",
			VerifiedBy:         operator,
			VerifiedAt:         &now,
			AddedBy:            operator,
			AddedAt:            now,
			IsEnabled:          true,
		}

		if err := database.DB.Create(&newHost).Error; err != nil {
			errors = append(errors, fmt.Sprintf("Host %d: failed to add - %s", i+1, err.Error()))
			skipped++
			continue
		}

		// 记录历史
		history := models.FIMKnownHostsHistory{
			HostID:         newHost.ID,
			Action:         "added",
			NewKeyType:     &newHost.KeyType,
			NewPublicKey:   &newHost.PublicKey,
			NewFingerprint: &newHost.FingerprintSHA256,
			OperatedBy:     operator,
			IPAddress:      c.ClientIP(),
		}
		database.DB.Create(&history)

		added++
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"added":   added,
		"skipped": skipped,
		"errors":  errors,
	})
}