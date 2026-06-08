# Phase 1 Implementation Plan - 基础框架搭建

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 完成运维管理平台的基础框架搭建，包括 Go 后端、React 前端、Docker Compose 部署环境、用户认证和 MySQL 数据库初始化。

**Architecture:**
- Go 单体应用，使用 Gin 框架，通过 internal 包隔离子系统边界
- React + Vite + TypeScript + Ant Design 前端
- Docker Compose 编排所有服务（Go、React、MySQL、Redis、Nginx）
- JWT Token 认证机制
- GORM + MySQL 数据持久化

**Tech Stack:** Go 1.21+, React 18, MySQL 8.0, Redis 7.0, Docker Compose 2.x

---

## Task 1: Go 后端项目脚手架

**Files:**
- Create: `backend/go.mod`
- Create: `backend/cmd/server/main.go`
- Create: `backend/internal/server/server.go`
- Create: `backend/pkg/config/config.go`
- Create: `backend/pkg/logger/logger.go`

**Step 1: Initialize Go module**

```bash
cd backend
go mod init github.com/edy/ops-platform
```

**Step 2: 创建 main.go**

```go
package main

import (
    "log"
    "github.com/edy/ops-platform/internal/server"
)

func main() {
    if err := server.Run(); err != nil {
        log.Fatalf("Failed to start server: %v", err)
    }
}
```

**Step 3: 创建 server.go**

```go
package server

import (
    "context"
    "fmt"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/edy/ops-platform/pkg/config"
    "github.com/edy/ops-platform/pkg/logger"
)

func Run() error {
    cfg := config.Load()
    log := logger.New(cfg.LogLevel)

    r := gin.Default()

    // Health check
    r.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"status": "ok"})
    })

    addr := fmt.Sprintf(":%d", cfg.Port)
    srv := &http.Server{
        Addr:        addr,
        Handler:     r,
        IdleTimeout:  120 * time.Second,
        ReadTimeout:  5 * time.Second,
        WriteTimeout: 5 * time.Second,
    }

    log.Info("Starting server on " + addr)

    go func() {
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatal("Server failed: " + err.Error())
        }
    }()

    return nil
}
```

**Step 4: 创建 config.go**

```go
package config

import (
    "os"

    "github.com/spf13/viper"
)

type Config struct {
    Port     int    `mapstructure:"port"`
    LogLevel string `mapstructure:"log_level"`
    Database struct {
        Host     string `mapstructure:"host"`
        Port     int    `mapstructure:"port"`
        User     string `mapstructure:"user"`
        Password string `mapstructure:"password"`
        Name     string `mapstructure:"name"`
    } `mapstructure:"database"`
    Redis struct {
        Host     string `mapstructure:"host"`
        Port     int    `mapstructure:"port"`
        Password string `mapstructure:"password"`
    } `mapstructure:"redis"`
    JWT struct {
        Secret string `mapstructure:"secret"`
    } `mapstructure:"jwt"`
}

func Load() *Config {
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath("./configs")
    viper.AddConfigPath(".")

    // 设置默认值
    viper.SetDefault("port", 8080)
    viper.SetDefault("log_level", "info")
    viper.SetDefault("database.port", 3306)
    viper.SetDefault("redis.port", 6379)

    if err := viper.ReadInConfig(); err != nil {
        // 使用环境变量覆盖
        viper.AutomaticEnv()
    }

    var cfg Config
    viper.Unmarshal(&cfg)

    // 从环境变量读取敏感配置
    cfg.Database.Password = os.Getenv("DB_PASSWORD")
    cfg.Redis.Password = os.Getenv("REDIS_PASSWORD")
    cfg.JWT.Secret = os.Getenv("JWT_SECRET")

    return &cfg
}
```

**Step 5: 创建 logger.go**

```go
package logger

import (
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

type Logger struct {
    *zap.Logger
}

func New(level string) *Logger {
    var zapLevel zapcore.Level
    switch level {
    case "debug":
        zapLevel = zapcore.DebugLevel
    case "info":
        zapLevel = zapcore.InfoLevel
    case "warn":
        zapLevel = zapcore.WarnLevel
    case "error":
        zapLevel = zapcore.ErrorLevel
    default:
        zapLevel = zapcore.InfoLevel
    }

    config := zap.Config{
        Level:            zap.NewAtomicLevelAt(zapLevel),
        Development:      false,
        Encoding:         "json",
        EncoderConfig:    zapcore.EncoderConfig{MessageKey: "msg"},
        OutputPaths:      []string{"stdout"},
        ErrorOutputPaths: []string{"stderr"},
    }

    logger, _ := config.Build()
    return &Logger{logger}
}

func (l *Logger) Info(msg string) {
    l.Logger.Info(msg)
}

func (l *Logger) Error(msg string) {
    l.Logger.Error(msg)
}

func (l *Logger) Fatal(msg string) {
    l.Logger.Fatal(msg)
}
```

