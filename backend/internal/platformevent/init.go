package platformevent

import (
	"fmt"
	"strings"
	"time"

	"github.com/edy/ops-platform/internal/database"
	"github.com/edy/ops-platform/internal/models"
	"gorm.io/gorm"
)

const (
	EventCategoryDeploy    = "deploy"
	EventCategoryArchive   = "archive"
	EventCategoryAlert     = "alert"
	EventCategoryAssistant = "assistant"

	sourceSystemDeploy    = "deploy"
	sourceSystemAlert     = "alert"
	sourceSystemAssistant = "assistant"
)

func Init() error {
	if err := database.DB.AutoMigrate(&models.PlatformEvent{}); err != nil {
		return fmt.Errorf("failed to migrate platform_event_stream: %w", err)
	}
	if err := SyncSeedData(database.DB); err != nil {
		return fmt.Errorf("failed to sync platform event seed data: %w", err)
	}
	return nil
}

func RecordAssistantSession(session models.AssistantSession) error {
	return persistEvent(buildAssistantSessionEvent(session))
}

func RecordDeployRecord(record DeployRecordPayload) error {
	return persistEvent(buildDeployRecordEvent(record))
}

func RecordDeployRecordDeleted(record DeployRecordPayload) error {
	return persistEvent(buildDeployRecordDeletedEvent(record))
}

func RecordArchiveRecord(record ArchiveRecordPayload) error {
	return persistEvent(buildArchiveRecordEvent(record))
}

func RecordArchiveRecordDeleted(record ArchiveRecordPayload) error {
	return persistEvent(buildArchiveRecordDeletedEvent(record))
}

func RecordAlertEvent(event AlertEventPayload) error {
	return persistEvent(buildAlertEvent(event))
}

func RecordAlertEventDeleted(event AlertEventPayload) error {
	return persistEvent(buildAlertDeletedEvent(event))
}

func RecordAssistantSessionDeleted(session models.AssistantSession) error {
	return persistEvent(buildAssistantSessionDeletedEvent(session))
}

func RecordAssistantMessage(message models.AssistantMessage) error {
	return persistEvent(buildAssistantMessageEvent(message))
}

func SyncSeedData(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("nil db")
	}

	var deployRecords []DeployRecordPayload
	if err := db.Order("id desc").Limit(1000).Find(&deployRecords).Error; err != nil {
		return err
	}
	for _, record := range deployRecords {
		if err := upsertEvent(db, buildDeployRecordEvent(record)); err != nil {
			return err
		}
	}

	var alertEvents []AlertEventPayload
	if err := db.Order("id desc").Limit(1000).Find(&alertEvents).Error; err != nil {
		return err
	}
	for _, event := range alertEvents {
		if err := upsertEvent(db, buildAlertEvent(event)); err != nil {
			return err
		}
	}

	var sessions []models.AssistantSession
	if err := db.Order("id desc").Limit(1000).Find(&sessions).Error; err != nil {
		return err
	}
	for _, session := range sessions {
		if err := upsertEvent(db, buildAssistantSessionEvent(session)); err != nil {
			return err
		}
	}

	var messages []models.AssistantMessage
	if err := db.Where("role = ?", "assistant").Order("id desc").Limit(1000).Find(&messages).Error; err != nil {
		return err
	}
	for _, message := range messages {
		if err := upsertEvent(db, buildAssistantMessageEvent(message)); err != nil {
			return err
		}
	}

	return nil
}

