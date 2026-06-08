package assistant

import (
	"strconv"
	"strings"

	"gorm.io/gorm"
)

func pageFilterValue(pageContext *AssistantPageContext, keys ...string) string {
	if pageContext == nil || len(pageContext.Filters) == 0 {
		return ""
	}

	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if value := strings.TrimSpace(pageContext.Filters[key]); value != "" {
			return value
		}
	}

	return ""
}

func pageFilterUint(pageContext *AssistantPageContext, keys ...string) (uint, bool) {
	value := pageFilterValue(pageContext, keys...)
	if value == "" {
		return 0, false
	}

	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil || parsed == 0 {
		return 0, false
	}
	return uint(parsed), true
}

func pageObjectID(pageContext *AssistantPageContext) (uint, bool) {
	if pageContext == nil {
		return 0, false
	}

	value := strings.TrimSpace(pageContext.ObjectID)
	if value == "" && len(pageContext.SelectedRecordIDs) > 0 {
		value = strings.TrimSpace(pageContext.SelectedRecordIDs[0])
	}
	if value == "" {
		return 0, false
	}

	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil || parsed == 0 {
		return 0, false
	}
	return uint(parsed), true
}

func pageObjectType(pageContext *AssistantPageContext) string {
	if pageContext == nil {
		return ""
	}
	return strings.TrimSpace(pageContext.ObjectType)
}

func pageHasObjectContext(pageContext *AssistantPageContext, objectTypes ...string) bool {
	if pageContext == nil {
		return false
	}
	if _, ok := pageObjectID(pageContext); !ok {
		return false
	}
	if len(objectTypes) == 0 {
		return pageObjectType(pageContext) != ""
	}

	current := pageObjectType(pageContext)
	for _, objectType := range objectTypes {
		if current == strings.TrimSpace(objectType) {
			return true
		}
	}
	return false
}

func shouldUseFocusedObjectQuery(message string, pageContext *AssistantPageContext, objectTypes ...string) bool {
	if !pageHasObjectContext(pageContext, objectTypes...) {
		return false
	}

	msg := strings.ToLower(strings.TrimSpace(message))
	return containsAny(msg,
		"这条", "这个", "当前这条", "当前这个", "当前这次", "这次", "该记录", "当前记录", "这个记录",
		"这个告警", "这条告警", "该告警", "这个漏洞", "这条漏洞", "该漏洞",
		"这次部署", "该部署", "这个部署", "这条部署", "这个归档", "这条归档", "该归档",
		"是什么意思", "什么情况", "失败点", "失败原因", "怎么处理", "要不要", "怎么下载", "怎么看",
	)
}

func applyCreatedAtRange(query *gorm.DB, pageContext *AssistantPageContext) *gorm.DB {
	if query == nil || pageContext == nil {
		return query
	}

	if start := pageFilterValue(pageContext, "startTime", "start_time"); start != "" {
		query = query.Where("created_at >= ?", start)
	}
	if end := pageFilterValue(pageContext, "endTime", "end_time"); end != "" {
		query = query.Where("created_at <= ?", end)
	}
	return query
}
