package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/edy/ops-platform/internal/audit"
	"github.com/edy/ops-platform/internal/auth"
	"github.com/edy/ops-platform/internal/database"
	"github.com/edy/ops-platform/internal/models"
	"github.com/edy/ops-platform/internal/platformevent"
	"github.com/edy/ops-platform/pkg/config"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct {
	service *Service
	limiter *rateLimiter
}

type assistantUser struct {
	ID       string
	Username string
	RealName string
	Role     string
}

func NewHandler(cfg *config.Config) *Handler {
	return &Handler{
		service: NewService(cfg),
		limiter: newRateLimiter(cfg.Assistant.RateLimitPerMinute, time.Minute),
	}
}

func Init() error {
	return database.DB.AutoMigrate(
		&models.AssistantSession{},
		&models.AssistantMessage{},
		&models.AssistantCitation{},
	)
}

func (h *Handler) ListSessions(c *gin.Context) {
	user, ok := readAssistantUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var query SessionQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid query params"})
		return
	}

	page, limit := normalizeSessionPagination(query.Page, query.Limit)
	dbQuery := database.DB.Model(&models.AssistantSession{}).Where("user_id = ?", user.ID)
	if search := strings.TrimSpace(query.Query); search != "" {
		like := "%" + search + "%"
		dbQuery = dbQuery.Where("(title LIKE ? OR summary LIKE ?)", like, like)
	}
	if status := strings.TrimSpace(query.Status); status != "" && status != "all" {
		if status != "active" && status != "archived" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session status"})
			return
		}
		dbQuery = dbQuery.Where("status = ?", status)
	}

	var total int64
	if err := dbQuery.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count assistant sessions"})
		return
	}

	var sessions []models.AssistantSession
	if err := dbQuery.
		Order("pinned desc").
		Order("updated_at desc").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&sessions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load assistant sessions"})
		return
	}

	items := make([]SessionItem, 0, len(sessions))
	for _, session := range sessions {
		items = append(items, buildSessionItem(session))
	}

	c.JSON(http.StatusOK, SessionListResponse{
		Sessions: items,
		Page:     page,
		Limit:    limit,
		Total:    total,
		HasMore:  int64(page*limit) < total,
	})
}

func (h *Handler) CreateSession(c *gin.Context) {
	if err := Init(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to initialize assistant tables"})
		return
	}

	var req SessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if req.Scene == "" {
		req.Scene = "web"
	}
	if req.UserAgent == "" {
		req.UserAgent = c.Request.UserAgent()
	}
	if req.IPAddress == "" {
		req.IPAddress = audit.ExtractRequestIP(c)
	}

	user, ok := readAssistantUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if !req.ForceNew {
		var existing models.AssistantSession
		if err := database.DB.Where("user_id = ? AND scene = ? AND status = ?", user.ID, req.Scene, "active").
			Order("updated_at desc").
			First(&existing).Error; err == nil {
			c.JSON(http.StatusOK, SessionResponse{
				Session: buildSessionItem(existing),
			})
			return
		}
	} else {
		_ = database.DB.Model(&models.AssistantSession{}).
			Where("user_id = ? AND scene = ? AND status = ?", user.ID, req.Scene, "active").
			Updates(map[string]any{
				"status":     "archived",
				"updated_at": time.Now(),
			}).Error
	}

	session := &models.AssistantSession{
		SessionID: generateID("asst"),
		UserID:    user.ID,
		Scene:     req.Scene,
		Status:    "active",
		UserAgent: req.UserAgent,
		IPAddress: req.IPAddress,
	}

	if err := database.DB.Create(session).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create assistant session"})
		return
	}
	_ = platformevent.RecordAssistantSession(*session)

	c.JSON(http.StatusOK, SessionResponse{
		Session: buildSessionItem(*session),
	})
}

