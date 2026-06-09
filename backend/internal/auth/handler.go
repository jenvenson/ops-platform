package auth

import (
	"net/http"
	"strconv"
	"time"

	"github.com/jenvenson/ops-platform/internal/audit"
	"github.com/jenvenson/ops-platform/internal/database"
	"github.com/jenvenson/ops-platform/internal/models"
	"github.com/jenvenson/ops-platform/pkg/config"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"crypto/rand"
	"encoding/hex"
)

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string       `json:"token"`
	User  models.User  `json:"user"`
	Menus []MenuConfig `json:"menus"`
}

type MenuConfig struct {
	Key      string       `json:"key"`
	Path     string       `json:"path"`
	Title    string       `json:"title"`
	Icon     string       `json:"icon"`
	Children []MenuConfig `json:"children,omitempty"`
}

func RegisterRoutes(r *gin.Engine, cfg *config.Config) {
	ensureDefaultAuditLogSetting()
	ensureDefaultAssistantModelSetting()
	ensureDefaultSystemGeneralSetting()

	auth := r.Group("/api/auth")
	auth.Use(RateLimitAuth())
	{
		auth.POST("/login", Login(cfg))
		auth.POST("/forgot-password", ForgotPassword())
		auth.POST("/reset-password", ResetPassword())
	}

	protected := r.Group("/api")
	protected.Use(AuthMiddleware(cfg.JWT.Secret))
	{
		protected.GET("/user/me", GetCurrentUser())
		protected.GET("/user/menus", GetCurrentUserMenus())
		protected.PUT("/user/password", ChangePassword())
		protected.PUT("/user/profile", UpdateProfile())
	}

	// Admin routes - require admin or ops role
	admin := r.Group("/api/admin")
	admin.Use(AuthMiddleware(cfg.JWT.Secret), RequireRole("admin", "ops"))
	{
		// 用户管理
		admin.GET("/users", GetUsers())
		admin.POST("/users", CreateUser())
		admin.PUT("/users/:id", UpdateUser())
		admin.DELETE("/users/:id", DeleteUser())

		// 角色管理
		admin.GET("/roles", GetRoles())
		admin.POST("/roles", CreateRole())
		admin.PUT("/roles/:id", UpdateRole())
		admin.DELETE("/roles/:id", DeleteRole())

		// 菜单管理
		admin.GET("/menus", GetMenus())
		admin.POST("/menus", CreateMenu())
		admin.PUT("/menus/:id", UpdateMenu())
		admin.DELETE("/menus/:id", DeleteMenu())

		// 角色菜单关联
		admin.GET("/roles/:id/menus", GetRoleMenus())
		admin.PUT("/roles/:id/menus", UpdateRoleMenus())

		// 系统设置
		admin.GET("/settings/audit-logs", GetAuditLogSetting())
		admin.PUT("/settings/audit-logs", UpdateAuditLogSetting())
		admin.GET("/settings/fim-ssh", GetFIMSSHSetting())
		admin.PUT("/settings/fim-ssh", UpdateFIMSSHSetting())
		admin.POST("/settings/fim-ssh/test", TestFIMSSHConnection())
			admin.GET("/settings/assistant-model", GetAssistantModelSetting())
			admin.PUT("/settings/assistant-model", UpdateAssistantModelSetting())
		admin.GET("/settings/general", GetSystemGeneralSetting())
		admin.PUT("/settings/general", UpdateSystemGeneralSetting())

		admin.GET("/settings/license", GetLicenseStatus(cfg))

		// 平台审计
		admin.GET("/audit/access-logs", GetPlatformAccessLogs())
		admin.DELETE("/audit/access-logs/:id", DeletePlatformAccessLog())
		admin.GET("/audit/operation-logs", GetPlatformAuditLogs())
		admin.DELETE("/audit/operation-logs/:id", DeletePlatformAuditLog())
		admin.GET("/audit/login-logs", GetPlatformLoginLogs())
		admin.DELETE("/audit/login-logs/:id", DeletePlatformLoginLog())
		admin.GET("/audit/archive-stats", GetPlatformArchiveStats())
		admin.GET("/audit/archive-logs", GetPlatformArchivedLogs())
		admin.GET("/audit/export", ExportPlatformAuditLogs())
		admin.DELETE("/audit/archive-logs/:type/:id", DeletePlatformArchivedLog())
		admin.POST("/audit/archive", ArchivePlatformAuditLogs())
		admin.POST("/audit/cleanup-online", CleanupPlatformOnlineAuditLogs())
		admin.POST("/audit/cleanup", CleanupPlatformAuditLogs())
	}
}

// Login godoc
// @Summary      用户登录
// @Description  使用用户名密码登录，返回JWT令牌和菜单权限。首次登录(must_change_password=true)需强制修改密码。
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body LoginRequest true "登录请求"
// @Success      200  {object}  LoginResponse
// @Failure      401  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Router       /auth/login [post]
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

		token, err := GenerateToken(user.ID, user.Username, user.RealName, user.Role, cfg.JWT.Secret)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
			return
		}

		// 根据角色返回菜单
		menus, err := GetMenusByUser(user.ID, user.Role)
		if err != nil {
			// 如果获取菜单失败，使用默认菜单
			menus = getDefaultMenus()
		}

		c.JSON(http.StatusOK, LoginResponse{
			Token: token,
			User:  user,
			Menus: menus,
		})
	}
}

