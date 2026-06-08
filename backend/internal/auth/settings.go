package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/edy/ops-platform/internal/audit"
	"github.com/edy/ops-platform/internal/database"
	"github.com/edy/ops-platform/internal/models"
	"github.com/edy/ops-platform/internal/secureconfig"
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
		audit.SetOperationAuditSummary(c, "更新了平台审计开关配置。")
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
		audit.SetOperationAuditSummary(c, "更新了 FIM SSH 配置。")
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
