// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package cmdb

import (
	"log"

	"github.com/jenvenson/ops-platform/internal/database"
	"github.com/jenvenson/ops-platform/internal/models"
)

// Init 数据库初始化
func Init() error {
	// 自动迁移所有模型，忽略可能的索引冲突错误
	if err := database.DB.AutoMigrate(
		&Project{},
		&Environment{},
		&Server{},
		&Application{},
		&ServerApp{},
		&Tag{},
		&AssetTag{},
		&DeployRecord{},
		&ArchiveRecord{},
		&models.AggregatePackageTask{},
		&models.AggregatePackageResult{},
	); err != nil {
		// 记录错误但不阻止启动
		log.Printf("Warning: migration warning (app may still work): %v", err)
	}
	return nil
}