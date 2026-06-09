// SPDX-License-Identifier: MIT
// Copyright (c) 2026 OPS Platform Contributors

// @title           OPS Platform API
// @version         1.0
// @description     运维管理平台 API 文档
// @contact.name   OPS Platform
// @contact.url    https://github.com/jenvenson/ops-platform
// @license.name  MIT
// @license.url   https://github.com/jenvenson/ops-platform/blob/main/LICENSE
// @host           localhost:8080
// @BasePath       /api
// @schemes        http https
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization

package main

import (
	"fmt"
	"os"
	"time"

	_ "github.com/jenvenson/ops-platform/docs"
	"github.com/jenvenson/ops-platform/internal/server"
)

func init() {
	// 设置时区，默认 Asia/Shanghai，可通过 TZ 环境变量覆盖
	tz := os.Getenv("TZ")
	if tz == "" {
		tz = "Asia/Shanghai"
	}
	loc, err := time.LoadLocation(tz)
	if err == nil {
		time.Local = loc
	}
}

func main() {
	if err := server.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start server: %v\n", err)
		os.Exit(1)
	}
}
