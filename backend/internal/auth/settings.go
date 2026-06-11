// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jenvenson/ops-platform/internal/audit"
	"github.com/jenvenson/ops-platform/internal/database"
	"github.com/jenvenson/ops-platform/internal/models"
	"github.com/jenvenson/ops-platform/internal/secureconfig"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

type UpdateFIMSSHSettingRequest struct {
	AuthMode   string `json:"auth_mode" binding:"required"`
	SSHUser    string `json:"ssh_user" binding:"required"`
	Password   string `json:"password"`
	PrivateKey string `json:"private_key"`
	TimeoutSec int    `json:"timeout_sec"`
}

type FIMSSHSettingResponse struct {
	AuthMode             string `json:"auth_mode"`
	SSHUser              string `json:"ssh_user"`
	TimeoutSec           int    `json:"timeout_sec"`
	PasswordConfigured   bool   `json:"password_configured"`
	PrivateKeyConfigured bool   `json:"private_key_configured"`
}

type TestFIMSSHConnectionRequest struct {
	Host string `json:"host" binding:"required"`
	Port int    `json:"port"`
}

type TestFIMSSHConnectionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Output  string `json:"output,omitempty"`
}

type AuditLogSettingResponse struct {
	AccessLogEnabled    bool `json:"access_log_enabled"`
	OperationLogEnabled bool `json:"operation_log_enabled"`
	LoginLogEnabled     bool `json:"login_log_enabled"`
}

type UpdateAuditLogSettingRequest struct {
	AccessLogEnabled    bool `json:"access_log_enabled"`
	OperationLogEnabled bool `json:"operation_log_enabled"`
	LoginLogEnabled     bool `json:"login_log_enabled"`
}

func GetFIMSSHSetting() gin.HandlerFunc {
	return func(c *gin.Context) {
		setting, err := loadFIMSSHSettingModel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load fim ssh settings"})
			return
		}
		c.JSON(http.StatusOK, buildFIMSSHSettingResponse(setting))
	}
}

func GetAuditLogSetting() gin.HandlerFunc {
	return func(c *gin.Context) {
		setting, err := loadAuditLogSettingModel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load audit log settings"})
			return
		}
		c.JSON(http.StatusOK, buildAuditLogSettingResponse(setting))
	}
}

func UpdateAuditLogSetting() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req UpdateAuditLogSettingRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		current, err := loadAuditLogSettingModel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load audit log settings"})
			return
		}
		if current == nil {
			current = &models.AuditLogSetting{}
		}
		before := buildAuditLogSettingResponse(current)

		current.AccessLogEnabled = req.AccessLogEnabled
		current.OperationLogEnabled = req.OperationLogEnabled
		current.LoginLogEnabled = req.LoginLogEnabled
		current.UpdatedBy = currentAdminUsername(c)
		audit.SetOperationAuditBefore(c, before)
		if err := database.DB.Save(current).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save audit log settings"})
			return
		}
		resp := buildAuditLogSettingResponse(current)
		audit.SetOperationAuditAfter(c, resp)
		audit.SetOperationAuditSummary(c, "Updated platform audit log settings.")
		c.JSON(http.StatusOK, resp)
	}
}

