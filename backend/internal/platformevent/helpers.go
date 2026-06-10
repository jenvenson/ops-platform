// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package platformevent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jenvenson/ops-platform/internal/models"
)

func eventID(eventType, sourceSystem string, id uint) string {
	return fmt.Sprintf("%s:%s:%d", strings.TrimSpace(eventType), strings.TrimSpace(sourceSystem), id)
}

func objectUID(objectType, sourceSystem string, id uint) string {
	return fmt.Sprintf("%s:%s:%d", strings.TrimSpace(objectType), strings.TrimSpace(sourceSystem), id)
}

func uintString(value uint) string {
	return fmt.Sprintf("%d", value)
}

func mustEventMetadataJSON(value map[string]any) string {
	data, err := json.Marshal(value)
	if err != nil || string(data) == "null" {
		return "{}"
	}
	return string(data)
}

func deployEventTitle(record DeployRecordPayload) string {
	name := strings.TrimSpace(record.AppName)
	env := strings.TrimSpace(record.EnvName)
	if name != "" && env != "" {
		return name + " / " + env
	}
	if name != "" {
		return name
	}
	if env != "" {
		return env
	}
	return "deploy event"
}

func deployEventSummary(record DeployRecordPayload) string {
	parts := []string{"状态：" + defaultString(strings.TrimSpace(record.Status), "unknown")}
	if deployType := strings.TrimSpace(record.DeployType); deployType != "" {
		parts = append(parts, "类型："+deployType)
	}
	if projectCode := strings.TrimSpace(record.ProjectCode); projectCode != "" {
		parts = append(parts, "项目："+projectCode)
	}
	if operator := strings.TrimSpace(record.TriggeredBy); operator != "" {
		parts = append(parts, "触发人："+operator)
	}
	if errMsg := strings.TrimSpace(record.ErrorMessage); errMsg != "" && strings.EqualFold(strings.TrimSpace(record.Status), "failed") {
		parts = append(parts, "错误："+shorten(errMsg, 80))
	}
	return strings.Join(parts, " | ")
}

func deploySeverity(status string) string {
	if strings.EqualFold(strings.TrimSpace(status), "failed") {
		return "high"
	}
	return "info"
}

func archiveEventTitle(record ArchiveRecordPayload) string {
	name := strings.TrimSpace(record.AppName)
	env := strings.TrimSpace(record.EnvName)
	if name != "" && env != "" {
		return name + " / " + env
	}
	if name != "" {
		return name
	}
	if env != "" {
		return env
	}
	return "archive event"
}

func archiveEventSummary(record ArchiveRecordPayload) string {
	parts := []string{"状态：" + defaultString(strings.TrimSpace(record.Status), "unknown")}
	if deployType := strings.TrimSpace(record.DeployType); deployType != "" {
		parts = append(parts, "类型："+deployType)
	}
	if projectCode := strings.TrimSpace(record.ProjectCode); projectCode != "" {
		parts = append(parts, "项目："+projectCode)
	}
	if operator := strings.TrimSpace(record.Operator); operator != "" {
		parts = append(parts, "操作人："+operator)
	}
	if errMsg := strings.TrimSpace(record.ErrorMessage); errMsg != "" && strings.EqualFold(strings.TrimSpace(record.Status), "failed") {
		parts = append(parts, "错误："+shorten(errMsg, 80))
	}
	return strings.Join(parts, " | ")
}

func archiveSeverity(status string) string {
	if strings.EqualFold(strings.TrimSpace(status), "failed") {
		return "high"
	}
	return "info"
}

func alertEventSummary(event AlertEventPayload) string {
	parts := []string{
		"级别：" + defaultString(strings.TrimSpace(event.Severity), "unknown"),
		"状态：" + defaultString(strings.TrimSpace(event.Status), "unknown"),
	}
	if source := strings.TrimSpace(event.Source); source != "" {
		parts = append(parts, "来源："+source)
	}
	if content := strings.TrimSpace(event.Content); content != "" {
		parts = append(parts, "内容："+shorten(content, 80))
	}
	return strings.Join(parts, " | ")
}

func assistantSessionSummary(session models.AssistantSession) string {
	parts := []string{"状态：" + defaultString(strings.TrimSpace(session.Status), "active")}
	if scene := strings.TrimSpace(session.Scene); scene != "" {
		parts = append(parts, "场景："+scene)
	}
	if summary := strings.TrimSpace(session.Summary); summary != "" {
		parts = append(parts, shorten(summary, 80))
	}
	return strings.Join(parts, " | ")
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func shorten(text string, max int) string {
	text = strings.TrimSpace(text)
	if max <= 0 || len([]rune(text)) <= max {
		return text
	}
	runes := []rune(text)
	return string(runes[:max]) + "..."
}