func (h *Handler) GetSessionMessages(c *gin.Context) {
	user, ok := readAssistantUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	sessionID := strings.TrimSpace(c.Param("sessionId"))
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sessionId is required"})
		return
	}

	var session models.AssistantSession
	if err := database.DB.Where("session_id = ? AND user_id = ?", sessionID, user.ID).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "assistant session not found"})
		return
	}

	var query MessageQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid query params"})
		return
	}
	page, limit := normalizeSessionPagination(query.Page, query.Limit)

	messageQuery := database.DB.Model(&models.AssistantMessage{}).Where("session_id = ?", sessionID)
	var total int64
	if err := messageQuery.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count assistant messages"})
		return
	}

	var messages []models.AssistantMessage
	if err := messageQuery.
		Order("created_at asc").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load assistant messages"})
		return
	}

	messageIDs := make([]string, 0, len(messages))
	for _, message := range messages {
		messageIDs = append(messageIDs, message.MessageID)
	}

	citationMap := loadCitations(messageIDs)
	items := make([]MessageHistoryItem, 0, len(messages))
	for _, message := range messages {
		actions, resultCards := decodeMessageMetadata(message)
		items = append(items, MessageHistoryItem{
			MessageID:   message.MessageID,
			Role:        message.Role,
			Intent:      message.Intent,
			Text:        message.Content,
			Model:       message.ModelName,
			ResultCards: resultCards,
			Citations:   citationMap[message.MessageID],
			Actions:     actions,
			CreatedAt:   message.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, MessageHistoryResponse{
		SessionID: sessionID,
		Messages:  items,
		Page:      page,
		Limit:     limit,
		Total:     total,
		HasMore:   int64(page*limit) < total,
	})
}

func (h *Handler) UpdateSession(c *gin.Context) {
	user, ok := readAssistantUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	sessionID := strings.TrimSpace(c.Param("sessionId"))
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sessionId is required"})
		return
	}

	var req SessionUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	var session models.AssistantSession
	if err := database.DB.Where("session_id = ? AND user_id = ?", sessionID, user.ID).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "assistant session not found"})
		return
	}

	updates := map[string]any{
		"updated_at": time.Now(),
	}

	if title := strings.TrimSpace(req.Title); title != "" {
		normalized := shorten(title, 80)
		updates["title"] = normalized
		session.Title = normalized
	}

	if status := strings.TrimSpace(req.Status); status != "" {
		if status != "active" && status != "archived" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session status"})
			return
		}
		updates["status"] = status
		session.Status = status
	}
	if req.Pinned != nil {
		updates["pinned"] = *req.Pinned
		session.Pinned = *req.Pinned
	}

	if len(updates) == 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no session fields to update"})
		return
	}

	session.UpdatedAt = updates["updated_at"].(time.Time)
	if err := database.DB.Model(&session).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update assistant session"})
		return
	}
	_ = platformevent.RecordAssistantSession(session)

	c.JSON(http.StatusOK, SessionResponse{
		Session: buildSessionItem(session),
	})
}

func (h *Handler) ExportSession(c *gin.Context) {
	user, ok := readAssistantUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	sessionID := strings.TrimSpace(c.Param("sessionId"))
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sessionId is required"})
		return
	}

	var session models.AssistantSession
	if err := database.DB.Where("session_id = ? AND user_id = ?", sessionID, user.ID).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "assistant session not found"})
		return
	}

	var messages []models.AssistantMessage
	if err := database.DB.Where("session_id = ?", sessionID).Order("created_at asc").Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load assistant messages"})
		return
	}
	messageIDs := make([]string, 0, len(messages))
	for _, message := range messages {
		messageIDs = append(messageIDs, message.MessageID)
	}
	citationMap := loadCitations(messageIDs)

	format := strings.ToLower(strings.TrimSpace(c.DefaultQuery("format", "markdown")))
	filenameBase := session.Title
	if strings.TrimSpace(filenameBase) == "" {
		filenameBase = session.SessionID
	}
	filenameBase = sanitizeFilename(filenameBase)

	switch format {
	case "json":
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.json\"", filenameBase))
		c.JSON(http.StatusOK, gin.H{
			"session":  buildSessionItem(session),
			"messages": buildExportMessages(messages, citationMap),
		})
	case "markdown", "md":
		c.Header("Content-Type", "text/markdown; charset=utf-8")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.md\"", filenameBase))
		c.String(http.StatusOK, buildSessionMarkdown(session, messages, citationMap))
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid export format"})
	}
}

func (h *Handler) CleanupSessions(c *gin.Context) {
	user, ok := readAssistantUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req SessionCleanupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if req.OlderThanDays <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "olderThanDays must be greater than 0"})
		return
	}
	if req.Status != "" && req.Status != "active" && req.Status != "archived" && req.Status != "all" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session status"})
		return
	}

	cutoff := time.Now().AddDate(0, 0, -req.OlderThanDays)
	query := database.DB.Model(&models.AssistantSession{}).
		Where("user_id = ? AND updated_at < ?", user.ID, cutoff)
	if req.Status != "" && req.Status != "all" {
		query = query.Where("status = ?", req.Status)
	}
	if !req.IncludePinned {
		query = query.Where("pinned = ?", false)
	}

	var sessions []models.AssistantSession
	if err := query.Find(&sessions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load assistant sessions"})
		return
	}
	if len(sessions) == 0 {
		c.JSON(http.StatusOK, gin.H{"deleted": 0})
		return
	}

	sessionIDs := make([]string, 0, len(sessions))
	for _, session := range sessions {
		sessionIDs = append(sessionIDs, session.SessionID)
	}
	deleted, err := deleteSessionsByIDs(sessionIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cleanup assistant sessions"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": deleted})
}