func buildDeployRecordEvent(record DeployRecordPayload) models.PlatformEvent {
	eventType := "deploy_started"
	switch strings.ToLower(strings.TrimSpace(record.Status)) {
	case "failed":
		eventType = "deploy_failed"
	case "success":
		eventType = "deployed"
	}

	objectID := objectUID("deploy_record", sourceSystemDeploy, record.ID)
	return models.PlatformEvent{
		EventID:       eventID(eventType, sourceSystemDeploy, record.ID),
		EventType:     eventType,
		EventCategory: EventCategoryDeploy,
		SourceSystem:  sourceSystemDeploy,
		SourceTable:   record.TableName(),
		SourceID:      uintString(record.ID),
		ObjectType:    "deploy_record",
		ObjectID:      objectID,
		Title:         deployEventTitle(record),
		Summary:       deployEventSummary(record),
		Status:        strings.TrimSpace(record.Status),
		Severity:      deploySeverity(record.Status),
		OperatorID:    strings.TrimSpace(record.TriggeredBy),
		OperatorName:  strings.TrimSpace(record.TriggeredBy),
		TriggerMode:   "user",
		StartedAt:     record.StartTime,
		FinishedAt:    record.EndTime,
		OccurredAt:    record.CreatedAt,
		MetadataJSON: mustEventMetadataJSON(map[string]any{
			"appId":       record.AppID,
			"appName":     strings.TrimSpace(record.AppName),
			"envId":       record.EnvID,
			"envName":     strings.TrimSpace(record.EnvName),
			"projectCode": strings.TrimSpace(record.ProjectCode),
			"deployType":  strings.TrimSpace(record.DeployType),
		}),
	}
}

func buildDeployRecordDeletedEvent(record DeployRecordPayload) models.PlatformEvent {
	return models.PlatformEvent{
		EventID:       eventID("deploy_deleted", sourceSystemDeploy, record.ID),
		EventType:     "deploy_deleted",
		EventCategory: EventCategoryDeploy,
		SourceSystem:  sourceSystemDeploy,
		SourceTable:   record.TableName(),
		SourceID:      uintString(record.ID),
		ObjectType:    "deploy_record",
		ObjectID:      objectUID("deploy_record", sourceSystemDeploy, record.ID),
		Title:         deployEventTitle(record),
		Summary:       "部署记录已删除",
		Status:        "closed",
		Severity:      "info",
		OperatorID:    strings.TrimSpace(record.TriggeredBy),
		OperatorName:  strings.TrimSpace(record.TriggeredBy),
		TriggerMode:   "user",
		OccurredAt:    record.UpdatedAt,
	}
}

func buildArchiveRecordEvent(record ArchiveRecordPayload) models.PlatformEvent {
	eventType := "archive_started"
	switch strings.ToLower(strings.TrimSpace(record.Status)) {
	case "failed":
		eventType = "archive_failed"
	case "success":
		eventType = "archived"
	}

	return models.PlatformEvent{
		EventID:       eventID(eventType, sourceSystemDeploy, record.ID),
		EventType:     eventType,
		EventCategory: EventCategoryArchive,
		SourceSystem:  sourceSystemDeploy,
		SourceTable:   record.TableName(),
		SourceID:      uintString(record.ID),
		ObjectType:    "archive_record",
		ObjectID:      objectUID("archive_record", sourceSystemDeploy, record.ID),
		Title:         archiveEventTitle(record),
		Summary:       archiveEventSummary(record),
		Status:        strings.TrimSpace(record.Status),
		Severity:      archiveSeverity(record.Status),
		OperatorID:    strings.TrimSpace(record.Operator),
		OperatorName:  strings.TrimSpace(record.Operator),
		TriggerMode:   "user",
		StartedAt:     record.StartTime,
		FinishedAt:    record.EndTime,
		OccurredAt:    record.CreatedAt,
		MetadataJSON: mustEventMetadataJSON(map[string]any{
			"appId":       record.AppID,
			"appName":     strings.TrimSpace(record.AppName),
			"envId":       record.EnvID,
			"envName":     strings.TrimSpace(record.EnvName),
			"projectCode": strings.TrimSpace(record.ProjectCode),
			"deployType":  strings.TrimSpace(record.DeployType),
		}),
	}
}

func buildArchiveRecordDeletedEvent(record ArchiveRecordPayload) models.PlatformEvent {
	return models.PlatformEvent{
		EventID:       eventID("archive_deleted", sourceSystemDeploy, record.ID),
		EventType:     "archive_deleted",
		EventCategory: EventCategoryArchive,
		SourceSystem:  sourceSystemDeploy,
		SourceTable:   record.TableName(),
		SourceID:      uintString(record.ID),
		ObjectType:    "archive_record",
		ObjectID:      objectUID("archive_record", sourceSystemDeploy, record.ID),
		Title:         archiveEventTitle(record),
		Summary:       "归档记录已删除",
		Status:        "closed",
		Severity:      "info",
		OperatorID:    strings.TrimSpace(record.Operator),
		OperatorName:  strings.TrimSpace(record.Operator),
		TriggerMode:   "user",
		OccurredAt:    record.UpdatedAt,
	}
}

