package security

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestParseFIMCollectOutput(t *testing.T) {
	output := strings.Join([]string{
		"F\t/etc/app/config.yaml\t128\t1711920000\tabc123",
		"F\t/etc/app/secret.env\t64\t1711920300\tdef456",
	}, "\n")

	items, err := parseFIMCollectOutput(output)
	if err != nil {
		t.Fatalf("parseFIMCollectOutput returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Path != "/etc/app/config.yaml" {
		t.Fatalf("unexpected first path: %s", items[0].Path)
	}
	if items[1].SHA256 != "def456" {
		t.Fatalf("unexpected second sha: %s", items[1].SHA256)
	}
}

func TestParseFIMCollectOutputPresenceOnly(t *testing.T) {
	output := strings.Join([]string{
		"P\t/var/opt/pkg/a.tar",
		"P\t/var/opt/pkg/b.tar",
	}, "\n")

	items, err := parseFIMCollectOutput(output)
	if err != nil {
		t.Fatalf("parseFIMCollectOutput returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].EntryType != "presence" {
		t.Fatalf("unexpected first entry type: %s", items[0].EntryType)
	}
	if items[0].Path != "/var/opt/pkg/a.tar" {
		t.Fatalf("unexpected first path: %s", items[0].Path)
	}
	if items[1].SHA256 != "" {
		t.Fatalf("expected empty sha for presence entry, got %s", items[1].SHA256)
	}
}

func TestCompareFIMEntries(t *testing.T) {
	baselineTime := time.Unix(1711920000, 0)
	currentTime := time.Unix(1711923600, 0)

	baseline := map[string]FIMSnapshotEntry{
		"/etc/app/old.conf": {
			Path:   "/etc/app/old.conf",
			Size:   10,
			SHA256: "old",
			Mtime:  &baselineTime,
		},
		"/etc/app/change.conf": {
			Path:   "/etc/app/change.conf",
			Size:   20,
			SHA256: "before",
			Mtime:  &baselineTime,
		},
	}
	current := map[string]FIMSnapshotEntry{
		"/etc/app/change.conf": {
			Path:   "/etc/app/change.conf",
			Size:   30,
			SHA256: "after",
			Mtime:  &currentTime,
		},
		"/etc/app/new.conf": {
			Path:   "/etc/app/new.conf",
			Size:   12,
			SHA256: "new",
			Mtime:  &currentTime,
		},
	}

	diffs := compareFIMEntries(baseline, current, "warning")
	if len(diffs) != 2 {
		t.Fatalf("expected 2 diffs, got %d", len(diffs))
	}

	kinds := map[string]string{}
	for _, diff := range diffs {
		kinds[diff.Path] = diff.EventType
	}

	if kinds["/etc/app/change.conf"] != "modify" {
		t.Fatalf("expected modify diff for change.conf, got %s", kinds["/etc/app/change.conf"])
	}
	if kinds["/etc/app/old.conf"] != "delete" {
		t.Fatalf("expected delete diff for old.conf, got %s", kinds["/etc/app/old.conf"])
	}
	if _, exists := kinds["/etc/app/new.conf"]; exists {
		t.Fatalf("did not expect create diff for new.conf, got %s", kinds["/etc/app/new.conf"])
	}
}

func TestCompareFIMEntriesPresenceOnly(t *testing.T) {
	baseline := map[string]FIMSnapshotEntry{
		"/var/opt/pkg/old.tar": {
			Path:      "/var/opt/pkg/old.tar",
			EntryType: "presence",
		},
		"/var/opt/pkg/keep.tar": {
			Path:      "/var/opt/pkg/keep.tar",
			EntryType: "presence",
		},
	}
	current := map[string]FIMSnapshotEntry{
		"/var/opt/pkg/keep.tar": {
			Path:      "/var/opt/pkg/keep.tar",
			EntryType: "presence",
		},
		"/var/opt/pkg/new.tar": {
			Path:      "/var/opt/pkg/new.tar",
			EntryType: "presence",
		},
	}

	diffs := compareFIMEntries(baseline, current, "warning")
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].Path != "/var/opt/pkg/old.tar" || diffs[0].EventType != "delete" {
		t.Fatalf("unexpected diff: %#v", diffs[0])
	}
}

func TestParseNotifyChannelIDs(t *testing.T) {
	ids := parseNotifyChannelIDs("3, 1, 3, invalid, 0, 2")
	if len(ids) != 3 {
		t.Fatalf("expected 3 ids, got %d", len(ids))
	}
	if ids[0] != 3 || ids[1] != 1 || ids[2] != 2 {
		t.Fatalf("unexpected ids: %#v", ids)
	}
}

func TestNormalizeNotifyChannels(t *testing.T) {
	value := normalizeNotifyChannels("5,2, 5, nope")
	if value != "5,2" {
		t.Fatalf("unexpected normalized value: %s", value)
	}
}

func TestShouldRunFIMScheduledScan(t *testing.T) {
	now := time.Unix(1711923600, 0)
	last := now.Add(-10 * time.Minute)
	recent := now.Add(-30 * time.Second)

	if !shouldRunFIMScheduledScan(now, nil, 300) {
		t.Fatalf("expected nil last scan to be due")
	}
	if !shouldRunFIMScheduledScan(now, &last, 300) {
		t.Fatalf("expected overdue target to be due")
	}
	if shouldRunFIMScheduledScan(now, &recent, 300) {
		t.Fatalf("did not expect recent target to be due")
	}
}

func TestBuildFIMAlertDedupKey(t *testing.T) {
	key := buildFIMAlertDedupKey(" /tmp/fim-test/base.conf ", " delete ")
	if key != "/tmp/fim-test/base.conf::delete" {
		t.Fatalf("unexpected dedup key: %s", key)
	}
}

func TestBuildFIMExecutionLockKey(t *testing.T) {
	key := buildFIMExecutionLockKey(12, 34)
	if key != "12:34" {
		t.Fatalf("unexpected execution lock key: %s", key)
	}
}

func TestNormalizeFIMSnapshotType(t *testing.T) {
	cases := map[string]string{
		"":           "manual",
		"manual":     "manual",
		" baseline ": "baseline",
		"scheduled":  "scheduled",
		"unexpected": "manual",
	}
	for input, expected := range cases {
		if actual := normalizeFIMSnapshotType(input); actual != expected {
			t.Fatalf("normalizeFIMSnapshotType(%q) = %q, want %q", input, actual, expected)
		}
	}
}

func TestNormalizeFIMWatchPathScanMode(t *testing.T) {
	cases := map[string]string{
		"":               "full_hash",
		"full_hash":      "full_hash",
		"presence_only":  "presence_only",
		" other ":        "full_hash",
	}
	for input, expected := range cases {
		if actual := normalizeFIMWatchPathScanMode(input); actual != expected {
			t.Fatalf("normalizeFIMWatchPathScanMode(%q) = %q, want %q", input, actual, expected)
		}
	}
}

func TestUniqueStrings(t *testing.T) {
	values := uniqueStrings([]string{" open ", "acknowledged", "open", "", " acknowledged "})
	if len(values) != 2 {
		t.Fatalf("expected 2 values, got %d", len(values))
	}
	if values[0] != "open" || values[1] != "acknowledged" {
		t.Fatalf("unexpected values: %#v", values)
	}
}

func TestAcquireFIMExecutionLock(t *testing.T) {
	lockKey := fmt.Sprintf("test-lock-%d", time.Now().UnixNano())
	release, acquired := acquireFIMExecutionLock(lockKey)
	if !acquired {
		t.Fatalf("expected first lock acquire to succeed")
	}

	_, acquiredAgain := acquireFIMExecutionLock(lockKey)
	if acquiredAgain {
		t.Fatalf("expected second lock acquire to fail while held")
	}

	release()

	releaseAfterUnlock, acquiredAfterUnlock := acquireFIMExecutionLock(lockKey)
	if !acquiredAfterUnlock {
		t.Fatalf("expected lock acquire to succeed after release")
	}
	releaseAfterUnlock()
}

func TestBuildFIMAlertReuseUpdates(t *testing.T) {
	now := time.Unix(1711923600, 0)
	existing := FIMAlert{
		ID:              11,
		OccurrenceCount: 3,
	}
	diff := FIMDiffEvent{
		ID:         22,
		Path:       "/tmp/fim-test/base.conf",
		EventType:  "delete",
		Severity:   "high",
		PolicyID:   2,
		PolicyName: "tmp目录监测",
		ServerID:   1,
		ServerName: "cluser-zb-control",
		ServerIP:   "192.0.2.1",
	}

	updates := buildFIMAlertReuseUpdates(existing, diff, now)
	if updates["diff_event_id"] != uint(22) {
		t.Fatalf("unexpected diff_event_id: %#v", updates["diff_event_id"])
	}
	if updates["occurrence_count"] != 4 {
		t.Fatalf("expected occurrence_count 4, got %#v", updates["occurrence_count"])
	}
	if updates["last_seen_at"] != now {
		t.Fatalf("unexpected last_seen_at: %#v", updates["last_seen_at"])
	}
	title, _ := updates["title"].(string)
	if !strings.Contains(title, "/tmp/fim-test/base.conf") {
		t.Fatalf("unexpected title: %s", title)
	}
	summary, _ := updates["summary"].(string)
	if !strings.Contains(summary, "文件删除") {
		t.Fatalf("unexpected summary: %s", summary)
	}
}