// GetCurrentUser godoc
// @Summary      获取当前用户信息
// @Description  根据JWT令牌返回当前登录用户的详细信息
// @Tags         user
// @Produce      json
// @Security     ApiKeyAuth
// @Success      200  {object}  models.User
// @Failure      404  {object}  map[string]string
// @Router       /user/me [get]
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

func GetCurrentUserMenus() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 明确设置响应的字符编码
		c.Header("Content-Type", "application/json; charset=utf-8")
		ensureDefaultMenusInDB()

		userID := c.GetUint("user_id")
		role := c.GetString("role")
		menus, err := GetMenusByUser(userID, role)
		if err != nil {
			menus = getDefaultMenus()
		}

		// 显式设置字符集并返回JSON
		c.JSON(http.StatusOK, gin.H{"menus": menus})
	}
}

// ChangePassword 修改当前用户密码（首次登录强制修改时无需旧密码）
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// ChangePassword godoc
// @Summary      修改密码
// @Description  修改当前用户密码。首次登录强制修改时无需提供旧密码
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        request body ChangePasswordRequest true "修改密码请求"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Router       /user/password [put]
func ChangePassword() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint("user_id")

		var req ChangePasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效，新密码长度至少6位"})
			return
		}

		var user models.User
		if err := database.DB.First(&user, userID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
			return
		}

		// 首次登录强制修改时跳过旧密码验证
		if !user.MustChangePassword {
			if req.OldPassword == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "请输入旧密码"})
				return
			}
			if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "旧密码不正确"})
				return
			}
		}

		// 加密新密码
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
			return
		}

		user.Password = string(hashedPassword)
		user.MustChangePassword = false
		if err := database.DB.Save(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "密码修改失败"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "密码修改成功"})
	}
}

// UpdateProfile 更新当前用户个人信息
type UpdateProfileRequest struct {
	RealName string `json:"real_name"`
	Email    string `json:"email"`
}

func UpdateProfile() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint("user_id")

		var req UpdateProfileRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效"})
			return
		}

		var user models.User
		if err := database.DB.First(&user, userID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
			return
		}

		if req.RealName != "" {
			user.RealName = req.RealName
		}
		if req.Email != "" {
			user.Email = req.Email
		}

		if err := database.DB.Save(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新个人信息失败"})
			return
		}

		c.JSON(http.StatusOK, user)
	}
}

// User CRUD handlers
// GetUsers godoc
// @Summary      获取所有用户
// @Description  管理员接口，返回所有用户列表
// @Tags         admin
// @Produce      json
// @Security     ApiKeyAuth
// @Success      200  {array}   models.User
// @Failure      500  {object}  map[string]string
// @Router       /admin/users [get]
func GetUsers() gin.HandlerFunc {
	return func(c *gin.Context) {
		var users []models.User
		if err := database.DB.Find(&users).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get users"})
			return
		}
		c.JSON(http.StatusOK, users)
	}
}

type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	RealName string `json:"real_name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

type auditUserRecord struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	RealName string `json:"real_name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

type auditRoleMenuRecord struct {
	RoleID  uint   `json:"role_id"`
	Role    string `json:"role"`
	MenuIDs []uint `json:"menu_ids"`
}

func buildAuditUserRecord(user models.User) auditUserRecord {
	return auditUserRecord{
		ID:       user.ID,
		Username: user.Username,
		RealName: user.RealName,
		Email:    user.Email,
		Role:     user.Role,
	}
}

// CreateUser godoc
// @Summary      创建用户
// @Description  管理员接口，创建新用户。新用户首次登录时需强制修改密码
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        request body CreateUserRequest true "创建用户请求"
// @Success      200  {object}  models.User
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /admin/users [post]
func CreateUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateUserRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		// 检查用户名是否已存在
		var existingUser models.User
		if err := database.DB.Where("username = ?", req.Username).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "username already exists"})
			return
		}

		// 密码加密
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
			return
		}

		// 设置默认角色
		role := req.Role
		if role == "" {
			role = "user"
		}

		user := models.User{
			Username: req.Username,
			Password: string(hashedPassword),
			RealName: req.RealName,
			Email:    req.Email,
			Role:     role,
		}

		if err := database.DB.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
			return
		}
		audit.SetOperationAuditAfter(c, buildAuditUserRecord(user))
		audit.SetOperationAuditSummary(c, "创建了用户。")

		c.JSON(http.StatusOK, user)
	}
}

type UpdateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	RealName string `json:"real_name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

func UpdateUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
			return
		}

		var req UpdateUserRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		var user models.User
		if err := database.DB.First(&user, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		before := buildAuditUserRecord(user)

		// 更新字段
		if req.Username != "" {
			user.Username = req.Username
		}
		if req.RealName != "" {
			user.RealName = req.RealName
		}
		if req.Email != "" {
			user.Email = req.Email
		}
		if req.Role != "" {
			user.Role = req.Role
		}
		if req.Password != "" {
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
				return
			}
			user.Password = string(hashedPassword)
		}

		if err := database.DB.Save(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
			return
		}
		audit.SetOperationAuditBefore(c, before)
		audit.SetOperationAuditAfter(c, buildAuditUserRecord(user))
		audit.SetOperationAuditSummary(c, "更新了用户信息。")

		c.JSON(http.StatusOK, user)
	}
}

func DeleteUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
			return
		}

		// 不能删除自己
		currentUserID := c.GetUint("user_id")
		if uint(id) == currentUserID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete yourself"})
			return
		}
		var user models.User
		if err := database.DB.First(&user, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		if err := database.DB.Delete(&models.User{}, id).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete user"})
			return
		}
		audit.SetOperationAuditBefore(c, buildAuditUserRecord(user))
		audit.SetOperationAuditSummary(c, "删除了用户。")

		c.JSON(http.StatusOK, gin.H{"message": "user deleted"})
	}
}

// ============ 角色管理 API ============

// GetRoles 获取角色列表
func GetRoles() gin.HandlerFunc {
	return func(c *gin.Context) {
		var roles []models.Role
		if err := database.DB.Find(&roles).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get roles"})
			return
		}
		c.JSON(http.StatusOK, roles)
	}
}

// CreateRoleRequest 创建角色请求
type CreateRoleRequest struct {
	Name        string `json:"name" binding:"required"`
	Code        string `json:"code" binding:"required"`
	Description string `json:"description"`
	Status      int    `json:"status"`
}

// CreateRole 创建角色
func CreateRole() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateRoleRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		// 检查角色编码是否已存在
		var existingRole models.Role
		if err := database.DB.Where("code = ?", req.Code).First(&existingRole).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "role code already exists"})
			return
		}

		status := req.Status
		if status != 0 {
			status = 1
		}

		role := models.Role{
			Name:        req.Name,
			Code:        req.Code,
			Description: req.Description,
			Status:      status,
		}

		if err := database.DB.Create(&role).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create role"})
			return
		}
		audit.SetOperationAuditAfter(c, role)
		audit.SetOperationAuditSummary(c, "创建了角色。")

		c.JSON(http.StatusOK, role)
	}
}

// UpdateRoleRequest 更新角色请求
type UpdateRoleRequest struct {
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Status      int    `json:"status"`
}

// UpdateRole 更新角色
func UpdateRole() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role id"})
			return
		}

		var req UpdateRoleRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		var role models.Role
		if err := database.DB.First(&role, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
			return
		}
		before := role

		// 检查新编码是否被其他角色使用
		if req.Code != "" && req.Code != role.Code {
			var existingRole models.Role
			if err := database.DB.Where("code = ? AND id != ?", req.Code, id).First(&existingRole).Error; err == nil {
				c.JSON(http.StatusConflict, gin.H{"error": "role code already exists"})
				return
			}
			role.Code = req.Code
		}

		if req.Name != "" {
			role.Name = req.Name
		}
		if req.Description != "" {
			role.Description = req.Description
		}
		if req.Status == 0 || req.Status == 1 {
			role.Status = req.Status
		}

		if err := database.DB.Save(&role).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update role"})
			return
		}
		audit.SetOperationAuditBefore(c, before)
		audit.SetOperationAuditAfter(c, role)
		audit.SetOperationAuditSummary(c, "更新了角色配置。")

		c.JSON(http.StatusOK, role)
	}
}

// DeleteRole 删除角色
func DeleteRole() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role id"})
			return
		}

		// 不能删除 admin 角色
		if id == 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete admin role"})
			return
		}
		var role models.Role
		if err := database.DB.First(&role, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
			return
		}

		if err := database.DB.Delete(&models.Role{}, id).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete role"})
			return
		}
		audit.SetOperationAuditBefore(c, role)
		audit.SetOperationAuditSummary(c, "删除了角色。")

		c.JSON(http.StatusOK, gin.H{"message": "role deleted"})
	}
}

// ============ 菜单管理 API ============

// GetMenus 获取菜单列表
func GetMenus() gin.HandlerFunc {
	return func(c *gin.Context) {
		ensureDefaultMenusInDB()
		var menus []models.Menu
		if err := database.DB.Order("parent_id ASC, sort ASC").Find(&menus).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get menus"})
			return
		}
		c.JSON(http.StatusOK, menus)
	}
}

// CreateMenuRequest 创建菜单请求
type CreateMenuRequest struct {
	Title    string `json:"title" binding:"required"`
	Key      string `json:"key" binding:"required"`
	Path     string `json:"path"`
	Icon     string `json:"icon"`
	ParentID uint   `json:"parent_id"`
	Sort     int    `json:"sort"`
	Status   int    `json:"status"`
}

// CreateMenu 创建菜单
func CreateMenu() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateMenuRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		// 检查 key 是否已存在
		var existingMenu models.Menu
		if err := database.DB.Where("`key` = ?", req.Key).First(&existingMenu).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "menu key already exists"})
			return
		}

		status := req.Status
		if status != 0 {
			status = 1
		}

		menu := models.Menu{
			Title:    req.Title,
			Key:      req.Key,
			Path:     req.Path,
			Icon:     req.Icon,
			ParentID: req.ParentID,
			Sort:     req.Sort,
			Status:   status,
		}

		if err := database.DB.Create(&menu).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create menu"})
			return
		}
		audit.SetOperationAuditAfter(c, menu)
		audit.SetOperationAuditSummary(c, "创建了菜单配置。")

		c.JSON(http.StatusOK, menu)
	}
}

// UpdateMenuRequest 更新菜单请求
type UpdateMenuRequest struct {
	Title    *string `json:"title"`
	Key      *string `json:"key"`
	Path     *string `json:"path"`
	Icon     *string `json:"icon"`
	ParentID *uint   `json:"parent_id"`
	Sort     *int    `json:"sort"`
	Status   *int    `json:"status"`
}

// UpdateMenu 更新菜单
func UpdateMenu() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid menu id"})
			return
		}

		var req UpdateMenuRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		var menu models.Menu
		if err := database.DB.First(&menu, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "menu not found"})
			return
		}
		before := menu

		// 检查新 key 是否被其他菜单使用
		if req.Key != nil && *req.Key != menu.Key {
			var existingMenu models.Menu
			if err := database.DB.Where("`key` = ? AND id != ?", *req.Key, id).First(&existingMenu).Error; err == nil {
				c.JSON(http.StatusConflict, gin.H{"error": "menu key already exists"})
				return
			}
			menu.Key = *req.Key
		}

		if req.Title != nil {
			menu.Title = *req.Title
		}
		if req.Path != nil {
			menu.Path = *req.Path
		}
		if req.Icon != nil {
			menu.Icon = *req.Icon
		}
		if req.ParentID != nil {
			menu.ParentID = *req.ParentID
		}
		if req.Sort != nil {
			menu.Sort = *req.Sort
		}
		if req.Status != nil {
			menu.Status = *req.Status
		}

		if err := database.DB.Save(&menu).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update menu"})
			return
		}
		audit.SetOperationAuditBefore(c, before)
		audit.SetOperationAuditAfter(c, menu)
		audit.SetOperationAuditSummary(c, "更新了菜单配置。")

		c.JSON(http.StatusOK, menu)
	}
}

// DeleteMenu 删除菜单
func DeleteMenu() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid menu id"})
			return
		}

		var menu models.Menu
		if err := database.DB.First(&menu, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "menu not found"})
			return
		}

		// 删除子菜单
		if err := database.DB.Where("parent_id = ?", id).Delete(&models.Menu{}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete child menus"})
			return
		}

		// 删除角色菜单关联
		if err := database.DB.Where("menu_id = ?", id).Delete(&models.RoleMenu{}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete role menu relations"})
			return
		}

		if err := database.DB.Delete(&models.Menu{}, id).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete menu"})
			return
		}
		audit.SetOperationAuditBefore(c, menu)
		audit.SetOperationAuditSummary(c, "删除了菜单配置。")

		c.JSON(http.StatusOK, gin.H{"message": "menu deleted"})
	}
}

// GetRoleMenus 获取角色的菜单权限
func GetRoleMenus() gin.HandlerFunc {
	return func(c *gin.Context) {
		roleID, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role id"})
			return
		}

		// 获取角色已分配的菜单ID
		var roleMenus []models.RoleMenu
		if err := database.DB.Where("role_id = ?", roleID).Find(&roleMenus).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get role menus"})
			return
		}

		menuIDs := make([]uint, 0, len(roleMenus))
		for _, rm := range roleMenus {
			menuIDs = append(menuIDs, rm.MenuID)
		}

		// 获取所有菜单
		var allMenus []models.Menu
		if err := database.DB.Order("parent_id ASC, sort ASC").Find(&allMenus).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get menus"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"menu_ids": menuIDs,
			"menus":    allMenus,
		})
	}
}

// UpdateRoleMenus 更新角色的菜单权限
func UpdateRoleMenus() gin.HandlerFunc {
	return func(c *gin.Context) {
		roleID, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role id"})
			return
		}

		var req struct {
			MenuIDs []uint `json:"menu_ids"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}
		var role models.Role
		if err := database.DB.First(&role, roleID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
			return
		}
		var existingRoleMenus []models.RoleMenu
		if err := database.DB.Where("role_id = ?", roleID).Find(&existingRoleMenus).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load existing role menus"})
			return
		}
		beforeMenuIDs := make([]uint, 0, len(existingRoleMenus))
		for _, item := range existingRoleMenus {
			beforeMenuIDs = append(beforeMenuIDs, item.MenuID)
		}

		// 删除旧的关联
		if err := database.DB.Where("role_id = ?", roleID).Delete(&models.RoleMenu{}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete old role menus"})
			return
		}

		// 创建新的关联
		roleMenus := make([]models.RoleMenu, 0, len(req.MenuIDs))
		for _, menuID := range req.MenuIDs {
			roleMenus = append(roleMenus, models.RoleMenu{
				RoleID: uint(roleID),
				MenuID: menuID,
			})
		}
		if err := database.DB.Create(&roleMenus).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create role menus"})
			return
		}
		audit.SetOperationAuditBefore(c, auditRoleMenuRecord{RoleID: uint(roleID), Role: role.Name, MenuIDs: beforeMenuIDs})
		audit.SetOperationAuditAfter(c, auditRoleMenuRecord{RoleID: uint(roleID), Role: role.Name, MenuIDs: req.MenuIDs})
		audit.SetOperationAuditSummary(c, "更新了角色菜单授权。")

		c.JSON(http.StatusOK, gin.H{"message": "role menus updated"})
	}
}