func UpdateFIMSSHSetting() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req UpdateFIMSSHSettingRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}
		authMode := strings.TrimSpace(req.AuthMode)
		if authMode != "password" && authMode != "private_key" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid auth_mode"})
			return
		}
		sshUser := strings.TrimSpace(req.SSHUser)
		if sshUser == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ssh_user is required"})
			return
		}
		timeoutSec := req.TimeoutSec
		if timeoutSec <= 0 {
			timeoutSec = 15
		}

		current, err := loadFIMSSHSettingModel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load fim ssh settings"})
			return
		}
		if current == nil {
			current = &models.FIMSSHSetting{}
		}
		before := buildFIMSSHSettingResponse(current)

		passwordEncrypted := current.PasswordEncrypted
		privateKeyEncrypted := current.PrivateKeyEncrypted

		if password := strings.TrimSpace(req.Password); password != "" {
			encrypted, err := secureconfig.EncryptString(password)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt fim password"})
				return
			}
			passwordEncrypted = encrypted
		}
		if privateKey := strings.TrimSpace(req.PrivateKey); privateKey != "" {
			encrypted, err := secureconfig.EncryptString(privateKey)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt fim private key"})
				return
			}
			privateKeyEncrypted = encrypted
		}

		if authMode == "password" {
			if passwordEncrypted == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "password is required for password auth"})
				return
			}
			privateKeyEncrypted = ""
		}
		if authMode == "private_key" {
			if privateKeyEncrypted == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "private_key is required for private_key auth"})
				return
			}
			passwordEncrypted = ""
		}

		current.AuthMode = authMode
		current.SSHUser = sshUser
		current.PasswordEncrypted = passwordEncrypted
		current.PrivateKeyEncrypted = privateKeyEncrypted
		current.TimeoutSec = timeoutSec
		current.UpdatedBy = currentAdminUsername(c)
		audit.SetOperationAuditBefore(c, before)

		if err := database.DB.Save(current).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save fim ssh settings"})
			return
		}

		resp := buildFIMSSHSettingResponse(current)
		audit.SetOperationAuditAfter(c, resp)
		audit.SetOperationAuditSummary(c, "Updated FIM SSH config.")
		c.JSON(http.StatusOK, resp)
	}
}

func TestFIMSSHConnection() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req TestFIMSSHConnectionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		host := strings.TrimSpace(req.Host)
		if host == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "host is required"})
			return
		}
		port := req.Port
		if port <= 0 {
			port = 22
		}

		clientConfig, err := buildFIMSSHClientConfig()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		address := fmt.Sprintf("%s:%d", host, port)
		client, err := ssh.Dial("tcp", address, clientConfig)
		if err != nil {
			c.JSON(http.StatusOK, TestFIMSSHConnectionResponse{
				Success: false,
				Message: fmt.Sprintf("连接失败: %v", err),
			})
			return
		}
		defer client.Close()

		session, err := client.NewSession()
		if err != nil {
			c.JSON(http.StatusOK, TestFIMSSHConnectionResponse{
				Success: false,
				Message: fmt.Sprintf("SSH 会话创建失败: %v", err),
			})
			return
		}
		defer session.Close()

		output, err := session.CombinedOutput("hostname")
		if err != nil {
			c.JSON(http.StatusOK, TestFIMSSHConnectionResponse{
				Success: false,
				Message: fmt.Sprintf("连接成功但执行测试命令失败: %v", err),
				Output:  strings.TrimSpace(string(output)),
			})
			return
		}

		c.JSON(http.StatusOK, TestFIMSSHConnectionResponse{
			Success: true,
			Message: fmt.Sprintf("连接成功: %s", address),
			Output:  strings.TrimSpace(string(output)),
		})
	}
}

func loadFIMSSHSettingModel() (*models.FIMSSHSetting, error) {
	var setting models.FIMSSHSetting
	if err := database.DB.First(&setting).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &setting, nil
}

func loadAuditLogSettingModel() (*models.AuditLogSetting, error) {
	var setting models.AuditLogSetting
	query := database.DB.Order("id ASC").Limit(1).Find(&setting)
	if query.Error != nil {
		return nil, query.Error
	}
	if query.RowsAffected == 0 {
		return nil, nil
	}
	return &setting, nil
}

func ensureDefaultAuditLogSetting() {
	if database.DB == nil {
		return
	}

	var setting models.AuditLogSetting
	query := database.DB.Order("id ASC").Limit(1).Find(&setting)
	if query.Error != nil || query.RowsAffected > 0 {
		return
	}

	_ = database.DB.Create(&models.AuditLogSetting{
		AccessLogEnabled:    true,
		OperationLogEnabled: true,
		LoginLogEnabled:     true,
		UpdatedBy:           "system",
	}).Error
}