**Step 6: 创建 config.yaml**

```yaml
port: 8080
log_level: info
database:
  host: mysql
  port: 3306
  user: root
  password: ""
  name: ops_platform
redis:
  host: redis
  port: 6379
  password: ""
jwt:
  secret: ""
```

**Step 7: 下载依赖并编译**

```bash
cd backend
go mod download
go build -o bin/server cmd/server/main.go
```

**Step 8: Commit**

```bash
git add backend/
git commit -m "feat: Go 后端项目脚手架搭建

- 使用 Gin 框架
- 集成 Viper 配置管理
- 集成 zap 结构化日志
- 提供 health check 接口"
```

---

## Task 2: React 前端项目脚手架

**Files:**
- Create: `frontend/package.json`
- Create: `frontend/vite.config.ts`
- Create: `frontend/tsconfig.json`
- Create: `frontend/src/main.tsx`
- Create: `frontend/src/App.tsx`
- Create: `frontend/src/api/client.ts`

**Step 1: 创建 package.json**

```json
{
  "name": "ops-platform-frontend",
  "version": "0.1.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc && vite build",
    "preview": "vite preview"
  },
  "dependencies": {
    "react": "^18.2.0",
    "react-dom": "^18.2.0",
    "react-router-dom": "^6.20.0",
    "antd": "^5.12.0",
    "axios": "^1.6.0"
  },
  "devDependencies": {
    "@types/react": "^18.2.0",
    "@types/react-dom": "^18.2.0",
    "@vitejs/plugin-react": "^4.2.0",
    "typescript": "^5.3.0",
    "vite": "^5.0.0"
  }
}
```

**Step 2: 创建 vite.config.ts**

```ts
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      }
    }
  }
})
```

**Step 3: 创建 tsconfig.json**

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "useDefineForClassFields": true,
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": true,
    "jsx": "react-jsx",
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true
  },
  "include": ["src"],
  "references": [{ "path": "./tsconfig.node.json" }]
}
```

**Step 4: 创建 tsconfig.node.json**

```json
{
  "compilerOptions": {
    "composite": true,
    "skipLibCheck": true,
    "module": "ESNext",
    "moduleResolution": "bundler",
    "allowSyntheticDefaultImports": true
  },
  "include": ["vite.config.ts"]
}
```

**Step 5: 创建 index.html**

```html
<!doctype html>
<html lang="zh-CN">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>运维管理平台</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
```

**Step 6: 创建 main.tsx**

```tsx
import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import 'antd/dist/reset.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
)
```

**Step 7: 创建 App.tsx**

```tsx
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { Layout } from 'antd'
import MainLayout from './components/MainLayout'
import LoginPage from './pages/LoginPage'

const { Content } = Layout

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/" element={<MainLayout />}>
          {/* 路由将在后续添加 */}
        </Route>
      </Routes>
    </BrowserRouter>
  )
}

export default App
```

**Step 8: 创建 API 客户端**

```ts
import axios from 'axios'

const apiClient = axios.create({
  baseURL: '/api',
  timeout: 10000,
})

// 请求拦截器
apiClient.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// 响应拦截器
apiClient.interceptors.response.use(
  (response) => response.data,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

export default apiClient
```

**Step 9: 创建基础组件**

```tsx
// src/components/MainLayout.tsx
import { Layout, Menu } from 'antd'
import { Outlet } from 'react-router-dom'

const { Header, Sider, Content } = Layout

export default function MainLayout() {
  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider>
        <div style={{ height: 32, margin: 16, color: 'white' }}>
          运维管理平台
        </div>
        <Menu theme="dark">
          <Menu.Item key="cmdb">CMDB</Menu.Item>
          <Menu.Item key="monitor">监控</Menu.Item>
          <Menu.Item key="cicd">CI/CD</Menu.Item>
        </Menu>
      </Sider>
      <Layout>
        <Header>运维管理平台</Header>
        <Content>
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  )
}
```

```tsx
// src/pages/LoginPage.tsx
import { useState } from 'react'
import { Form, Input, Button, message } from 'antd'
import { useNavigate } from 'react-router-dom'
import apiClient from '../api/client'

