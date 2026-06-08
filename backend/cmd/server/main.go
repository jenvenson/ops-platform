package main

import (
	"fmt"
	"os"
	"time"

	"github.com/edy/ops-platform/internal/server"
)

func init() {
	// 设置时区为 CST (中国标准时间 UTC+8)
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err == nil {
		time.Local = loc
		os.Setenv("TZ", "Asia/Shanghai")
	}
}

func main() {
	if err := server.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start server: %v\n", err)
		os.Exit(1)
	}
}