func buildFIMSSHSettingResponse(setting *models.FIMSSHSetting) FIMSSHSettingResponse {
	if setting == nil {
		return FIMSSHSettingResponse{
			AuthMode:             "password",
			TimeoutSec:           15,
			PasswordConfigured:   false,
			PrivateKeyConfigured: false,
		}
	}
	return FIMSSHSettingResponse{
		AuthMode:             setting.AuthMode,
		SSHUser:              setting.SSHUser,
		TimeoutSec:           setting.TimeoutSec,
		PasswordConfigured:   setting.PasswordEncrypted != "",
		PrivateKeyConfigured: setting.PrivateKeyEncrypted != "",
	}
}

func buildAuditLogSettingResponse(setting *models.AuditLogSetting) AuditLogSettingResponse {
	if setting == nil {
		return AuditLogSettingResponse{
			AccessLogEnabled:    true,
			OperationLogEnabled: true,
			LoginLogEnabled:     true,
		}
	}
	return AuditLogSettingResponse{
		AccessLogEnabled:    setting.AccessLogEnabled,
		OperationLogEnabled: setting.OperationLogEnabled,
		LoginLogEnabled:     setting.LoginLogEnabled,
	}
}

func currentAdminUsername(c *gin.Context) string {
	if username := strings.TrimSpace(c.GetString("username")); username != "" {
		return username
	}
	return "system"
}

func buildFIMSSHClientConfig() (*ssh.ClientConfig, error) {
	setting, err := loadFIMSSHSettingModel()
	if err != nil {
		return nil, err
	}
	if setting == nil {
		return nil, fmt.Errorf("FIM SSH settings are not configured")
	}

	user := strings.TrimSpace(setting.SSHUser)
	if user == "" {
		return nil, fmt.Errorf("ssh_user is not configured")
	}

	authMethods := make([]ssh.AuthMethod, 0, 1)
	switch setting.AuthMode {
	case "", "password":
		password, err := secureconfig.DecryptString(setting.PasswordEncrypted)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt fim ssh password: %w", err)
		}
		if strings.TrimSpace(password) == "" {
			return nil, fmt.Errorf("SSH password is not configured")
		}
		authMethods = append(authMethods, ssh.Password(password))
	case "private_key":
		privateKey, err := secureconfig.DecryptString(setting.PrivateKeyEncrypted)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt fim ssh private key: %w", err)
		}
		if strings.TrimSpace(privateKey) == "" {
			return nil, fmt.Errorf("SSH private key is not configured")
		}
		signer, err := ssh.ParsePrivateKey([]byte(privateKey))
		if err != nil {
			return nil, fmt.Errorf("failed to parse fim ssh private key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	default:
		return nil, fmt.Errorf("unsupported auth mode: %s", setting.AuthMode)
	}

	timeoutSec := setting.TimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = 15
	}

	return &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Duration(timeoutSec) * time.Second,
	}, nil
}

type UpdateAssistantModelSettingRequest struct {
	Provider    string  `json:"provider"`
	Enabled     bool    `json:"enabled"`
	APIKey      string  `json:"api_key"`
	BaseURL     string  `json:"base_url"`
	ChatModel   string  `json:"chat_model"`
	EmbedModel  string  `json:"embed_model"`
	Temperature float64 `json:"temperature"`
	TimeoutSec  int     `json:"timeout_sec"`
}

type AssistantModelSettingResponse struct {
	Provider          string  `json:"provider"`
	Enabled           bool    `json:"enabled"`
	APIKeyConfigured  bool    `json:"api_key_configured"`
	BaseURL           string  `json:"base_url"`
	ChatModel         string  `json:"chat_model"`
	EmbedModel        string  `json:"embed_model"`
	Temperature       float64 `json:"temperature"`
	TimeoutSec        int     `json:"timeout_sec"`
}

func GetAssistantModelSetting() gin.HandlerFunc {
	return func(c *gin.Context) {
		setting, err := loadAssistantModelSettingModel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load model settings"})
			return
		}
		c.JSON(http.StatusOK, buildAssistantModelSettingResponse(setting))
	}
}

