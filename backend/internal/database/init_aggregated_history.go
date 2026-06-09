package database

import (
	"github.com/jenvenson/ops-platform/internal/models"
	"github.com/jenvenson/ops-platform/pkg/logger"
	"gorm.io/gorm"
)

// InitAggregatedHistoryTable 初始化聚合历史表
func InitAggregatedHistoryTable(db *gorm.DB, log *logger.Logger) error {
	// 迁移聚合历史表结构
	if err := db.AutoMigrate(&models.AggregatedHistory{}); err != nil {
		log.Error("Failed to migrate aggregated history table: " + err.Error())
		return err
	}

	log.Info("Aggregated history table initialized successfully")
	return nil
}