func buildAlertEvent(event AlertEventPayload) models.PlatformEvent {
	eventType := "alert_fired"
	switch strings.TrimSpace(event.Status) {
	case "acknowledged":
		eventType = "alert_acked"
	case "resolved":
		eventType = "alert_resolved"
	case "closed":
		eventType = "alert_closed"
	}

	objectID := objectUID("alert_event", sourceSystemAlert, event.ID)
	return models.PlatformEvent{
		EventID:       eventID(eventType, sourceSystemAlert, event.ID),
		EventType:     eventType,
		EventCategory: EventCategoryAlert,
		SourceSystem:  sourceSystemAlert,
		SourceTable:   event.TableName(),
		SourceID:      uintString(event.ID),
		ObjectType:    "alert_event",
		ObjectID:      objectID,
		Title:         defaultString(strings.TrimSpace(event.RuleName), "告警中心"),
		Summary:       alertEventSummary(event),
		Status:        strings.TrimSpace(event.Status),
		Severity:      strings.TrimSpace(event.Severity),
		OperatorID:    strings.TrimSpace(defaultString(event.AckedBy, event.ClosedBy)),
		OperatorName:  strings.TrimSpace(defaultString(event.AckedBy, event.ClosedBy)),
		TriggerMode:   "system",
		OccurredAt:    event.FiredAt,
		MetadataJSON: mustEventMetadataJSON(map[string]any{
			"category":     strings.TrimSpace(event.Category),
			"source":       strings.TrimSpace(event.Source),
			"notifyStatus": strings.TrimSpace(event.NotifyStatus),
			"fingerprint":  strings.TrimSpace(event.Fingerprint),
		}),
	}
}

func buildAlertDeletedEvent(event AlertEventPayload) models.PlatformEvent {
	return models.PlatformEvent{
		EventID:       eventID("alert_deleted", sourceSystemAlert, event.ID),
		EventType:     "alert_deleted",
		EventCategory: EventCategoryAlert,
		SourceSystem:  sourceSystemAlert,
		SourceTable:   event.TableName(),
		SourceID:      uintString(event.ID),
		ObjectType:    "alert_event",
		ObjectID:      objectUID("alert_event", sourceSystemAlert, event.ID),
		Title:         defaultString(strings.TrimSpace(event.RuleName), "告警中心"),
		Summary:       "告警已删除",
		Status:        "closed",
		Severity:      strings.TrimSpace(event.Severity),
		OperatorID:    strings.TrimSpace(defaultString(event.AckedBy, event.ClosedBy)),
		OperatorName:  strings.TrimSpace(defaultString(event.AckedBy, event.ClosedBy)),
		TriggerMode:   "user",
		OccurredAt:    time.Now(),
	}
}

func buildAssistantSessionEvent(session models.AssistantSession) models.PlatformEvent {
	eventType := "session_created"
	if strings.TrimSpace(session.Status) == "archived" {
		eventType = "session_archived"
	}

	return models.PlatformEvent{
		EventID:       eventID(eventType, sourceSystemAssistant, session.ID),
		EventType:     eventType,
		EventCategory: EventCategoryAssistant,
		SourceSystem:  sourceSystemAssistant,
		SourceTable:   session.TableName(),
		SourceID:      uintString(session.ID),
		ObjectType:    "assistant_session",
		ObjectID:      defaultString(strings.TrimSpace(session.SessionID), objectUID("assistant_session", sourceSystemAssistant, session.ID)),
		Title:         defaultString(strings.TrimSpace(session.Title), "assistant session"),
		Summary:       assistantSessionSummary(session),
		Status:        strings.TrimSpace(session.Status),
		OperatorID:    strings.TrimSpace(session.UserID),
		OperatorName:  strings.TrimSpace(session.UserID),
		TriggerMode:   "user",
		OccurredAt:    session.CreatedAt,
		MetadataJSON: mustEventMetadataJSON(map[string]any{
			"scene":  strings.TrimSpace(session.Scene),
			"pinned": session.Pinned,
		}),
	}
}

