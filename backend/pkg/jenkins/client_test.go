package jenkins

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestApprovePendingScriptApprovalItems(t *testing.T) {
	var approvedScripts []string
	var approvedSignatures []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/scriptApproval/":
			_, _ = w.Write([]byte(`
				<html><body>
				<script>
				var mgr = makeStaplerProxy('/$stapler/bound/token-1','proxy-crumb',['approveScript','approveSignature','aclApproveSignature']);
				</script>
				<div id="ps-hash-1"><button onclick="approveScript('hash-1')">Approve</button></div>
				<div id="ps-hash-2"><button onclick="approveScript('hash-2')">Approve</button></div>
				<div id="s-1"><button onclick="approveSignature('method hudson.model.ItemGroup getItems', 's-1')">Approve</button></div>
				</body></html>
			`))
		case r.Method == http.MethodPost && r.URL.Path == "/$stapler/bound/token-1/approveScript":
			if r.Header.Get("Crumb") != "proxy-crumb" {
				t.Fatalf("missing crumb header, got %q", r.Header.Get("Crumb"))
			}
			if got := r.Header.Get("Content-Type"); !strings.HasPrefix(got, "application/x-stapler-method-invocation") {
				t.Fatalf("unexpected content type %q", got)
			}
			var payload []string
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode payload: %v", err)
			}
			approvedScripts = append(approvedScripts, payload...)
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && r.URL.Path == "/$stapler/bound/token-1/approveSignature":
			var payload []string
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode payload: %v", err)
			}
			approvedSignatures = append(approvedSignatures, payload...)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "user", "token", 5*time.Second)
	result, err := client.ApprovePendingScriptApprovalItems()
	if err != nil {
		t.Fatalf("ApprovePendingScriptApprovalItems returned error: %v", err)
	}
	if result.ApprovedScripts != 2 || result.ApprovedSignatures != 1 {
		t.Fatalf("unexpected approval result: %#v", result)
	}
	if len(approvedScripts) != 2 || approvedScripts[0] != "hash-1" || approvedScripts[1] != "hash-2" {
		t.Fatalf("unexpected approved script hashes: %#v", approvedScripts)
	}
	if len(approvedSignatures) != 1 || approvedSignatures[0] != "method hudson.model.ItemGroup getItems" {
		t.Fatalf("unexpected approved signatures: %#v", approvedSignatures)
	}
}

func TestApprovePendingScriptApprovalItems_NoPendingItems(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/scriptApproval/":
			_, _ = w.Write([]byte(`
				<html><body>
				<script>
				var mgr = makeStaplerProxy('/$stapler/bound/token-2','proxy-crumb',['approveScript','approveSignature','aclApproveSignature']);
				</script>
				<p>No pending script approvals.</p>
				<p>No pending signature approvals.</p>
				</body></html>
			`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "user", "token", 5*time.Second)
	result, err := client.ApprovePendingScriptApprovalItems()
	if err != nil {
		t.Fatalf("ApprovePendingScriptApprovalItems returned error: %v", err)
	}
	if result.ApprovedCount() != 0 {
		t.Fatalf("expected approved count 0, got %#v", result)
	}
}
