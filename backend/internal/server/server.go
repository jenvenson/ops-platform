package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jenvenson/ops-platform/internal/alert"
	"github.com/jenvenson/ops-platform/internal/audit"
	"github.com/jenvenson/ops-platform/internal/assistant"
	"github.com/jenvenson/ops-platform/internal/auth"
	"github.com/jenvenson/ops-platform/internal/cicd"
	"github.com/jenvenson/ops-platform/internal/cmdb"
	"github.com/jenvenson/ops-platform/internal/consul"
	"github.com/jenvenson/ops-platform/internal/database"
	"github.com/jenvenson/ops-platform/internal/models"
	"github.com/jenvenson/ops-platform/internal/monitor"
	"github.com/jenvenson/ops-platform/internal/platformevent"
	"github.com/jenvenson/ops-platform/internal/platformobject"
	"github.com/jenvenson/ops-platform/internal/security"
	"github.com/jenvenson/ops-platform/internal/tasks" // 添加tasks导入
	"github.com/jenvenson/ops-platform/pkg/config"
	"github.com/jenvenson/ops-platform/pkg/jenkins" // 添加jenkins导入
	"github.com/jenvenson/ops-platform/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	log, err := logger.New(cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}
	defer func() {
		if log != nil {
			log.Sync()
		}
	}()

	// 初始化数据库
	if err := database.Init(cfg, log); err != nil {
		return fmt.Errorf("failed to init database: %w", err)
	}

	// 初始化聚合历史表
	if err := database.InitAggregatedHistoryTable(database.DB, log); err != nil {
		return fmt.Errorf("failed to init aggregated history table: %w", err)
	}

	// 初始化 CMDB 模型
	if err := cmdb.Init(); err != nil {
		return fmt.Errorf("failed to init cmdb: %w", err)
	}

	// 初始化告警模块
	if err := alert.Init(); err != nil {
		return fmt.Errorf("failed to init alert: %w", err)
	}

	// 初始化 Consul 模块
	if err := consul.Init(); err != nil {
		return fmt.Errorf("failed to init consul: %w", err)
	}

	// 初始化统一对象索引
	if err := platformobject.Init(); err != nil {
		log.Warn("Failed to initialize platform object index: " + err.Error())
	}

	// 初始化统一事件流
	if err := platformevent.Init(); err != nil {
		log.Warn("Failed to initialize platform event stream: " + err.Error())
	}

	// 初始化默认角色
	if err := initDefaultRoles(log); err != nil {
		return fmt.Errorf("failed to init default roles: %w", err)
	}

	r := gin.Default()

	// 设置全局中间件以确保正确的字符编码
	r.Use(func(c *gin.Context) {
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.Next()
	})
	r.Use(audit.Middleware())

	// Health check
	healthHandler := func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
	r.GET("/health", healthHandler)
	r.GET("/api/health", healthHandler)

	// Prometheus metrics endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// 注册认证路由
	auth.RegisterRoutes(r, cfg)

	// 注册 CMDB 路由
	cmdb.RegisterRoutes(r, cfg)

	// 注册聚合打包路由
	aggHandler := &cmdb.Handler{Cfg: cfg}
	aggHandler.RegisterAggregatePackageRoutes(r, cfg)

	// 创建聚合历史轮询的停止通道
	aggHistoryStopChan := make(chan struct{})

	// 启动聚合历史记录的Jenkins状态定时刷新
	go cmdb.ScheduleAggregatedHistoryRefresh(
		jenkins.NewClient(
			cfg.Jenkins.URL,
			cfg.Jenkins.Username,
			cfg.Jenkins.Token,
			time.Duration(cfg.Jenkins.Timeout)*time.Second,
		),
		time.Duration(cfg.Jenkins.PollInterval)*time.Second,
		aggHistoryStopChan,
	)

	// 启动任务管理器的清理例程
	taskManager := tasks.GetDefaultTaskManager()
	taskManager.StartCleanupRoutine()

	// 注册 CI/CD 路由
	cicd.RegisterRoutes(r, cfg)

	// 注册监控路由
	monitor.RegisterRoutes(r, cfg)

	// 注册告警路由
	alert.RegisterRoutes(r, cfg)

	// 注册安全扫描路由
	security.RegisterRoutes(r, cfg)

	// 注册 Assistant 路由
	assistant.RegisterRoutes(r, cfg)

	// 注册统一事件流路由
	platformevent.RegisterRoutes(r, cfg)

	// 初始化 Assistant 模块
	if err := assistant.Init(); err != nil {
		log.Warn("Failed to initialize assistant module: " + err.Error())
	}

	// 注册 Consul 管理路由
	consulService := consul.NewService(database.DB)
	consulHandler := consul.NewHandler(consulService)
	consulGroup := r.Group("/api")
	consulGroup.Use(auth.AuthMiddleware(cfg.JWT.Secret))
	consulHandler.RegisterRoutes(consulGroup)

	// 启动服务器健康检查定时任务
	go monitor.StartChecker()

	addr := fmt.Sprintf(":%d", cfg.Port)
	srvr := &http.Server{
		Addr:         addr,
		Handler:      r,
		IdleTimeout:  3600 * time.Second,  // 1小时空闲超时
		ReadTimeout:  3600 * time.Second,  // 1小时读取超时
		WriteTimeout: 10800 * time.Second, // 3小时写入超时（为长时间任务留更多空间）
	}

	log.Info("Starting server on " + addr)

	go func() {
		if err := srvr.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed: " + err.Error())
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// 停止聚合历史轮询
	close(aggHistoryStopChan)
	cmdb.StopAggregatedHistoryRefresh()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srvr.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	log.Info("Server stopped")
	return nil
}

// initDefaultRoles 初始化默认角色
func initDefaultRoles(log *logger.Logger) error {
	roles := []struct {
		Name        string
		Code        string
		Description string
	}{
		{"超级管理员", "admin", "拥有所有权限"},
		{"运维人员", "ops", "负责系统运维工作"},
		{"开发人员", "dev", "负责应用开发"},
		{"普通用户", "user", "普通用户角色"},
	}

	for _, role := range roles {
		var existing models.Role
		if err := database.DB.Where("code = ?", role.Code).First(&existing).Error; err == nil {
			continue // 角色已存在
		}

		newRole := models.Role{
			Name:        role.Name,
			Code:        role.Code,
			Description: role.Description,
			Status:      1,
		}
		if err := database.DB.Create(&newRole).Error; err != nil {
			log.Warn("Failed to create role: " + role.Code)
		} else {
			log.Info("Created default role: " + role.Name)
		}
	}
	return nil
}
