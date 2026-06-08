package security

import "testing"

func TestBuildScanRunUpdatesMapsTaskFields(t *testing.T) {
	updates := buildScanRunUpdates(map[string]interface{}{
		"status":      "running",
		"progress":    30,
		"total_ips":   12,
		"scanned_ips": 3,
		"message":     "scanning",
		"high_risk":   1,
	})

	if updates["status"] != "running" {
		t.Fatalf("expected status to be copied, got %#v", updates["status"])
	}
	if updates["progress"] != 30 {
		t.Fatalf("expected progress to be copied, got %#v", updates["progress"])
	}
	if updates["total_targets"] != 12 {
		t.Fatalf("expected total_ips to map to total_targets, got %#v", updates["total_targets"])
	}
	if updates["scanned_targets"] != 3 {
		t.Fatalf("expected scanned_ips to map to scanned_targets, got %#v", updates["scanned_targets"])
	}
	if updates["high_risk"] != 1 {
		t.Fatalf("expected high_risk to be copied, got %#v", updates["high_risk"])
	}
}

func TestIsTerminalTaskStatus(t *testing.T) {
	for _, status := range []string{"completed", "failed", "cancelled"} {
		if !isTerminalTaskStatus(status) {
			t.Fatalf("expected %s to be terminal", status)
		}
	}

	for _, status := range []string{"pending", "running", "paused"} {
		if isTerminalTaskStatus(status) {
			t.Fatalf("expected %s to be non-terminal", status)
		}
	}
}
