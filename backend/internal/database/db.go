package database

import (
	"fmt"
	"time"

	"github.com/jenvenson/ops-platform/internal/models"
	"github.com/jenvenson/ops-platform/pkg/config"
	"github.com/jenvenson/ops-platform/pkg/logger"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init(cfg *config.Config, log *logger.Logger) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Asia%%2FShanghai",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
	)

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}

	sqlDB, _ := DB.DB()
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 设置会话字符集
	sqlDB.Exec("SET NAMES utf8mb4 COLLATE utf8mb4_unicode_ci")

	// 自动迁移
	if err := DB.AutoMigrate(
		&models.User{},
		&models.Role{},
		&models.Menu{},
		&models.RoleMenu{},
		&models.AuditLogSetting{},
		&models.FIMSSHSetting{},
		&models.AssistantModelSetting{},
		&models.SystemGeneralSetting{},
			&models.PasswordResetToken{},
		&models.PlatformAccessLog{},
		&models.PlatformAccessLogArchive{},
		&models.PlatformAuditLog{},
		&models.PlatformAuditLogArchive{},
		&models.PlatformLoginLog{},
		&models.PlatformLoginLogArchive{},
	); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Info("Database connected and migrated")
	return nil
}
