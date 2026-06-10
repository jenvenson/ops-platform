// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package assistant

import "time"

type SessionQuery struct {
	Query  string `form:"query"`
	Status string `form:"status"`
	Page   int    `form:"page"`
	Limit  int    `form:"limit"`
}

type MessageQuery struct {
	Page  int `form:"page"`
	Limit int `form:"limit"`
}

type SessionRequest struct {
	Scene     string `json:"scene"`
	UserAgent string `json:"userAgent"`
	IPAddress string `json:"ipAddress"`
	ForceNew  bool   `json:"forceNew"`
}

type SessionUpdateRequest struct {
	Title  string `json:"title"`
	Status string `json:"status"`
	Pinned *bool  `json:"pinned"`
}

type SessionCleanupRequest struct {
	Status        string `json:"status"`
	OlderThanDays int    `json:"olderThanDays"`
	IncludePinned bool   `json:"includePinned"`
}

type SessionResponse struct {
	Session SessionItem `json:"session"`
}

type SessionItem struct {
	SessionID    string    `json:"sessionId"`
	Scene        string    `json:"scene"`
	Status       string    `json:"status"`
	Title        string    `json:"title,omitempty"`
	Pinned       bool      `json:"pinned"`
	Summary      string    `json:"summary,omitempty"`
	MessageCount int64     `json:"messageCount"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type SessionListResponse struct {
	Sessions []SessionItem `json:"sessions"`
	Page     int           `json:"page"`
	Limit    int           `json:"limit"`
	Total    int64         `json:"total"`
	HasMore  bool          `json:"hasMore"`
}

type MessageRequest struct {
	SessionID   string                `json:"sessionId"`
	Message     string                `json:"message"`
	PageContext *AssistantPageContext `json:"pageContext,omitempty"`
}

type AssistantIntent struct {
	Name       string  `json:"name"`
	SubIntent  string  `json:"subIntent,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
	NeedTools  bool    `json:"needTools"`
}

type AssistantPageContext struct {
	PagePath          string            `json:"pagePath,omitempty"`
	ModuleKey         string            `json:"moduleKey,omitempty"`
	ObjectType        string            `json:"objectType,omitempty"`
	ObjectID          string            `json:"objectId,omitempty"`
	SelectedRecordIDs []string          `json:"selectedRecordIds,omitempty"`
	PageTitle         string            `json:"pageTitle,omitempty"`
	Filters           map[string]string `json:"filters,omitempty"`
}

type AssistantContext struct {
	Message      string                `json:"message,omitempty"`
	HistoryCount int                   `json:"historyCount,omitempty"`
	PageContext  *AssistantPageContext `json:"pageContext,omitempty"`
}

type Citation struct {
	Title   string `json:"title"`
	Path    string `json:"path"`
	Snippet string `json:"snippet,omitempty"`
}

type Action struct {
	Type  string `json:"type"`
	Label string `json:"label"`
	Path  string `json:"path,omitempty"`
}

type ResultCard struct {
	Title      string `json:"title"`
	Subtitle   string `json:"subtitle,omitempty"`
	Meta       string `json:"meta,omitempty"`
	ToolName   string `json:"toolName,omitempty"`
	SourceType string `json:"sourceType,omitempty"`
}

type AssistantExecutionPlanStep struct {
	Tool     string `json:"tool"`
	Purpose  string `json:"purpose,omitempty"`
	Readonly bool   `json:"readonly,omitempty"`
}

type AssistantExecutionPlan struct {
	PlanID string                       `json:"planId"`
	Steps  []AssistantExecutionPlanStep `json:"steps,omitempty"`
}

type AssistantDecision struct {
	Intent           AssistantIntent         `json:"intent"`
	Context          AssistantContext        `json:"context,omitempty"`
	Summary          string                  `json:"summary"`
	Citations        []Citation              `json:"citations,omitempty"`
	Actions          []Action                `json:"actions,omitempty"`
	ResultCards      []ResultCard            `json:"resultCards,omitempty"`
	RiskLevel        string                  `json:"riskLevel,omitempty"`
	NeedConfirmation bool                    `json:"needConfirmation"`
	ExecutionPlan    *AssistantExecutionPlan `json:"executionPlan,omitempty"`
}

type MessageResponse struct {
	MessageID   string             `json:"messageId"`
	Intent      string             `json:"intent"`
	Answer      string             `json:"answer"`
	Citations   []Citation         `json:"citations,omitempty"`
	Actions     []Action           `json:"actions,omitempty"`
	ResultCards []ResultCard       `json:"resultCards,omitempty"`
	Decision    *AssistantDecision `json:"decision,omitempty"`
	Model       string             `json:"model,omitempty"`
	Error       string             `json:"error,omitempty"`
}

type MessageHistoryItem struct {
	MessageID   string       `json:"messageId"`
	Role        string       `json:"role"`
	Intent      string       `json:"intent,omitempty"`
	Text        string       `json:"text"`
	Model       string       `json:"model,omitempty"`
	ResultCards []ResultCard `json:"resultCards,omitempty"`
	Citations   []Citation   `json:"citations,omitempty"`
	Actions     []Action     `json:"actions,omitempty"`
	CreatedAt   time.Time    `json:"createdAt"`
}

type MessageHistoryResponse struct {
	SessionID string               `json:"sessionId"`
	Messages  []MessageHistoryItem `json:"messages"`
	Page      int                  `json:"page"`
	Limit     int                  `json:"limit"`
	Total     int64                `json:"total"`
	HasMore   bool                 `json:"hasMore"`
}

type historyMessage struct {
	Role    string
	Content string
}