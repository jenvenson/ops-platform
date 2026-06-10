// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package assistant

import (
	"net/http/httptest"
	"testing"
)

func TestOllamaRequestHostForDockerHost(t *testing.T) {
	got := ollamaRequestHost("http://docker-host:11434")
	if got != "127.0.0.1:11434" {
		t.Fatalf("expected docker-host to override host header, got %q", got)
	}

	got = ollamaRequestHost("http://host.docker.internal:11434")
	if got != "127.0.0.1:11434" {
		t.Fatalf("expected host.docker.internal to override host header, got %q", got)
	}

	got = ollamaRequestHost("http://127.0.0.1:11434")
	if got != "" {
		t.Fatalf("expected localhost address to keep original host header, got %q", got)
	}
}

func TestPrepareRequestOverridesHostForDockerHost(t *testing.T) {
	client := newOllamaClient("http://docker-host:11434", "qwen3:8b", 0.2)
	req := httptest.NewRequest("POST", "http://docker-host:11434/api/chat", nil)

	client.prepareRequest(req)

	if req.Host != "127.0.0.1:11434" {
		t.Fatalf("expected overridden host header, got %q", req.Host)
	}
}