// GetMenusByUser 根据用户ID获取菜单
func GetMenusByUser(userID uint, roleCode string) ([]MenuConfig, error) {
	ensureDefaultMenusInDB()
	// 如果是超级管理员，返回所有菜单
	if roleCode == "admin" {
		return getAllMenusFromDB()
	}

	// 获取用户角色的菜单ID
	var role models.Role
	if err := database.DB.Where("code = ?", roleCode).First(&role).Error; err != nil {
		return nil, err
	}

	var roleMenus []models.RoleMenu
	if err := database.DB.Where("role_id = ?", role.ID).Find(&roleMenus).Error; err != nil {
		return nil, err
	}

	// 如果角色没有配置菜单权限，返回所有菜单
	if len(roleMenus) == 0 {
		return getAllMenusFromDB()
	}

	menuIDs := make([]uint, 0, len(roleMenus))
	for _, rm := range roleMenus {
		menuIDs = append(menuIDs, rm.MenuID)
	}

	var menus []models.Menu
	if err := database.DB.Where("id IN ? AND status = 1", menuIDs).Order("parent_id ASC, sort ASC").Find(&menus).Error; err != nil {
		return nil, err
	}

	// 如果有子菜单没有父菜单（父菜单未被授权），需要包含父菜单
	if len(menus) > 0 {
		var childMenuIDs []uint
		for _, m := range menus {
			if m.ParentID != 0 {
				childMenuIDs = append(childMenuIDs, m.ParentID)
			}
		}
		if len(childMenuIDs) > 0 {
			// 查询不在当前列表中但被作为父菜单的菜单
			var missingParentMenus []models.Menu
			if err := database.DB.Where("id IN ? AND status = 1", childMenuIDs).
				Where("id NOT IN ?", menuIDs).
				Order("parent_id ASC, sort ASC").
				Find(&missingParentMenus).Error; err != nil {
				return nil, err
			}
			menus = append(menus, missingParentMenus...)
		}
	}

	return buildMenuTree(menus), nil
}