func UpdateAssistantModelSetting() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req UpdateAssistantModelSettingRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request: %v", err)})
			return
		}

		current, err := loadAssistantModelSettingModel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load model settings"})
			return
		}
		if current == nil {
			current = &models.AssistantModelSetting{}
		}

		audit.SetOperationAuditBefore(c, buildAssistantModelSettingResponse(current))

		current.Provider = strings.TrimSpace(req.Provider)
		current.Enabled = req.Enabled
		current.BaseURL = strings.TrimSpace(req.BaseURL)
		current.ChatModel = strings.TrimSpace(req.ChatModel)
		current.EmbedModel = strings.TrimSpace(req.EmbedModel)
		current.Temperature = req.Temperature
		current.TimeoutSec = req.TimeoutSec
		current.UpdatedBy = currentAdminUsername(c)

		if req.APIKey != "" {
			encryptedKey, err := secureconfig.EncryptString(req.APIKey)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt api key"})
				return
			}
			current.APIKey = encryptedKey
		}

		if err := database.DB.Save(current).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save model settings"})
			return
		}

		audit.SetOperationAuditAfter(c, buildAssistantModelSettingResponse(current))
		audit.SetOperationAuditSummary(c, fmt.Sprintf("Updated assistant model settings: provider=%s enabled=%t model=%s",
			current.Provider, current.Enabled, current.ChatModel))

		go tryReloadAssistantProvider(current)

		c.JSON(http.StatusOK, gin.H{"message": "模型配置已保存，将在下一次请求生效"})
	}
}

func loadAssistantModelSettingModel() (*models.AssistantModelSetting, error) {
	var setting models.AssistantModelSetting
	if err := database.DB.First(&setting).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &setting, nil
}

func ensureDefaultAssistantModelSetting() {
	if database.DB == nil {
		return
	}
	var setting models.AssistantModelSetting
	if err := database.DB.First(&setting).Error; err == nil {
		tryReloadAssistantProvider(&setting)
		return
	}
	_ = database.DB.Create(&models.AssistantModelSetting{
		Provider:    "ollama",
		Enabled:     false,
		Temperature: 0.2,
		TimeoutSec:  20,
		UpdatedBy:   "system",
	}).Error
}

func buildAssistantModelSettingResponse(setting *models.AssistantModelSetting) AssistantModelSettingResponse {
	if setting == nil {
		return AssistantModelSettingResponse{
			Provider:          "ollama",
			Enabled:           false,
			APIKeyConfigured:  false,
			Temperature:       0.2,
			TimeoutSec:        20,
		}
	}
	return AssistantModelSettingResponse{
		Provider:          setting.Provider,
		Enabled:           setting.Enabled,
		APIKeyConfigured:  setting.APIKey != "",
		BaseURL:           setting.BaseURL,
		ChatModel:         setting.ChatModel,
		EmbedModel:        setting.EmbedModel,
		Temperature:       setting.Temperature,
		TimeoutSec:        setting.TimeoutSec,
	}
}

func decryptAndGetSettingAPIKey(setting *models.AssistantModelSetting) string {
	if setting == nil || setting.APIKey == "" {
		return ""
	}
	decrypted, err := secureconfig.DecryptString(setting.APIKey)
	if err != nil {
		return ""
	}
	return decrypted
}

func tryReloadAssistantProvider(setting *models.AssistantModelSetting) {
	if setting == nil || !setting.Enabled {
		return
	}
	currentAssistantMu.Lock()
	defer currentAssistantMu.Unlock()

	apiKey := decryptAndGetSettingAPIKey(setting)
	currentAssistantConfig = &assistantRuntimeConfig{
		Provider:    setting.Provider,
		Enabled:     setting.Enabled,
		APIKey:      apiKey,
		BaseURL:     setting.BaseURL,
		ChatModel:   setting.ChatModel,
		EmbedModel:  setting.EmbedModel,
		Temperature: setting.Temperature,
		TimeoutSec:  setting.TimeoutSec,
		dirty:       true,
	}
}

var (
	currentAssistantMu     sync.Mutex
	currentAssistantConfig *assistantRuntimeConfig
)

