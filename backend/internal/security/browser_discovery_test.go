package security

import (
	"reflect"
	"testing"
)

func TestBrowserHelperURLsRequiresExplicitSetting(t *testing.T) {
	if got := browserHelperURLs(""); got != nil {
		t.Fatalf("expected nil urls when setting is empty, got %#v", got)
	}
}

func TestBrowserHelperURLsParsesExplicitURLs(t *testing.T) {
	got := browserHelperURLs(" http://helper-a:31730/discover, ,http://helper-b:31730/discover ")
	want := []string{
		"http://helper-a:31730/discover",
		"http://helper-b:31730/discover",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("browserHelperURLs() = %#v, want %#v", got, want)
	}
}
