package security

import "testing"

func TestWebRuleEnabledForStandardScan(t *testing.T) {
	if !webRuleEnabled(nil, "information-disclosure") {
		t.Fatal("expected standard scan to enable information-disclosure web-rule")
	}
	if !webRuleEnabled(nil, "broken-access", "broken-access-control") {
		t.Fatal("expected standard scan to enable broken-access web-rule")
	}
}

func TestWebRuleEnabledRespectsCustomCategories(t *testing.T) {
	if !webRuleEnabled([]string{"information-disclosure"}, "information-disclosure") {
		t.Fatal("expected information-disclosure option to enable matching web-rule")
	}
	if webRuleEnabled([]string{"information-disclosure"}, "broken-access", "broken-access-control") {
		t.Fatal("expected information-disclosure option to exclude broken-access web-rule")
	}
	if !webRuleEnabled([]string{"broken-access"}, "broken-access-control") {
		t.Fatal("expected broken-access option to enable canonical broken-access-control web-rule")
	}
	if webRuleEnabled([]string{"xss"}, "information-disclosure") {
		t.Fatal("expected unrelated option to exclude information-disclosure web-rule")
	}
}

