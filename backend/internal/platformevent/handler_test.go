package platformevent

import (
	"testing"
	"time"

	"github.com/jenvenson/ops-platform/internal/models"
)

func TestClampPageSize(t *testing.T) {
	if got := clampPageSize(0); got != defaultPageSize {
		t.Fatalf("expected default page size, got %d", got)
	}
	if got := clampPageSize(500); got != maxPageSize {
		t.Fatalf("expected max page size, got %d", got)
	}
	if got := clampPageSize(50); got != 50 {
		t.Fatalf("expected unchanged page size, got %d", got)
	}
}

func TestParseDateTime(t *testing.T) {
	from, err := parseDateTime("2026-03-27", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if from == nil || from.Format("2006-01-02 15:04:05") != "2026-03-27 00:00:00" {
		t.Fatalf("unexpected date start: %#v", from)
	}

	to, err := parseDateTime("2026-03-27", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if to == nil || to.Format("2006-01-02 15:04:05") != "2026-03-27 23:59:59" {
		t.Fatalf("unexpected date end: %#v", to)
	}

	rfc3339, err := parseDateTime("2026-03-27T12:34:56Z", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rfc3339 == nil || rfc3339.UTC().Format(time.RFC3339) != "2026-03-27T12:34:56Z" {
		t.Fatalf("unexpected RFC3339 parse result: %#v", rfc3339)
	}

	if _, err := parseDateTime("not-a-date", false); err == nil {
		t.Fatal("expected invalid date error")
	}
}

func TestParseMetadata(t *testing.T) {
	got := parseMetadata(`{"appName":"web-api","envName":"prod"}`)
	if got["appName"] != "web-api" || got["envName"] != "prod" {
		t.Fatalf("unexpected metadata: %#v", got)
	}

	empty := parseMetadata("{broken")
	if len(empty) != 0 {
		t.Fatalf("expected empty metadata on invalid JSON, got %#v", empty)
	}
}

func TestToEventListItem(t *testing.T) {
	now := time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)
	item := toEventListItem(models.PlatformEvent{
		ID:            7,
		EventID:       "deploy_failed:deploy:7",
		EventType:     "deploy_failed",
		EventCategory: EventCategoryDeploy,
		SourceSystem:  "deploy",
		SourceTable:   "deploy_records",
		SourceID:      "7",
		ObjectType:    "deploy_record",
		ObjectID:      "deploy_record:deploy:7",
		Title:         "web-api / prod",
		Summary:       "状态：failed",
		Status:        "failed",
		Severity:      "high",
		OperatorName:  "admin",
		TriggerMode:   "user",
		OccurredAt:    now,
		MetadataJSON:  `{"appName":"web-api"}`,
		CreatedAt:     now,
		UpdatedAt:     now,
	})

	if item.EventID != "deploy_failed:deploy:7" {
		t.Fatalf("unexpected event id: %#v", item)
	}
	if item.Metadata["appName"] != "web-api" {
		t.Fatalf("unexpected metadata in list item: %#v", item.Metadata)
	}
}

func TestClampTimelineLimit(t *testing.T) {
	if got := clampPageSize(200); got != maxPageSize {
		t.Fatalf("expected max page size for timeline, got %d", got)
	}
}
