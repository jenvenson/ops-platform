// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package cmdb

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFindLatestAggregatedPackageURLFromBase(t *testing.T) {
	lastModified := map[string]time.Time{
		"/aggregation/a.tar": time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC),
		"/aggregation/b.tar": time.Date(2026, 3, 12, 11, 0, 0, 0, time.UTC),
		"/aggregation/c.tar": time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if r.URL.Path != "/aggregation/" {
				http.NotFound(w, r)
				return
			}
			fmt.Fprint(w, `
<html><body>
<a href="a.tar">a.tar</a>
<a href="b.tar">b.tar</a>
<a href="/aggregation/c.tar">c.tar</a>
</body></html>`)
		case http.MethodHead:
			modified, ok := lastModified[r.URL.Path]
			if !ok {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Last-Modified", modified.Format(http.TimeFormat))
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	latest := findLatestAggregatedPackageURLFromBase(server.URL + "/aggregation/")
	expected := server.URL + "/aggregation/b.tar"
	if latest != expected {
		t.Fatalf("expected latest tar %q, got %q", expected, latest)
	}
}