func (h *Handler) DeleteSession(c *gin.Context) {
	user, ok := readAssistantUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	sessionID := strings.TrimSpace(c.Param("sessionId"))
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sessionId is required"})
		return
	}

	var session models.AssistantSession
	if err := database.DB.Where("session_id = ? AND user_id = ?", sessionID, user.ID).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "assistant session not found"})
		return
	}

	var messageIDs []string
	if err := database.DB.Model(&models.AssistantMessage{}).
		Where("session_id = ?", sessionID).
		Pluck("message_id", &messageIDs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load assistant messages"})
		return
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		return deleteSessionGraph(tx, []string{sessionID}, messageIDs)
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete assistant session"})
		return
	}
	_ = platformevent.RecordAssistantSessionDeleted(session)

	c.JSON(http.StatusOK, gin.H{"deleted": true, "sessionId": sessionID})
}

func (h *Handler) SendMessage(c *gin.Context) {
	if err := Init(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to initialize assistant tables"})
		return
	}

	var req MessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if strings.TrimSpace(req.SessionID) == "" || strings.TrimSpace(req.Message) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sessionId and message are required"})
		return
	}
	req.Message = strings.TrimSpace(req.Message)
	req.PageContext = sanitizePageContext(req.PageContext)
	if exceedsRunes(req.Message, h.service.cfg.Assistant.MaxMessageRunes) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message too long"})
		return
	}
	user, ok := readAssistantUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if !h.limiter.Allow(user.ID) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "assistant rate limit exceeded"})
		return
	}

	if rtCfg := auth.FetchAssistantRuntimeConfig(); rtCfg != nil {
		h.service.ReloadFromRuntimeConfig(rtCfg.Provider, rtCfg.Enabled, rtCfg.APIKey,
			rtCfg.BaseURL, rtCfg.ChatModel, rtCfg.EmbedModel, rtCfg.Temperature, rtCfg.TimeoutSec)
	}

	var session models.AssistantSession
	if err := database.DB.Where("session_id = ? AND user_id = ?", req.SessionID, user.ID).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "assistant session not found"})
		return
	}

	userMessageID := generateID("msg")
	if err := database.DB.Create(&models.AssistantMessage{
		SessionID: req.SessionID,
		MessageID: userMessageID,
		UserID:    user.ID,
		Role:      "user",
		Content:   req.Message,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save user message"})
		return
	}

	history, _ := loadHistory(req.SessionID, h.service.cfg.Assistant.MaxContextMessages)
	ctx, cancel := context.WithTimeout(c.Request.Context(), assistantTimeout(h.service.cfg))
	defer cancel()
	reply, promptTokens, completionTokens, latency := h.service.GenerateReply(ctx, req.Message, history, req.PageContext)
	reply.MessageID = generateID("msg")

	assistantMessage := models.AssistantMessage{
		SessionID:        req.SessionID,
		MessageID:        reply.MessageID,
		UserID:           user.ID,
		Role:             "assistant",
		Intent:           reply.Intent,
		Content:          reply.Answer,
		ModelName:        reply.Model,
		ActionsJSON:      mustJSON(reply.Actions),
		ResultCardsJSON:  mustJSON(reply.ResultCards),
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		LatencyMS:        latency,
	}
	if err := database.DB.Create(&assistantMessage).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save assistant message"})
		return
	}
	_ = platformevent.RecordAssistantMessage(assistantMessage)

	for _, citation := range reply.Citations {
		_ = database.DB.Create(&models.AssistantCitation{
			MessageID:   reply.MessageID,
			SourceType:  "document",
			SourceTitle: citation.Title,
			SourcePath:  citation.Path,
			Snippet:     citation.Snippet,
		}).Error
	}

	session.Title = updateSessionTitle(session.Title, req.Message)
	session.Summary = updateSessionSummary(session.Summary, req.Message, reply.Answer)
	session.UpdatedAt = time.Now()
	_ = database.DB.Model(&session).Updates(map[string]any{
		"title":      session.Title,
		"summary":    session.Summary,
		"updated_at": session.UpdatedAt,
	}).Error

	c.JSON(http.StatusOK, reply)
}

