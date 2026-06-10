// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package assistant

import "testing"

func TestDeriveAlertQueryOptions(t *testing.T) {
	t.Run("latest alert actions prefer unresolved ordered by update time", func(t *testing.T) {
		options := deriveAlertQueryOptions("最新告警动作")
		if options.orderBy != "updated_at desc" {
			t.Fatalf("expected updated_at desc, got %#v", options)
		}
		if len(options.statuses) != 2 || options.statuses[0] != "firing" || options.statuses[1] != "acknowledged" {
			t.Fatalf("expected unresolved statuses, got %#v", options)
		}
		if options.keyword != "" {
			t.Fatalf("expected no keyword filter, got %#v", options)
		}
	})

	t.Run("resolved alerts keep resolved status", func(t *testing.T) {
		options := deriveAlertQueryOptions("查看已恢复告警")
		if len(options.statuses) != 1 || options.statuses[0] != "resolved" {
			t.Fatalf("expected resolved filter, got %#v", options)
		}
	})

	t.Run("specific keyword survives cleanup", func(t *testing.T) {
		options := deriveAlertQueryOptions("查看数据库异常告警")
		if options.keyword != "数据库" {
			t.Fatalf("expected 数据库 keyword, got %#v", options)
		}
	})
}