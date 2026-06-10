// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package platformevent

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jenvenson/ops-platform/internal/auth"
	"github.com/jenvenson/ops-platform/internal/database"
	"github.com/jenvenson/ops-platform/internal/models"
	"github.com/jenvenson/ops-platform/pkg/config"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

type EventListFilter struct {
	Query         string
	EventCategory string
	SourceSystem  string
	EventType     string
	ObjectType    string
	ObjectID      string
	Status        string
	Severity      string
	Operator      string
	TriggerMode   string
	OccurredFrom  *time.Time
	OccurredTo    *time.Time
	Page          int
	Limit         int
}

type EventListItem struct {
	ID            uint           `json:"id"`
	EventID       string         `json:"event_id"`
	EventType     string         `json:"event_type"`
	EventCategory string         `json:"event_category"`
	SourceSystem  string         `json:"source_system"`
	SourceTable   string         `json:"source_table"`
	SourceID      string         `json:"source_id"`
	ObjectType    string         `json:"object_type"`
	ObjectID      string         `json:"object_id"`
	Title         string         `json:"title"`
	Summary       string         `json:"summary"`
	Status        string         `json:"status"`
	Severity      string         `json:"severity"`
	OperatorID    string         `json:"operator_id"`
	OperatorName  string         `json:"operator_name"`
	TriggerMode   string         `json:"trigger_mode"`
	StartedAt     *time.Time     `json:"started_at"`
	FinishedAt    *time.Time     `json:"finished_at"`
	OccurredAt    time.Time      `json:"occurred_at"`
	Metadata      map[string]any `json:"metadata"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

func RegisterRoutes(r *gin.Engine, cfg *config.Config) {
	api := r.Group("/api/platform")
	api.Use(auth.AuthMiddleware(cfg.JWT.Secret))
	{
		api.GET("/events", GetEvents)
		api.GET("/timeline", GetTimeline)
	}
}

func GetEvents(c *gin.Context) {
	filter, err := buildEventListFilter(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	items, total, err := ListEvents(database.DB, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch platform events"})
		return
	}

	resp := make([]EventListItem, 0, len(items))
	for _, item := range items {
		resp = append(resp, toEventListItem(item))
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  resp,
		"page":  filter.Page,
		"limit": filter.Limit,
		"total": total,
	})
}

func GetTimeline(c *gin.Context) {
	objectType := strings.TrimSpace(c.Query("object_type"))
	objectID := strings.TrimSpace(c.Query("object_id"))
	if objectType == "" || objectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "object_type and object_id are required"})
		return
	}

	limit := clampPageSize(parsePositiveInt(c.DefaultQuery("limit", "20"), 20))
	items, err := ListObjectTimeline(database.DB, objectType, objectID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch object timeline"})
		return
	}

	resp := make([]EventListItem, 0, len(items))
	for _, item := range items {
		resp = append(resp, toEventListItem(item))
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        resp,
		"object_type": objectType,
		"object_id":   objectID,
		"limit":       limit,
	})
}

func ListEvents(db *gorm.DB, filter EventListFilter) ([]models.PlatformEvent, int64, error) {
	query := db.Model(&models.PlatformEvent{})

	if q := strings.TrimSpace(filter.Query); q != "" {
		like := "%" + q + "%"
		query = query.Where(
			"title LIKE ? OR summary LIKE ? OR operator_name LIKE ? OR object_id LIKE ? OR source_id LIKE ?",
			like, like, like, like, like,
		)
	}
	if filter.EventCategory != "" {
		query = query.Where("event_category = ?", filter.EventCategory)
	}
	if filter.SourceSystem != "" {
		query = query.Where("source_system = ?", filter.SourceSystem)
	}
	if filter.EventType != "" {
		query = query.Where("event_type = ?", filter.EventType)
	}
	if filter.ObjectType != "" {
		query = query.Where("object_type = ?", filter.ObjectType)
	}
	if filter.ObjectID != "" {
		query = query.Where("object_id = ?", filter.ObjectID)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Severity != "" {
		query = query.Where("severity = ?", filter.Severity)
	}
	if filter.Operator != "" {
		query = query.Where("operator_id = ? OR operator_name = ?", filter.Operator, filter.Operator)
	}
	if filter.TriggerMode != "" {
		query = query.Where("trigger_mode = ?", filter.TriggerMode)
	}
	if filter.OccurredFrom != nil {
		query = query.Where("occurred_at >= ?", *filter.OccurredFrom)
	}
	if filter.OccurredTo != nil {
		query = query.Where("occurred_at <= ?", *filter.OccurredTo)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (filter.Page - 1) * filter.Limit
	var items []models.PlatformEvent
	if err := query.Order("occurred_at DESC").Order("id DESC").Offset(offset).Limit(filter.Limit).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func ListObjectTimeline(db *gorm.DB, objectType, objectID string, limit int) ([]models.PlatformEvent, error) {
	var items []models.PlatformEvent
	err := db.Model(&models.PlatformEvent{}).
		Where("object_type = ? AND object_id = ?", strings.TrimSpace(objectType), strings.TrimSpace(objectID)).
		Order("occurred_at DESC").
		Order("id DESC").
		Limit(clampPageSize(limit)).
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func buildEventListFilter(c *gin.Context) (EventListFilter, error) {
	page := parsePositiveInt(c.DefaultQuery("page", "1"), 1)
	limit := clampPageSize(parsePositiveInt(c.DefaultQuery("limit", strconv.Itoa(defaultPageSize)), defaultPageSize))

	occurredFrom, err := parseDateTime(c.Query("occurred_from"), false)
	if err != nil {
		return EventListFilter{}, err
	}
	occurredTo, err := parseDateTime(c.Query("occurred_to"), true)
	if err != nil {
		return EventListFilter{}, err
	}

	return EventListFilter{
		Query:         strings.TrimSpace(c.Query("q")),
		EventCategory: strings.TrimSpace(c.Query("event_category")),
		SourceSystem:  strings.TrimSpace(c.Query("source_system")),
		EventType:     strings.TrimSpace(c.Query("event_type")),
		ObjectType:    strings.TrimSpace(c.Query("object_type")),
		ObjectID:      strings.TrimSpace(c.Query("object_id")),
		Status:        strings.TrimSpace(c.Query("status")),
		Severity:      strings.TrimSpace(c.Query("severity")),
		Operator:      strings.TrimSpace(c.Query("operator")),
		TriggerMode:   strings.TrimSpace(c.Query("trigger_mode")),
		OccurredFrom:  occurredFrom,
		OccurredTo:    occurredTo,
		Page:          page,
		Limit:         limit,
	}, nil
}

func parsePositiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func clampPageSize(limit int) int {
	if limit <= 0 {
		return defaultPageSize
	}
	if limit > maxPageSize {
		return maxPageSize
	}
	return limit
}

func parseDateTime(raw string, endOfDay bool) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return &t, nil
	}

	if t, err := time.ParseInLocation("2006-01-02", raw, time.Local); err == nil {
		if endOfDay {
			value := t.Add(24*time.Hour - time.Nanosecond)
			return &value, nil
		}
		return &t, nil
	}

	return nil, errInvalidDateTime
}

var errInvalidDateTime = errors.New("invalid occurred_from or occurred_to, expected RFC3339 or YYYY-MM-DD")

func toEventListItem(item models.PlatformEvent) EventListItem {
	return EventListItem{
		ID:            item.ID,
		EventID:       item.EventID,
		EventType:     item.EventType,
		EventCategory: item.EventCategory,
		SourceSystem:  item.SourceSystem,
		SourceTable:   item.SourceTable,
		SourceID:      item.SourceID,
		ObjectType:    item.ObjectType,
		ObjectID:      item.ObjectID,
		Title:         item.Title,
		Summary:       item.Summary,
		Status:        item.Status,
		Severity:      item.Severity,
		OperatorID:    item.OperatorID,
		OperatorName:  item.OperatorName,
		TriggerMode:   item.TriggerMode,
		StartedAt:     item.StartedAt,
		FinishedAt:    item.FinishedAt,
		OccurredAt:    item.OccurredAt,
		Metadata:      parseMetadata(item.MetadataJSON),
		CreatedAt:     item.CreatedAt,
		UpdatedAt:     item.UpdatedAt,
	}
}

func parseMetadata(raw string) map[string]any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(raw), &data); err != nil || data == nil {
		return map[string]any{}
	}
	return data
}