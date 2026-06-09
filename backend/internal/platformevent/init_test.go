package platformevent

import (
	"strings"
	"testing"
	"time"

	"github.com/jenvenson/ops-platform/internal/models"
)

func TestEventID(t *testing.T) {
	got := eventID("deploy_failed", "deploy", 7)
	if got != "deploy_failed:deploy:7" {
		t.Fatalf("unexpected event id: %q", got)
	}
}

func TestBuildDeployRecordEvent(t *testing.T) {
	now := time.Now()
	record := DeployRecordPayload{
		ID:           18,
		AppID:        3,
		AppName:      "web-api",
		EnvID:        5,
		EnvName:      "prod",
		ProjectCode:  "ops",
		DeployType:   "backend",
		Status:       "failed",
		ErrorMessage: "jenkins timeout",
		TriggeredBy:  "admin",
		CreatedAt:    now,
	}

	event := buildDeployRecordEvent(record)
	if event.EventType != "deploy_failed" || event.EventCategory != EventCategoryDeploy {
		t.Fatalf("unexpected deploy event: %#v", event)
	}
	if event.ObjectID != "deploy_record:deploy:18" {
		t.Fatalf("unexpected deploy object id: %#v", event)
	}
	if event.Severity != "high" {
		t.Fatalf("expected high severity, got %#v", event)
	}
}

func TestBuildAlertEvent(t *testing.T) {
	now := time.Now()
	event := buildAlertEvent(AlertEventPayload{
		ID:         9,
		RuleName:   "CPU High",
		Severity:   "critical",
		Status:     "resolved",
		Content:    "cpu usage > 90%",
		Source:     "node-01",
		FiredAt:    now,
		ResolvedAt: &now,
	})

	if event.EventType != "alert_resolved" || event.EventCategory != EventCategoryAlert {
		t.Fatalf("unexpected alert event: %#v", event)
	}
	if event.ObjectID != "alert_event:alert:9" {
		t.Fatalf("unexpected alert object id: %#v", event)
	}
	if !strings.Contains(event.Summary, "来源：node-01") {
		t.Fatalf("unexpected alert summary: %#v", event)
	}
}

func TestBuildRealtimeEvents(t *testing.T) {
	now := time.Now()

	deployDeleted := buildDeployRecordDeletedEvent(DeployRecordPayload{
		ID:          21,
		AppName:     "web-api",
		EnvName:     "prod",
		TriggeredBy: "admin",
		UpdatedAt:   now,
	})
	if deployDeleted.EventType != "deploy_deleted" || deployDeleted.Status != "closed" {
		t.Fatalf("unexpected deploy deleted event: %#v", deployDeleted)
	}

	archiveDeleted := buildArchiveRecordDeletedEvent(ArchiveRecordPayload{
		ID:        22,
		AppName:   "web-api",
		EnvName:   "prod",
		Operator:  "admin",
		UpdatedAt: now,
	})
	if archiveDeleted.EventType != "archive_deleted" || archiveDeleted.Status != "closed" {
		t.Fatalf("unexpected archive deleted event: %#v", archiveDeleted)
	}

	alertDeleted := buildAlertDeletedEvent(AlertEventPayload{
		ID:       11,
		RuleName: "CPU High",
		Severity: "critical",
		AckedBy:  "admin",
	})
	if alertDeleted.EventType != "alert_deleted" || alertDeleted.Status != "closed" {
		t.Fatalf("unexpected alert deleted event: %#v", alertDeleted)
	}
}

func TestBuildAssistantEvents(t *testing.T) {
	session := buildAssistantSessionEvent(models.AssistantSession{
		ID:        4,
		SessionID: "asst_123",
		Status:    "archived",
		Scene:     "web",
		Summary:   "查看归档历史",
		CreatedAt: time.Now(),
	})
	if session.EventType != "session_archived" || session.EventCategory != EventCategoryAssistant {
		t.Fatalf("unexpected assistant session event: %#v", session)
	}
	if session.ObjectID != "asst_123" {
		t.Fatalf("unexpected assistant session object id: %#v", session)
	}

	message := buildAssistantMessageEvent(models.AssistantMessage{
		ID:        5,
		SessionID: "asst_123",
		MessageID: "msg_123",
		UserID:    "admin",
		Intent:    "readonly_query",
		Content:   "当前没有匹配的归档历史记录。",
		ModelName: "qwen3:8b",
		CreatedAt: time.Now(),
	})
	if message.EventType != "assistant_answered" {
		t.Fatalf("unexpected assistant message event: %#v", message)
	}
	if message.ObjectID != "asst_123" {
		t.Fatalf("unexpected assistant message object id: %#v", message)
	}
	if !strings.Contains(message.Summary, "归档历史") {
		t.Fatalf("unexpected assistant message summary: %#v", message)
	}
}

func TestBuildAssistantSessionDeletedEvent(t *testing.T) {
	event := buildAssistantSessionDeletedEvent(models.AssistantSession{
		ID:        7,
		SessionID: "asst_456",
		Title:     "归档历史排查",
		UserID:    "admin",
		Scene:     "web",
		UpdatedAt: time.Now(),
	})
	if event.EventType != "session_deleted" || event.Status != "closed" {
		t.Fatalf("unexpected deleted session event: %#v", event)
	}
	if event.ObjectID != "asst_456" {
		t.Fatalf("unexpected deleted session object id: %#v", event)
	}
}