func loadHistory(sessionID string, limit int) ([]historyMessage, error) {
	if limit <= 0 {
		limit = 12
	}

	var messages []models.AssistantMessage
	if err := database.DB.Where("session_id = ?", sessionID).
		Order("created_at desc").
		Limit(limit).
		Find(&messages).Error; err != nil {
		return nil, err
	}

	history := make([]historyMessage, 0, len(messages))
	for i := len(messages) - 1; i >= 0; i-- {
		history = append(history, historyMessage{
			Role:    messages[i].Role,
			Content: messages[i].Content,
		})
	}
	return history, nil
}

func loadCitations(messageIDs []string) map[string][]Citation {
	results := make(map[string][]Citation)
	if len(messageIDs) == 0 {
		return results
	}

	var citations []models.AssistantCitation
	if err := database.DB.Where("message_id IN ?", messageIDs).Find(&citations).Error; err != nil {
		return results
	}

	for _, citation := range citations {
		results[citation.MessageID] = append(results[citation.MessageID], Citation{
			Title:   citation.SourceTitle,
			Path:    citation.SourcePath,
			Snippet: citation.Snippet,
		})
	}
	return results
}

func readAssistantUser(c *gin.Context) (assistantUser, bool) {
	userIDValue, exists := c.Get("user_id")
	if !exists {
		return assistantUser{}, false
	}

	userID := strings.TrimSpace(fmt.Sprint(userIDValue))
	if userID == "" {
		return assistantUser{}, false
	}

	return assistantUser{
		ID:       userID,
		Username: strings.TrimSpace(c.GetString("username")),
		RealName: strings.TrimSpace(c.GetString("real_name")),
		Role:     strings.TrimSpace(c.GetString("role")),
	}, true
}

func generateID(prefix string) string {
	return prefix + "_" + time.Now().Format("20060102150405.000000000")
}

func buildSessionItem(session models.AssistantSession) SessionItem {
	return SessionItem{
		SessionID:    session.SessionID,
		Scene:        session.Scene,
		Status:       session.Status,
		Title:        session.Title,
		Pinned:       session.Pinned,
		Summary:      session.Summary,
		MessageCount: countSessionMessages(session.SessionID),
		CreatedAt:    session.CreatedAt,
		UpdatedAt:    session.UpdatedAt,
	}
}

func countSessionMessages(sessionID string) int64 {
	if strings.TrimSpace(sessionID) == "" {
		return 0
	}

	var count int64
	if err := database.DB.Model(&models.AssistantMessage{}).Where("session_id = ?", sessionID).Count(&count).Error; err != nil {
		return 0
	}
	return count
}

func buildSessionTitle(message string) string {
	title := shorten(strings.TrimSpace(message), 40)
	if title == "" {
		return "新会话"
	}
	return title
}

func updateSessionTitle(currentTitle, latestUserMessage string) string {
	currentTitle = strings.TrimSpace(currentTitle)
	if currentTitle != "" && currentTitle != "新会话" {
		return currentTitle
	}
	return buildSessionTitle(latestUserMessage)
}

func updateSessionSummary(currentSummary, latestUserMessage, latestReply string) string {
	userText := shorten(strings.TrimSpace(latestUserMessage), 100)
	replyText := shorten(singleLine(latestReply), 120)

	switch {
	case userText != "" && replyText != "":
		return userText + " | " + replyText
	case replyText != "":
		return replyText
	case userText != "":
		return userText
	default:
		return strings.TrimSpace(currentSummary)
	}
}

func normalizeSessionPagination(page, limit int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	return page, limit
}

func singleLine(text string) string {
	parts := strings.Fields(strings.TrimSpace(text))
	return strings.Join(parts, " ")
}

func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "\n", " ", "\r", " ", "\t", " ")
	cleaned := strings.TrimSpace(replacer.Replace(name))
	if cleaned == "" {
		return "assistant-session"
	}
	return cleaned
}

