package platformobject

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/edy/ops-platform/internal/cmdb"
)

func objectUID(objectType, sourceModule string, id uint) string {
	return fmt.Sprintf("%s:%s:%d", strings.TrimSpace(objectType), strings.TrimSpace(sourceModule), id)
}

func uintString(value uint) string {
	return fmt.Sprintf("%d", value)
}

func mustObjectMetadataJSON(value map[string]any) string {
	data, err := json.Marshal(value)
	if err != nil || string(data) == "null" {
		return "{}"
	}
	return string(data)
}

func objectStatus(active bool, current string, fallback string) string {
	if strings.TrimSpace(current) != "" {
		return strings.TrimSpace(current)
	}
	if active {
		if strings.TrimSpace(fallback) != "" {
			return strings.TrimSpace(fallback)
		}
		return "active"
	}
	return "deleted"
}

func projectSummary(project cmdb.Project) string {
	parts := make([]string, 0, 2)
	if code := strings.TrimSpace(project.Code); code != "" {
		parts = append(parts, "项目编号："+code)
	}
	if desc := strings.TrimSpace(project.Description); desc != "" {
		parts = append(parts, shorten(desc, 80))
	}
	return strings.Join(parts, " | ")
}

func applicationSummary(app cmdb.Application, projectName, envName string) string {
	parts := make([]string, 0, 3)
	if projectName != "" {
		parts = append(parts, "项目："+projectName)
	}
	if envName != "" {
		parts = append(parts, "环境："+envName)
	}
	if job := strings.TrimSpace(app.JenkinsJob); job != "" {
		parts = append(parts, "Jenkins："+job)
	}
	return strings.Join(parts, " | ")
}

func deployRecordTitle(record cmdb.DeployRecord) string {
	appName := strings.TrimSpace(record.AppName)
	envName := strings.TrimSpace(record.EnvName)
	if appName != "" && envName != "" {
		return appName + " / " + envName
	}
	if appName != "" {
		return appName
	}
	if envName != "" {
		return envName
	}
	return "deploy_record"
}

func deployRecordSummary(record cmdb.DeployRecord) string {
	parts := []string{
		"状态：" + defaultString(strings.TrimSpace(record.Status), "unknown"),
	}
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

func defaultString(value string, fallback string) string {
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
