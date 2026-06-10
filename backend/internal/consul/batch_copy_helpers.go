// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package consul

import "sort"

func collectProjectNames(keys []string) []string {
	projectSet := make(map[string]struct{})
	for _, key := range keys {
		key = trimPluginPrefix(key)
		if key == "" {
			continue
		}
		project, _, _ := splitFirstSegment(key)
		if project == "" {
			continue
		}
		projectSet[project] = struct{}{}
	}

	projects := make([]string, 0, len(projectSet))
	for project := range projectSet {
		projects = append(projects, project)
	}
	sort.Strings(projects)
	return projects
}

func filterCopySourceKeys(keys []string, sourcePrefix string) []string {
	filtered := make([]string, 0, len(keys))
	for _, key := range keys {
		if key == "" || key == sourcePrefix+"/" {
			continue
		}
		filtered = append(filtered, key)
	}
	return filtered
}

func trimPluginPrefix(key string) string {
	const pluginPrefix = "plugin/"
	if len(key) >= len(pluginPrefix) && key[:len(pluginPrefix)] == pluginPrefix {
		return key[len(pluginPrefix):]
	}
	return key
}

func splitFirstSegment(s string) (head, tail string, ok bool) {
	for i := 0; i < len(s); i++ {
		if s[i] == '/' {
			return s[:i], s[i+1:], true
		}
	}
	return s, "", s != ""
}