func buildAssistantSessionDeletedEvent(session models.AssistantSession) models.PlatformEvent {
	return models.PlatformEvent{
		EventID:       eventID("session_deleted", sourceSystemAssistant, session.ID),
		EventType:     "session_deleted",
		EventCategory: EventCategoryAssistant,
		SourceSystem:  sourceSystemAssistant,
		SourceTable:   session.TableName(),
		SourceID:      uintString(session.ID),
		ObjectType:    "assistant_session",
		ObjectID:      defaultString(strings.TrimSpace(session.SessionID), objectUID("assistant_session", sourceSystemAssistant, session.ID)),
		Title:         defaultString(strings.TrimSpace(session.Title), "assistant session"),
		Summary:       "会话已删除",
		Status:        "closed",
		Severity:      "info",
		OperatorID:    strings.TrimSpace(session.UserID),
		OperatorName:  strings.TrimSpace(session.UserID),
		TriggerMode:   "user",
		OccurredAt:    session.UpdatedAt,
		MetadataJSON: mustEventMetadataJSON(map[string]any{
			"scene":  strings.TrimSpace(session.Scene),
			"pinned": session.Pinned,
		}),
	}
}

func buildAssistantMessageEvent(message models.AssistantMessage) models.PlatformEvent {
	return models.PlatformEvent{
		EventID:       eventID("assistant_answered", sourceSystemAssistant, message.ID),
		EventType:     "assistant_answered",
		EventCategory: EventCategoryAssistant,
		SourceSystem:  sourceSystemAssistant,
		SourceTable:   message.TableName(),
		SourceID:      uintString(message.ID),
		ObjectType:    "assistant_session",
		ObjectID:      strings.TrimSpace(message.SessionID),
		Title:         defaultString(strings.TrimSpace(message.Intent), "assistant_answered"),
		Summary:       shorten(strings.TrimSpace(message.Content), 160),
		Status:        "success",
		Severity:      "info",
		OperatorID:    strings.TrimSpace(message.UserID),
		OperatorName:  strings.TrimSpace(message.UserID),
		TriggerMode:   "assistant",
		OccurredAt:    message.CreatedAt,
		MetadataJSON: mustEventMetadataJSON(map[string]any{
			"messageId":        strings.TrimSpace(message.MessageID),
			"intent":           strings.TrimSpace(message.Intent),
			"model":            strings.TrimSpace(message.ModelName),
			"promptTokens":     message.PromptTokens,
			"completionTokens": message.CompletionTokens,
			"latencyMs":        message.LatencyMS,
		}),
	}
}

func upsertEvent(db *gorm.DB, event models.PlatformEvent) error {
	var existing models.PlatformEvent
	err := db.Where("event_id = ?", event.EventID).First(&existing).Error
	if err == nil {
		return db.Model(&existing).Updates(map[string]any{
			"event_type":     event.EventType,
			"event_category": event.EventCategory,
			"source_system":  event.SourceSystem,
			"source_table":   event.SourceTable,
			"source_id":      event.SourceID,
			"object_type":    event.ObjectType,
			"object_id":      event.ObjectID,
			"title":          event.Title,
			"summary":        event.Summary,
			"status":         event.Status,
			"severity":       event.Severity,
			"operator_id":    event.OperatorID,
			"operator_name":  event.OperatorName,
			"trigger_mode":   event.TriggerMode,
			"started_at":     event.StartedAt,
			"finished_at":    event.FinishedAt,
			"occurred_at":    event.OccurredAt,
			"metadata_json":  event.MetadataJSON,
		}).Error
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	return db.Create(&event).Error
}

func persistEvent(event models.PlatformEvent) error {
	if database.DB == nil {
		return fmt.Errorf("nil db")
	}
	return upsertEvent(database.DB, event)
}