// getAllMenusFromDB 从数据库获取所有菜单
func getAllMenusFromDB() ([]MenuConfig, error) {
	var menus []models.Menu
	if err := database.DB.Where("status = 1").Order("parent_id ASC, sort ASC").Find(&menus).Error; err != nil {
		return nil, err
	}
	return buildMenuTree(menus), nil
}

// buildMenuTree 递归构建菜单树
func buildMenuTree(menus []models.Menu) []MenuConfig {
	menuMap := make(map[uint]*models.Menu)
	for i := range menus {
		menuMap[menus[i].ID] = &menus[i]
	}

	var roots []models.Menu
	for i := range menus {
		if menus[i].ParentID == 0 {
			roots = append(roots, menus[i])
		}
	}

	// 按 sort 字段排序根菜单
	for i := 0; i < len(roots)-1; i++ {
		for j := i + 1; j < len(roots); j++ {
			if roots[i].Sort > roots[j].Sort {
				roots[i], roots[j] = roots[j], roots[i]
			}
		}
	}

	var result []MenuConfig
	for i := range roots {
		result = append(result, buildMenuNode(&roots[i], menuMap))
	}
	return result
}

// buildMenuNode 递归构建菜单节点及其子节点
func buildMenuNode(menu *models.Menu, menuMap map[uint]*models.Menu) MenuConfig {
	// 先收集所有子菜单
	var childMenus []*models.Menu
	for _, m := range menuMap {
		if m.ParentID == menu.ID {
			childMenus = append(childMenus, m)
		}
	}

	// 按 sort 字段排序
	for i := 0; i < len(childMenus)-1; i++ {
		for j := i + 1; j < len(childMenus); j++ {
			if childMenus[i].Sort > childMenus[j].Sort {
				childMenus[i], childMenus[j] = childMenus[j], childMenus[i]
			}
		}
	}

	// 构建子菜单节点
	children := make([]MenuConfig, 0, len(childMenus))
	for _, m := range childMenus {
		children = append(children, buildMenuNode(m, menuMap))
	}

	return MenuConfig{
		Key:      menu.Key,
		Path:     menu.Path,
		Title:    menu.Title, // 确保标题正确传递
		Icon:     menu.Icon,
		Children: children,
	}
}

