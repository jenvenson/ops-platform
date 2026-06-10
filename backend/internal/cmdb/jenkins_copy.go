// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package cmdb

import (
	"regexp"
	"strings"
)

type compiledTagPattern struct {
	re *regexp.Regexp
}

func normalizeJobNameReplacements(rules []JobNameReplacementRule) []JobNameReplacementRule {
	if len(rules) <= 1 {
		return rules
	}

	lastIndexByOldPattern := make(map[string]int, len(rules))
	for i, rule := range rules {
		if rule.OldPattern == "" || rule.NewPattern == "" {
			continue
		}
		lastIndexByOldPattern[rule.OldPattern] = i
	}

	normalized := make([]JobNameReplacementRule, 0, len(lastIndexByOldPattern))
	for i, rule := range rules {
		if rule.OldPattern == "" || rule.NewPattern == "" {
			continue
		}
		if lastIndexByOldPattern[rule.OldPattern] != i {
			continue
		}
		normalized = append(normalized, rule)
	}
	return normalized
}

func extractAppNameFromJob(viewName, jobName string) string {
	return extractAppNameFromJobWithPrefix(viewName, jobName, "")
}

func extractAppNameFromJobWithPrefix(viewName, jobName, explicitPrefix string) string {
	explicitPrefix = strings.TrimSpace(explicitPrefix)
	if trimmed := trimExplicitAppPrefix(jobName, explicitPrefix); trimmed != jobName {
		return trimmed
	}
	for _, prefix := range candidateJobPrefixes(viewName) {
		if prefix != "" && strings.HasPrefix(jobName, prefix) {
			trimmed := trimDerivedJobPrefix(strings.TrimPrefix(jobName, prefix))
			return trimExplicitAppPrefix(trimmed, explicitPrefix)
		}
	}
	return jobName
}

func trimExplicitAppPrefix(jobName, explicitPrefix string) string {
	if explicitPrefix != "" && strings.HasPrefix(jobName, explicitPrefix) {
		return strings.TrimPrefix(jobName, explicitPrefix)
	}
	return jobName
}

func trimDerivedJobPrefix(jobName string) string {
	re := regexp.MustCompile(`^\d+-`)
	if re.MatchString(jobName) {
		return re.ReplaceAllString(jobName, "")
	}
	return jobName
}

func candidateJobPrefixes(viewName string) []string {
	prefixes := make([]string, 0, 3)
	seen := make(map[string]struct{}, 3)

	add := func(prefix string) {
		if prefix == "" {
			return
		}
		if _, ok := seen[prefix]; ok {
			return
		}
		seen[prefix] = struct{}{}
		prefixes = append(prefixes, prefix)
	}

	primary := extractJobPrefix(viewName)
	add(primary)

	normalizedView := strings.ReplaceAll(viewName, "-", "")
	if normalizedView != "" {
		add(normalizedView + "-")
	}

	if primary != "" {
		normalizedPrimary := strings.ReplaceAll(strings.TrimSuffix(primary, "-"), "-", "")
		if normalizedPrimary != "" {
			add(normalizedPrimary + "-")
		}
	}

	return prefixes
}

func applyTagReplacements(config string, replacements []TagReplacementRule) string {
	updated := config
	for _, replacement := range replacements {
		if replacement.OldPattern == "" || replacement.NewPattern == "" {
			continue
		}
		updated = applyTagReplacement(updated, replacement)
	}
	return updated
}

func applyTagReplacement(config string, replacement TagReplacementRule) string {
	oldEscaped := regexp.QuoteMeta(replacement.OldPattern)
	patterns := []compiledTagPattern{
		{re: regexp.MustCompile(`(?m)\btag\s*:\s*"` + oldEscaped + `"`)},
		{re: regexp.MustCompile(`(?m)\btag\s*=\s*'` + oldEscaped + `'`)},
		{re: regexp.MustCompile(`(?m)\btag\s*=\s*"` + oldEscaped + `"`)},
		{re: regexp.MustCompile(`(?m)\btag\s*:\s*'` + oldEscaped + `'`)},
		{re: regexp.MustCompile(`(?m)\btag\s*=\s*` + oldEscaped + `(\s|[,)\r\n]|$)`)},
		{re: regexp.MustCompile(`(?m)\btag\s*:\s*` + oldEscaped + `(\s|[,)\r\n]|$)`)},
		{re: regexp.MustCompile(`(?m)\bname\s*:\s*'tag'\s*,\s*defaultValue\s*:\s*'` + oldEscaped + `'`)},
		{re: regexp.MustCompile(`(?m)\bname\s*:\s*"tag"\s*,\s*defaultValue\s*:\s*"` + oldEscaped + `"`)},
		{re: regexp.MustCompile(`(?m)\bname\s*:\s*'tag'\s*,\s*defaultValue\s*:\s*` + oldEscaped + `(\s|[,)\r\n]|$)`)},
		{re: regexp.MustCompile(`(?m)\bname\s*:\s*"tag"\s*,\s*defaultValue\s*:\s*` + oldEscaped + `(\s|[,)\r\n]|$)`)},
		{re: regexp.MustCompile(`(?s)string\s*\(\s*name\s*:\s*['"]tag['"][^)]*?\bdefaultValue\s*:\s*'` + oldEscaped + `'`)},
		{re: regexp.MustCompile(`(?s)string\s*\(\s*name\s*:\s*['"]tag['"][^)]*?\bdefaultValue\s*:\s*"` + oldEscaped + `"`)},
		{re: regexp.MustCompile(`(?s)string\s*\(\s*name\s*:\s*['"]tag['"][^)]*?\bdefaultValue\s*:\s*` + oldEscaped + `(\s|[,)\r\n]|$)`)},
	}

	updated := config
	for _, pattern := range patterns {
		updated = pattern.re.ReplaceAllStringFunc(updated, func(match string) string {
			return strings.Replace(match, replacement.OldPattern, replacement.NewPattern, 1)
		})
	}
	return updated
}