// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package cmdb

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jenvenson/ops-platform/pkg/jenkins"
)

func TestArchiveBuildNumberFromRecentBuildsMatchesParameters(t *testing.T) {
	start := time.Date(2026, 5, 14, 15, 37, 18, 0, time.Local)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/job/gkd-app_on-site_update/api/json":
			_, _ = w.Write([]byte(`{"builds":[{"number":33},{"number":32},{"number":31}]}`))
		case "/job/gkd-app_on-site_update/33/api/json":
			_, _ = w.Write([]byte(buildInfoJSON("app-base", "gkd", "all", start.Add(2*time.Minute))))
		case "/job/gkd-app_on-site_update/32/api/json":
			_, _ = w.Write([]byte(buildInfoJSON("app-pex", "gkd", "all", start.Add(1*time.Minute))))
		case "/job/gkd-app_on-site_update/31/api/json":
			_, _ = w.Write([]byte(buildInfoJSON("app-cex", "gkd", "all", start.Add(30*time.Second))))
		default:
			t.Fatalf("unexpected Jenkins API path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	record := &ArchiveRecord{
		AppName:        "app-pex",
		EnvName:        "gkd",
		DeployType:     "all",
		JenkinsJob:     "gkd-app_on-site_update",
		StartTime:      &start,
		JenkinsQueueID: 94081,
	}
	client := jenkins.NewClient(server.URL, "admin", "token")

	buildNum := archiveBuildNumberFromRecentBuilds(record, client)
	if buildNum != 32 {
		t.Fatalf("expected build 32, got %d", buildNum)
	}
}

func TestArchiveBuildMatchesRecordRejectsOlderBuild(t *testing.T) {
	start := time.Date(2026, 5, 14, 15, 37, 18, 0, time.Local)
	record := &ArchiveRecord{
		AppName:    "app-pex",
		EnvName:    "gkd",
		DeployType: "all",
		StartTime:  &start,
	}

	buildInfo := map[string]interface{}{
		"timestamp": float64(start.Add(-10 * time.Minute).UnixMilli()),
		"actions": []interface{}{
			map[string]interface{}{
				"parameters": []interface{}{
					map[string]interface{}{"name": "app", "value": "app-pex"},
					map[string]interface{}{"name": "tag", "value": "gkd"},
					map[string]interface{}{"name": "scope", "value": "all"},
				},
			},
		},
	}

	if archiveBuildMatchesRecord(record, buildInfo) {
		t.Fatal("expected old build to be rejected")
	}
}

func buildInfoJSON(app, tag, scope string, timestamp time.Time) string {
	return `{"timestamp":` +
		fmt.Sprintf("%d", timestamp.UnixMilli()) +
		`,"actions":[{"parameters":[{"name":"app","value":"` + app +
		`"},{"name":"tag","value":"` + tag +
		`"},{"name":"scope","value":"` + scope + `"}]}]}`
}