// getDefaultMenus 获取默认菜单（备用，当获取菜单失败时使用）
func getDefaultMenus() []MenuConfig {
	// 返回所有菜单作为默认配置
	return allDefaultMenus
}

func ensureDefaultMenusInDB() {
	if database.DB == nil {
		return
	}
	_ = database.DB.Transaction(func(tx *gorm.DB) error {
		if err := syncDefaultMenus(tx, allDefaultMenus, 0); err != nil {
			return err
		}
		if err := disableLegacyMenus(tx); err != nil {
			return err
		}
		if err := ensurePlatformAuditUnderAdmin(tx); err != nil {
			return err
		}
		return ensureAlarmCenterTitle(tx)
	})
}

func syncDefaultMenus(tx *gorm.DB, menus []MenuConfig, parentID uint) error {
	for index, menu := range menus {
		record := models.Menu{
			Title:    menu.Title,
			Key:      menu.Key,
			Path:     menu.Path,
			Icon:     menu.Icon,
			ParentID: parentID,
			Sort:     index,
			Status:   1,
		}

		var existing models.Menu
		query := tx.Where("`key` = ?", menu.Key).Limit(1).Find(&existing)
		if query.Error != nil {
			return query.Error
		}
		if query.RowsAffected == 0 {
			if err := tx.Create(&record).Error; err != nil {
				return err
			}
			existing = record
		}

		if len(menu.Children) > 0 {
			if err := syncDefaultMenus(tx, menu.Children, existing.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

func disableLegacyMenus(tx *gorm.DB) error {
	legacyKeys := []string{
		"cmdb-idc",
	}
	return tx.Model(&models.Menu{}).
		Where("`key` IN ?", legacyKeys).
		Updates(map[string]any{
			"status": 0,
		}).Error
}

func ensurePlatformAuditUnderAdmin(tx *gorm.DB) error {
	var adminMenu models.Menu
	if err := tx.Where("`key` = ?", "admin").First(&adminMenu).Error; err != nil {
		return err
	}

	var platformAudit models.Menu
	query := tx.Where("`key` = ?", "platform-audit").Limit(1).Find(&platformAudit)
	if query.Error != nil {
		return query.Error
	}
	if query.RowsAffected == 0 {
		record := models.Menu{
			Title:    "平台审计",
			Key:      "platform-audit",
			Path:     "/platform/audit",
			Icon:     "FileTextOutlined",
			ParentID: adminMenu.ID,
			Sort:     3,
			Status:   1,
		}
		return tx.Create(&record).Error
	}

	if platformAudit.ParentID == adminMenu.ID {
		return nil
	}

	if err := tx.Model(&models.Menu{}).
		Where("parent_id = ? AND sort >= ?", adminMenu.ID, 3).
		Update("sort", gorm.Expr("sort + 1")).Error; err != nil {
		return err
	}

	return tx.Model(&platformAudit).Updates(map[string]any{
		"parent_id": adminMenu.ID,
		"sort":      3,
		"path":      "/platform/audit",
		"icon":      "FileTextOutlined",
		"title":     "平台审计",
	}).Error
}

func ensureAlarmCenterTitle(tx *gorm.DB) error {
	return tx.Model(&models.Menu{}).
		Where("`key` = ?", "alarm").
		Updates(map[string]any{
			"title": "告警中心",
			"path":  "/alarm",
			"icon":  "BellOutlined",
		}).Error
}

// allDefaultMenus 所有默认菜单配置
var allDefaultMenus = []MenuConfig{
	{
		Key:   "dashboard",
		Path:  "/",
		Title: "工作台",
		Icon:  "DashboardOutlined",
	},
	{
		Key:   "cmdb",
		Path:  "/cmdb",
		Title: "资产中心",
		Icon:  "DatabaseOutlined",
		Children: []MenuConfig{
			{
				Key:   "cmdb-projects",
				Path:  "/cmdb/projects",
				Title: "项目管理",
				Icon:  "ProjectOutlined",
			},
			{
				Key:   "cmdb-environments",
				Path:  "/cmdb/environments",
				Title: "环境管理",
				Icon:  "CloudOutlined",
			},
			{
				Key:   "cmdb-servers",
				Path:  "/cmdb/servers",
				Title: "主机管理",
				Icon:  "DesktopOutlined",
			},
			{
				Key:   "cmdb-applications",
				Path:  "/cmdb/applications",
				Title: "应用流水线管理",
				Icon:  "AppstoreOutlined",
			},
		},
	},
	{
		Key:   "deploy",
		Path:  "/deploy",
		Title: "变更发布",
		Icon:  "RocketOutlined",
		Children: []MenuConfig{
			{
				Key:   "deploy-release",
				Path:  "/deploy/release",
				Title: "迭代部署",
				Icon:  "RocketOutlined",
			},
			{
				Key:   "deploy-history",
				Path:  "/deploy/history",
				Title: "部署记录",
				Icon:  "HistoryOutlined",
			},
			{
				Key:   "deploy-archive",
				Path:  "/deploy/archive",
				Title: "归档打包",
				Icon:  "InboxOutlined",
			},
			{
				Key:   "deploy-archived",
				Path:  "/deploy/archived",
				Title: "归档历史",
				Icon:  "HistoryOutlined",
			},
			{
				Key:   "deploy-aggregate-package",
				Path:  "/deploy/aggregate-package",
				Title: "聚合打包",
				Icon:  "InboxOutlined",
			},
			{
				Key:   "aggregated-history",
				Path:  "/deploy/aggregated-history",
				Title: "聚合历史",
				Icon:  "HistoryOutlined",
			},
			{
				Key:   "consul",
				Path:  "",
				Title: "Consul配置变更",
				Icon:  "ApiOutlined",
				Children: []MenuConfig{
					{
						Key:   "consul-config",
						Path:  "/consul/config",
						Title: "配置管理",
						Icon:  "ApiOutlined",
					},
					{
						Key:   "consul-batch-all",
						Path:  "/consul/batch-all",
						Title: "批量配置下发",
						Icon:  "CopyOutlined",
					},
					{
						Key:   "consul-operations",
						Path:  "/consul/operations",
						Title: "配置操作记录",
						Icon:  "ApiOutlined",
					},
				},
			},
			{
				Key:   "jenkins",
				Path:  "",
				Title: "Jenkins任务",
				Icon:  "AppstoreOutlined",
				Children: []MenuConfig{
					{
						Key:   "jenkins-views",
						Path:  "/jenkins/views",
						Title: "视图管理",
						Icon:  "AppstoreOutlined",
					},
				},
			},
		},
	},
	{
		Key:   "monitor",
		Path:  "/monitor",
		Title: "监控中心",
		Icon:  "MonitorOutlined",
		Children: []MenuConfig{
			{
				Key:   "monitor-bigscreen",
				Path:  "/monitor/bigscreen",
				Title: "监控大屏",
				Icon:  "FundProjectionScreenOutlined",
			},
			{
				Key:   "monitor-overview",
				Path:  "/monitor/overview",
				Title: "监控概览",
				Icon:  "LineChartOutlined",
			},
			{
				Key:   "monitor-dashboards",
				Path:  "/monitor/dashboards",
				Title: "Grafana仪表盘",
				Icon:  "DashboardOutlined",
			},
		},
	},
	{
		Key:   "platform-events",
		Path:  "/platform/events",
		Title: "平台事件中心",
		Icon:  "BellOutlined",
	},
	{
		Key:   "alarm",
		Path:  "/alarm",
		Title: "告警中心",
		Icon:  "BellOutlined",
		Children: []MenuConfig{
			{
				Key:   "alarm-events",
				Path:  "/alarm/events",
				Title: "告警事件",
				Icon:  "AlertOutlined",
			},
			{
				Key:   "alarm-rules",
				Path:  "/alarm/rules",
				Title: "告警规则",
				Icon:  "BellOutlined",
			},
			{
				Key:   "alarm-contacts",
				Path:  "/alarm/contacts",
				Title: "联系人管理",
				Icon:  "TeamOutlined",
			},
			{
				Key:   "alarm-channels",
				Path:  "/alarm/channels",
				Title: "通知渠道",
				Icon:  "SendOutlined",
			},
			{
				Key:   "alarm-templates",
				Path:  "/alarm/templates",
				Title: "通知模板",
				Icon:  "AlertOutlined",
			},
		},
	},
	{
		Key:   "security",
		Path:  "/security",
		Title: "安全中心",
		Icon:  "SafetyOutlined",
		Children: []MenuConfig{
			{
				Key:   "security-overview",
				Path:  "/security/overview",
				Title: "安全概览",
				Icon:  "SafetyOutlined",
			},
			{
				Key:   "security-fim",
				Path:  "/security/fim/policies",
				Title: "文件完整性巡检",
				Icon:  "SafetyOutlined",
				Children: []MenuConfig{
					{
						Key:   "security-fim-policies",
						Path:  "/security/fim/policies",
						Title: "巡检策略",
						Icon:  "SafetyOutlined",
					},
					{
						Key:   "security-fim-executions",
						Path:  "/security/fim/executions",
						Title: "执行记录",
						Icon:  "SyncOutlined",
					},
					{
						Key:   "security-fim-events",
						Path:  "/security/fim/events",
						Title: "文件变更事件",
						Icon:  "FileTextOutlined",
					},
					{
						Key:   "security-fim-alerts",
						Path:  "/security/fim/alerts",
						Title: "完整性告警",
						Icon:  "AlertOutlined",
					},
					{
						Key:   "security-fim-known-hosts",
						Path:  "/security/fim/known-hosts",
						Title: "SSH主机密钥",
						Icon:  "KeyOutlined",
					},
				},
			},
			{
				Key:   "security-tasks",
				Path:  "/security/tasks",
				Title: "扫描任务",
				Icon:  "ScanOutlined",
			},
			{
				Key:   "security-assets",
				Path:  "/security/assets",
				Title: "安全资产",
				Icon:  "DatabaseOutlined",
			},
			{
				Key:   "security-vulnerabilities",
				Path:  "/security/vulnerabilities",
				Title: "漏洞管理",
				Icon:  "BugOutlined",
			},
			{
				Key:   "security-tickets",
				Path:  "/security/tickets",
				Title: "漏洞工单",
				Icon:  "FileTextOutlined",
			},
			{
				Key:   "security-vuln-db",
				Path:  "/security/vuln-db",
				Title: "漏洞知识库",
				Icon:  "FundProjectionScreenOutlined",
			},
		},
	},
	{
		Key:   "admin",
		Path:  "/admin",
		Title: "系统管理",
		Icon:  "SettingOutlined",
		Children: []MenuConfig{
			{
				Key:   "admin-users",
				Path:  "/admin/users",
				Title: "用户管理",
				Icon:  "TeamOutlined",
			},
			{
				Key:   "admin-roles",
				Path:  "/admin/roles",
				Title: "角色管理",
				Icon:  "SafetyOutlined",
			},
			{
				Key:   "admin-menus",
				Path:  "/admin/menus",
				Title: "菜单管理",
				Icon:  "MenuOutlined",
			},
			{
				Key:   "platform-audit",
				Path:  "/platform/audit",
				Title: "平台审计",
				Icon:  "FileTextOutlined",
			},
			{
				Key:   "admin-settings",
				Path:  "/admin/settings",
				Title: "系统设置",
				Icon:  "ToolOutlined",
			},
		},
	},
}
// ForgotPassword generates a reset token for the given username.
// In self-hosted deployments without email, the token is returned directly.
type ForgotPasswordRequest struct {
    Username string `json:"username" binding:"required"`
}

// ForgotPassword godoc
// @Summary      忘记密码
// @Description  输入用户名获取一次性密码重置令牌，令牌有效期1小时
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body ForgotPasswordRequest true "忘记密码请求"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Router       /auth/forgot-password [post]
func ForgotPassword() gin.HandlerFunc {
    return func(c *gin.Context) {
        var req ForgotPasswordRequest
        if err := c.ShouldBindJSON(&req); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "请输入用户名"})
            return
        }

        var user models.User
        if err := database.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
            // 不暴露用户是否存在，统一返回成功
            c.JSON(http.StatusOK, gin.H{"message": "如果用户存在，重置令牌已生成"})
            return
        }

        // 清理该用户旧的未使用令牌
        database.DB.Where("user_id = ? AND used = 0", user.ID).Delete(&models.PasswordResetToken{})

        // 生成新令牌
        tokenBytes := make([]byte, 32)
        if _, err := rand.Read(tokenBytes); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "生成令牌失败"})
            return
        }
        token := hex.EncodeToString(tokenBytes)
        expiresAt := time.Now().Add(1 * time.Hour)

        resetToken := models.PasswordResetToken{
            UserID:    user.ID,
            Token:     token,
            ExpiresAt: expiresAt,
        }
        if err := database.DB.Create(&resetToken).Error; err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "保存令牌失败"})
            return
        }

        c.JSON(http.StatusOK, gin.H{
            "message":    "重置令牌已生成，有效期1小时",
            "token":      token,
            "expires_at": expiresAt.Format(time.RFC3339),
        })
    }
}