export default function LoginPage() {
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()

  const onFinish = async (values: any) => {
    setLoading(true)
    try {
      const result = await apiClient.post('/auth/login', values)
      localStorage.setItem('token', result.token)
      message.success('登录成功')
      navigate('/')
    } catch (error) {
      message.error('登录失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
      <Form onFinish={onFinish} style={{ width: 300 }}>
        <Form.Item name="username" rules={[{ required: true }]}>
          <Input placeholder="用户名" />
        </Form.Item>
        <Form.Item name="password" rules={[{ required: true }]}>
          <Input.Password placeholder="密码" />
        </Form.Item>
        <Form.Item>
          <Button type="primary" htmlType="submit" loading={loading} block>
            登录
          </Button>
        </Form.Item>
      </Form>
    </div>
  )
}
```

**Step 10: 安装依赖并构建**

```bash
cd frontend
pnpm install
pnpm build
```

**Step 11: Commit**

```bash
git add frontend/
git commit -m "feat: React 前端项目脚手架搭建

- 使用 Vite + TypeScript
- 集成 Ant Design
- 配置 API 客户端
- 基础布局和登录页面"
```

---

## Task 3: Docker Compose 部署环境

**Files:**
- Create: `deploy/docker-compose.yml`
- Create: `deploy/nginx.conf`
- Create: `deploy/Dockerfile.backend`
- Create: `deploy/Dockerfile.frontend`

**Step 1: 创建 docker-compose.yml**

```yaml
version: '3.8'

services:
  mysql:
    image: mysql:8.0
    container_name: ops-mysql
    environment:
      MYSQL_ROOT_PASSWORD: ${DB_PASSWORD:-change_me_in_production}
      MYSQL_DATABASE: ops_platform
    ports:
      - "3306:3306"
    volumes:
      - mysql-data:/var/lib/mysql
      - ./scripts/init.sql:/docker-entrypoint-initdb.d/init.sql
    networks:
      - ops-network

  redis:
    image: redis:7.0-alpine
    container_name: ops-redis
    ports:
      - "6379:6379"
    volumes:
      - redis-data:/data
    networks:
      - ops-network

  backend:
    build:
      context: .
      dockerfile: deploy/Dockerfile.backend
    container_name: ops-backend
    ports:
      - "8080:8080"
    environment:
      - DB_PASSWORD=${DB_PASSWORD:-change_me_in_production}
      - REDIS_PASSWORD=
      - JWT_SECRET=${JWT_SECRET:-secret123}
    depends_on:
      - mysql
      - redis
    networks:
      - ops-network

  frontend:
    build:
      context: .
      dockerfile: deploy/Dockerfile.frontend
    container_name: ops-frontend
    ports:
      - "3000:80"
    depends_on:
      - backend
    networks:
      - ops-network

  nginx:
    image: nginx:alpine
    container_name: ops-nginx
    ports:
      - "80:80"
    volumes:
      - ./deploy/nginx.conf:/etc/nginx/nginx.conf
    depends_on:
      - backend
      - frontend
    networks:
      - ops-network

volumes:
  mysql-data:
  redis-data:

networks:
  ops-network:
    driver: bridge
```

**Step 2: 创建 nginx.conf**

```nginx
events {
    worker_connections 1024;
}

http {
    upstream backend {
        server backend:8080;
    }

    upstream frontend {
        server frontend:80;
    }

    server {
        listen 80;

        location /api/ {
            proxy_pass http://backend;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
        }

        location / {
            proxy_pass http://frontend;
            proxy_set_header Host $host;
        }
    }
}
```

**Step 3: 创建 Dockerfile.backend**

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN go build -o server cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

COPY --from=builder /app/server .
COPY backend/configs ./configs

EXPOSE 8080
CMD ["./server"]
```

**Step 4: 创建 Dockerfile.frontend**

```dockerfile
FROM node:20-alpine AS builder

WORKDIR /app
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN npm install -g pnpm && pnpm install

COPY frontend/ ./
RUN pnpm build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

**Step 1: 创建 .env 文件**

```bash
DB_PASSWORD=change_me_in_production
JWT_SECRET=your-secret-key-change-in-production
```

**Step 2: 测试构建**

```bash
cd /path/to/ops-platform
docker-compose -f deploy/docker-compose.yml config
```

**Step 3: Commit**

```bash
git add deploy/
git commit -m "feat: Docker Compose 部署环境

- MySQL 8.0 数据库容器
- Redis 7.0 缓存容器

- Go 后端容器
- React 前端容器
- Nginx 反向代理"
```

---

## Task 4: 用户认证功能

**Files:**
- Create: `backend/scripts/init.sql`
- Create: `backend/internal/auth/handler.go`
- Create: `backend/internal/auth/middleware.go`
- Create: `backend/internal/auth/jwt.go`
- Create: `backend/internal/models/user.go`
- Create: `backend/internal/database/db.go`

**Step 1: 创建数据库初始化脚本**

```sql
-- backend/scripts/init.sql
CREATE DATABASE IF NOT EXISTS ops_platform CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE ops_platform;

CREATE TABLE IF NOT EXISTS users (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    email VARCHAR(100),
    role ENUM('admin', 'user') NOT NULL DEFAULT 'user',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_username (username),
    KEY idx_role (role)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户表';

-- 插入初始化管理员用户（初始口令不在文档中展示）
INSERT INTO users (username, password, email, role)
VALUES ('admin', '$2a$10$N9qo8uLOickgx2ZMRZoMy.Mo5Y6y5q3Z3Z5Y6y5q3Z3Z5Y6y5q3Z', 'admin@example.com', 'admin')
ON DUPLICATE KEY UPDATE username=username;
```

**Step 2: 创建 User 模型**

```go
package models

import "time"

type User struct {
    ID        uint      `json:"id" gorm:"primaryKey"`
    Username  string    `json:"username" gorm:"uniqueIndex;size:50;not null"`
    Password  string    `json:"-" gorm:"not null"`
    Email     string    `json:"email" gorm:"size:100"`
    Role      string    `json:"role" gorm:"size:10;default:'user'"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

func (User) TableName() string {
    return "users"
}
```

**Step 3: 创建数据库连接**

```go
package database

import (
    "fmt"
    "time"

    "gorm.io/driver/mysql"
    "gorm.io/gorm"
    "github.com/edy/ops-platform/pkg/config"
    "github.com/edy/ops-platform/pkg/logger"
    "github.com/edy/ops-platform/internal/models"
)

var DB *gorm.DB

func Init(cfg *config.Config, log *logger.Logger) error {
    dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
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

    // 自动迁移
    if err := DB.AutoMigrate(&models.User{}); err != nil {
        return fmt.Errorf("failed to migrate database: %w", err)
    }

    log.Info("Database connected and migrated")
    return nil
}
```

**Step 4: 创建 JWT 工具**

```go
package auth

import (
    "errors"
    "time"

    "github.com/golang-jwt/jwt/v5"
)

type Claims struct {
    UserID   uint   `json:"user_id"`
    Username string `json:"username"`
    Role     string `json:"role"`
    jwt.RegisteredClaims
}

func GenerateToken(userID uint, username, role, secret string) (string, error) {
    claims := Claims{
        UserID:   userID,
        Username: username,
        Role:     role,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(secret))
}

func ParseToken(tokenString, secret string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        return []byte(secret), nil
    })

    if err != nil {
        return nil, err
    }

    claims, ok := token.Claims.(*Claims)
    if !ok || !token.Valid {
        return nil, errors.New("invalid token")
    }

    return claims, nil
}
```

**Step 5: 创建认证中间件**

```go
package auth

import (
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
)

func AuthMiddleware(secret string) gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
            c.Abort()
            return
        }

        tokenString := strings.TrimPrefix(authHeader, "Bearer ")
        claims, err := ParseToken(tokenString, secret)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
            c.Abort()
            return
        }

        c.Set("user_id", claims.UserID)
        c.Set("username", claims.Username)
        c.Set("role", claims.Role)
        c.Next()
    }
}
```

**Step 6: 创建认证处理器**

```go
package auth

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "golang.org/x/crypto/bcrypt"
    "github.com/edy/ops-platform/internal/database"
    "github.com/edy/ops-platform/internal/models"
    "github.com/edy/ops-platform/pkg/config"
)

type LoginRequest struct {
    Username string `json:"username" binding:"required"`
    Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
    Token string      `json:"token"`
    User  models.User `json:"user"`
}

func RegisterRoutes(r *gin.Engine, cfg *config.Config) {
    auth := r.Group("/auth")
    {
        auth.POST("/login", Login(cfg))
    }

    protected := r.Group("/api")
    protected.Use(AuthMiddleware(cfg.JWT.Secret))
    {
        protected.GET("/user/me", GetCurrentUser())
    }
}

func Login(cfg *config.Config) gin.HandlerFunc {
    return func(c *gin.Context) {
        var req LoginRequest
        if err := c.ShouldBindJSON(&req); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
            return
        }

        var user models.User
        if err := database.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
            return
        }

        if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
            return
        }

        token, err := GenerateToken(user.ID, user.Username, user.Role, cfg.JWT.Secret)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
            return
        }

        c.JSON(http.StatusOK, LoginResponse{Token: token, User: user})
    }
}

func GetCurrentUser() gin.HandlerFunc {
    return func(c *gin.Context) {
        userID := c.GetUint("user_id")
        var user models.User
        if err := database.DB.First(&user, userID).Error; err != nil {
            c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
            return
        }
        c.JSON(http.StatusOK, user)
    }
}
```

**Step 7: 更新 server.go 集成认证**

```go
package server

import (
    "context"
    "fmt"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/edy/ops-platform/pkg/config"
    "github.com/edy/ops-platform/pkg/logger"
    "github.com/edy/ops-platform/internal/database"
    "github.com/edy/ops-platform/internal/auth"
)

func Run() error {
    cfg := config.Load()
    log := logger.New(cfg.LogLevel)

    // 初始化数据库
    if err := database.Init(cfg, log); err != nil {
        return fmt.Errorf("failed to init database: %w", err)
    }

    r := gin.Default()

    // Health check
    r.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"status": "ok"})
    })

    // 注册认证路由
    auth.RegisterRoutes(r, cfg)

    addr := fmt.Sprintf(":%d", cfg.Port)
    srv := &http.Server{
        Addr:        addr,
        Handler:     r,
        IdleTimeout:  120 * time.Second,
        ReadTimeout:  5 * time.Second,
        WriteTimeout: 5 * time.Second,
    }

    log.Info("Starting server on " + addr)

    go func() {
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatal("Server failed: " + err.Error())
        }
    }()

    return nil
}
```

**Step 8: 测试登录接口**

```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"${ADMIN_USERNAME}","password":"${ADMIN_PASSWORD}"}'
```

**Step 9: Commit**

```bash
git add backend/
git commit -m "feat: 用户认证功能

- JWT Token 认证机制
- 用户登录接口 /auth/login
- 认证中间件
- 获取当前用户接口 /api/user/me"
```

---

## Task 5: 验证基础框架

**Step 1: 启动所有服务**

```bash
cd /path/to/ops-platform
docker-compose -f deploy/docker-compose.yml up -d
```

**Step 2: 等待服务启动**

```bash
sleep 30
```

**Step 3: 检查服务健康状态**

```bash
curl http://localhost/health
curl http://localhost:8080/health
```

**Step 4: 测试登录接口**

```bash
curl -X POST http://localhost/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"${ADMIN_USERNAME}","password":"${ADMIN_PASSWORD}"}'
```

**Step 5: 检查日志**

```bash
docker-compose -f deploy/docker-compose.yml logs backend
```

**Step 6: Commit**

```bash
git add -A
git commit -m "chore: 验证基础框架

- 所有服务正常启动
- 登录接口可用
- API 路由正常"
```

---

## 完成清单

- [ ] Go 后端项目脚手架
  - [ ] Gin 框架集成
  - [ ] Viper 配置管理
  - [ ] zap 结构化日志
  - [ ] Health check 接口
- [ ] React 前端项目脚手架
  - [ ] Vite + TypeScript
  - [ ] Ant Design 集成
  - [ ] API 客户端封装
  - [ ] 基础布局和登录页面
- [ ] Docker Compose 部署环境
  - [ ] MySQL 容器
  - [ ] Redis 容器
  - [ ] Go 后端容器
  - [ ] React 前端容器
  - [ ] Nginx 反向代理
- [ ] 用户认证功能
  - [ ] JWT Token 认证
  - [ ] 用户登录接口
  - [ ] 认证中间件
- [ ] MySQL 数据库初始化
  - [ ] 用户表创建
  - [ ] 默认管理员用户

---

## 预计时间：2 周

每个任务预计 2-4 天，留出缓冲时间处理意外问题。