func buildSessionMarkdown(session models.AssistantSession, messages []models.AssistantMessage, citationMap map[string][]Citation) string {
	var builder strings.Builder
	title := strings.TrimSpace(session.Title)
	if title == "" {
		title = session.SessionID
	}
	builder.WriteString("# ")
	builder.WriteString(title)
	builder.WriteString("\n\n")
	builder.WriteString("- 会话ID: ")
	builder.WriteString(session.SessionID)
	builder.WriteString("\n")
	builder.WriteString("- 状态: ")
	builder.WriteString(session.Status)
	builder.WriteString("\n")
	builder.WriteString("- 创建时间: ")
	builder.WriteString(session.CreatedAt.Format(time.RFC3339))
	builder.WriteString("\n\n")

	for _, message := range messages {
		actions, resultCards := decodeMessageMetadata(message)
		role := "助手"
		if message.Role == "user" {
			role = "用户"
		}
		builder.WriteString("## ")
		builder.WriteString(role)
		builder.WriteString(" ")
		builder.WriteString(message.CreatedAt.Format("2006-01-02 15:04:05"))
		builder.WriteString("\n\n")
		builder.WriteString(strings.TrimSpace(message.Content))
		builder.WriteString("\n\n")
		if len(resultCards) > 0 {
			builder.WriteString("### 结果卡片\n\n")
			for _, card := range resultCards {
				builder.WriteString("- ")
				builder.WriteString(strings.TrimSpace(card.Title))
				if strings.TrimSpace(card.Subtitle) != "" {
					builder.WriteString(" | ")
					builder.WriteString(strings.TrimSpace(card.Subtitle))
				}
				builder.WriteString("\n")
			}
			builder.WriteString("\n")
		}
		if len(actions) > 0 {
			builder.WriteString("### 快捷操作\n\n")
			for _, action := range actions {
				builder.WriteString("- ")
				builder.WriteString(strings.TrimSpace(action.Label))
				if strings.TrimSpace(action.Path) != "" {
					builder.WriteString(" -> ")
					builder.WriteString(strings.TrimSpace(action.Path))
				}
				builder.WriteString("\n")
			}
			builder.WriteString("\n")
		}
		if citations := citationMap[message.MessageID]; len(citations) > 0 {
			builder.WriteString("### 引用\n\n")
			for _, citation := range citations {
				builder.WriteString("- ")
				builder.WriteString(strings.TrimSpace(citation.Title))
				if strings.TrimSpace(citation.Path) != "" {
					builder.WriteString(" (")
					builder.WriteString(strings.TrimSpace(citation.Path))
					builder.WriteString(")")
				}
				builder.WriteString("\n")
			}
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

func decodeMessageMetadata(message models.AssistantMessage) ([]Action, []ResultCard) {
	var actions []Action
	var resultCards []ResultCard
	if strings.TrimSpace(message.ActionsJSON) != "" {
		_ = json.Unmarshal([]byte(message.ActionsJSON), &actions)
	}
	if strings.TrimSpace(message.ResultCardsJSON) != "" {
		_ = json.Unmarshal([]byte(message.ResultCardsJSON), &resultCards)
	}
	return actions, resultCards
}

func mustJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil || string(data) == "null" {
		return ""
	}
	return string(data)
}

func buildExportMessages(messages []models.AssistantMessage, citationMap map[string][]Citation) []gin.H {
	items := make([]gin.H, 0, len(messages))
	for _, message := range messages {
		actions, resultCards := decodeMessageMetadata(message)
		items = append(items, gin.H{
			"messageId":   message.MessageID,
			"role":        message.Role,
			"intent":      message.Intent,
			"text":        message.Content,
			"model":       message.ModelName,
			"actions":     actions,
			"resultCards": resultCards,
			"citations":   citationMap[message.MessageID],
			"createdAt":   message.CreatedAt,
		})
	}
	return items
}

func deleteSessionsByIDs(sessionIDs []string) (int, error) {
	if len(sessionIDs) == 0 {
		return 0, nil
	}

	var messageIDs []string
	if err := database.DB.Model(&models.AssistantMessage{}).
		Where("session_id IN ?", sessionIDs).
		Pluck("message_id", &messageIDs).Error; err != nil {
		return 0, err
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		return deleteSessionGraph(tx, sessionIDs, messageIDs)
	}); err != nil {
		return 0, err
	}
	return len(sessionIDs), nil
}

func deleteSessionGraph(tx *gorm.DB, sessionIDs []string, messageIDs []string) error {
	if len(messageIDs) > 0 {
		if err := tx.Where("message_id IN ?", messageIDs).Delete(&models.AssistantCitation{}).Error; err != nil {
			return err
		}
	}
	if err := tx.Where("session_id IN ?", sessionIDs).Delete(&models.AssistantMessage{}).Error; err != nil {
		return err
	}
	if err := tx.Where("session_id IN ?", sessionIDs).Delete(&models.AssistantSession{}).Error; err != nil {
		return err
	}
	return nil
}

func exceedsRunes(text string, max int) bool {
	if max <= 0 {
		max = 1000
	}
	return len([]rune(text)) > max
}

func assistantTimeout(cfg *config.Config) time.Duration {
	if cfg == nil || cfg.Assistant.RequestTimeoutSec <= 0 {
		return 20 * time.Second
	}
	return time.Duration(cfg.Assistant.RequestTimeoutSec) * time.Second
}
