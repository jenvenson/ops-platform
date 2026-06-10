// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package platformobject

import (
	"strings"
	"testing"
	"time"

	"github.com/jenvenson/ops-platform/internal/cmdb"
)

func TestObjectUID(t *testing.T) {
	got := objectUID(ObjectTypeProject, sourceModuleCMDB, 12)
	if got != "project:cmdb:12" {
		t.Fatalf("unexpected object uid: %q", got)
	}
}

func TestBuildProjectObject(t *testing.T) {
	project := cmdb.Project{
		ID:          3,
		Name:        "OPS",
		Code:        "ops",
		Description: "运维平台项目",
	}

	object := buildProjectObject(project)
	if object.ObjectUID != "project:cmdb:3" {
		t.Fatalf("unexpected object uid: %#v", object)
	}
	if object.ObjectType != ObjectTypeProject || object.SourceModule != sourceModuleCMDB {
		t.Fatalf("unexpected object identity: %#v", object)
	}
	if object.Status != "active" {
		t.Fatalf("expected active status, got %#v", object)
	}
	if !strings.Contains(object.Summary, "项目编号：ops") {
		t.Fatalf("expected project summary to include code, got %#v", object)
	}
}

func TestBuildApplicationObject(t *testing.T) {
	app := cmdb.Application{
		ID:         8,
		Name:       "web-api",
		ProjectID:  2,
		EnvID:      5,
		JenkinsJob: "ops/web-api",
		Project:    &cmdb.Project{Name: "OPS"},
		Environment: &cmdb.Environment{
			Name: "prod",
		},
	}

	object := buildApplicationObject(app)
	if object.ObjectUID != "application:cmdb:8" {
		t.Fatalf("unexpected object uid: %#v", object)
	}
	if !strings.Contains(object.Summary, "项目：OPS") || !strings.Contains(object.Summary, "环境：prod") {
		t.Fatalf("unexpected application summary: %#v", object)
	}
}

func TestBuildDeployRecordObject(t *testing.T) {
	now := time.Now()
	record := cmdb.DeployRecord{
		ID:           18,
		AppID:        8,
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

	object := buildDeployRecordObject(record)
	if object.ObjectUID != "deploy_record:cmdb:18" {
		t.Fatalf("unexpected object uid: %#v", object)
	}
	if object.OwnerID != "admin" || object.Status != "failed" {
		t.Fatalf("unexpected deploy record ownership/status: %#v", object)
	}
	if !strings.Contains(object.Summary, "错误：jenkins timeout") {
		t.Fatalf("expected deploy summary to include error, got %#v", object)
	}
}