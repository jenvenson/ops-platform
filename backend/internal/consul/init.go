// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package consul

import (
	"github.com/jenvenson/ops-platform/internal/database"
)

// Init 初始化 Consul 模块数据库表
func Init() error {
	if err := database.DB.AutoMigrate(
		&ConsulConfig{},
		&ReplaceRule{},
		&CopyOperation{},
	); err != nil {
		return err
	}
	return nil
}