// ResetPassword validates a reset token and updates the user's password.
type ResetPasswordRequest struct {
    Token       string `json:"token" binding:"required"`
    NewPassword string `json:"new_password" binding:"required,min=6"`
}

// ResetPassword godoc
// @Summary      重置密码
// @Description  使用忘记密码获取的一次性令牌重置密码
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body ResetPasswordRequest true "重置密码请求"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Router       /auth/reset-password [post]
func ResetPassword() gin.HandlerFunc {
    return func(c *gin.Context) {
        var req ResetPasswordRequest
        if err := c.ShouldBindJSON(&req); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效，新密码长度至少6位"})
            return
        }

        var resetToken models.PasswordResetToken
        if err := database.DB.Where("token = ? AND used = 0 AND expires_at > ?", req.Token, time.Now()).First(&resetToken).Error; err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "令牌无效或已过期"})
            return
        }

        // 更新密码
        hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
            return
        }

        if err := database.DB.Model(&models.User{}).Where("id = ?", resetToken.UserID).Updates(map[string]interface{}{
            "password":            string(hashedPassword),
            "must_change_password": false,
        }).Error; err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "密码更新失败"})
            return
        }

        // 标记令牌已使用
        database.DB.Model(&resetToken).Update("used", 1)

        c.JSON(http.StatusOK, gin.H{"message": "密码重置成功，请使用新密码登录"})
    }
}