type assistantRuntimeConfig struct {
	Provider    string
	Enabled     bool
	APIKey      string
	BaseURL     string
	ChatModel   string
	EmbedModel  string
	Temperature float64
	TimeoutSec  int
	dirty       bool
}

// AssistantRuntimeConfig is the exported view of the current assistant config.
type AssistantRuntimeConfig struct {
	Provider    string
	Enabled     bool
	APIKey      string
	BaseURL     string
	ChatModel   string
	EmbedModel  string
	Temperature float64
	TimeoutSec  int
}

// FetchAssistantRuntimeConfig returns the runtime config if it has changed since last call.
func FetchAssistantRuntimeConfig() *AssistantRuntimeConfig {
	currentAssistantMu.Lock()
	defer currentAssistantMu.Unlock()
	if currentAssistantConfig == nil || !currentAssistantConfig.dirty {
		return nil
	}
	currentAssistantConfig.dirty = false
	return &AssistantRuntimeConfig{
		Provider:    currentAssistantConfig.Provider,
		Enabled:     currentAssistantConfig.Enabled,
		APIKey:      currentAssistantConfig.APIKey,
		BaseURL:     currentAssistantConfig.BaseURL,
		ChatModel:   currentAssistantConfig.ChatModel,
		EmbedModel:  currentAssistantConfig.EmbedModel,
		Temperature: currentAssistantConfig.Temperature,
		TimeoutSec:  currentAssistantConfig.TimeoutSec,
	}
}

// ---- 系统通用配置 ----

type SystemGeneralSettingResponse struct {
	SiteName string `json:"site_name"`
	Timezone string `json:"timezone"`
	Language string `json:"language"`
}

type UpdateSystemGeneralSettingRequest struct {
	SiteName string `json:"site_name"`
	Timezone string `json:"timezone"`
	Language string `json:"language"`
}

func ensureDefaultSystemGeneralSetting() {
	if database.DB == nil {
		return
	}
	var setting models.SystemGeneralSetting
	if err := database.DB.Order("id ASC").Limit(1).Find(&setting).Error; err != nil || setting.ID != 0 {
		return
	}
	_ = database.DB.Create(&models.SystemGeneralSetting{
		SiteName:  "运维管理平台",
		Timezone:  "Asia/Shanghai",
		Language:  "zh-CN",
		UpdatedBy: "system",
	}).Error
}

func GetSystemGeneralSetting() gin.HandlerFunc {
	return func(c *gin.Context) {
		var setting models.SystemGeneralSetting
		if err := database.DB.Order("id ASC").Limit(1).Find(&setting).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "读取系统配置失败"})
			return
		}
		c.JSON(http.StatusOK, SystemGeneralSettingResponse{
			SiteName: setting.SiteName,
			Timezone: setting.Timezone,
			Language: setting.Language,
		})
	}
}

func UpdateSystemGeneralSetting() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req UpdateSystemGeneralSettingRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效"})
			return
		}

		var setting models.SystemGeneralSetting
		if err := database.DB.Order("id ASC").Limit(1).Find(&setting).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "读取系统配置失败"})
			return
		}

		if req.SiteName != "" {
			setting.SiteName = req.SiteName
		}
		if req.Timezone != "" {
			setting.Timezone = req.Timezone
		}
		if req.Language != "" {
			setting.Language = req.Language
		}
		setting.UpdatedBy = c.GetString("username")
		setting.UpdatedAt = time.Now()

		if err := database.DB.Save(&setting).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存系统配置失败"})
			return
		}
		c.JSON(http.StatusOK, SystemGeneralSettingResponse{
			SiteName: setting.SiteName,
			Timezone: setting.Timezone,
			Language: setting.Language,
		})
	}
}

// GetPublicSiteName 无需认证，供前端页面标题等公共场景使用。
func GetPublicSiteName() gin.HandlerFunc {
	return func(c *gin.Context) {
		var setting models.SystemGeneralSetting
		if err := database.DB.Order("id ASC").Limit(1).Find(&setting).Error; err != nil || setting.SiteName == "" {
			c.JSON(http.StatusOK, gin.H{"site_name": "运维管理平台"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"site_name": setting.SiteName})
	}
}