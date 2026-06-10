// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package security

import (
	"strings"
	"testing"
)

func TestBuildAuthenticatedWebSessionRejectsAnonymous(t *testing.T) {
	if _, err := BuildAuthenticatedWebSession("https://demo.example.com", &WebScanConfig{}); err == nil {
		t.Fatal("expected authenticated session builder to reject anonymous config")
	}
}

func TestBuildAuthenticatedWebSessionAcceptsResolvedHeader(t *testing.T) {
	session, err := BuildAuthenticatedWebSession("https://demo.example.com", &WebScanConfig{
		AuthMode:   "bearer",
		Credential: "test-token",
	})
	if err != nil {
		t.Fatalf("expected authenticated session, got error: %v", err)
	}
	if session == nil || len(session.Headers) == 0 {
		t.Fatal("expected resolved auth headers")
	}
	if session.Headers[0].Name != "Authorization" {
		t.Fatalf("expected Authorization header, got %q", session.Headers[0].Name)
	}
}

func TestAppendNucleiAuthArgsPrefersProvidedSession(t *testing.T) {
	args, err := appendNucleiAuthArgs([]string{"-u", "https://demo.example.com"}, "https://demo.example.com", &WebScanConfig{
		AuthMode: "advanced",
	}, &WebSession{
		Headers: []AuthHeader{
			{Name: "token", Value: "cached-token"},
			{Name: "tenantkey", Value: "default"},
		},
	})
	if err != nil {
		t.Fatalf("expected cached session headers to be accepted, got error: %v", err)
	}

	joined := strings.Join(args, "\n")
	if !strings.Contains(joined, "token: cached-token") {
		t.Fatalf("expected nuclei args to include cached token header, got %v", args)
	}
	if !strings.Contains(joined, "tenantkey: default") {
		t.Fatalf("expected nuclei args to include cached tenant header, got %v", args)
	}
}