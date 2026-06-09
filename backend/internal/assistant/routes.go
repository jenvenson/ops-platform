package assistant

import (
	"github.com/jenvenson/ops-platform/internal/auth"
	"github.com/jenvenson/ops-platform/pkg/config"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, cfg *config.Config) {
	handler := NewHandler(cfg)

	group := r.Group("/api/assistant")
	group.Use(auth.AuthMiddleware(cfg.JWT.Secret))
	{
		group.GET("/sessions", handler.ListSessions)
		group.POST("/sessions", handler.CreateSession)
		group.POST("/sessions/cleanup", handler.CleanupSessions)
		group.PATCH("/sessions/:sessionId", handler.UpdateSession)
		group.DELETE("/sessions/:sessionId", handler.DeleteSession)
		group.GET("/sessions/:sessionId/messages", handler.GetSessionMessages)
		group.GET("/sessions/:sessionId/export", handler.ExportSession)
		group.POST("/messages", handler.SendMessage)
	}
}
