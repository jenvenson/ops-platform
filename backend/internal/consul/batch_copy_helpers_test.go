package consul

import (
	"reflect"
	"testing"
)

func TestCollectProjectNames(t *testing.T) {
	keys := []string{
		"plugin/app-b/V2.5.1",
		"plugin/app-a/V2.5.1/config.yaml",
		"plugin/app-a/V2.5.1/other.yaml",
		"plugin/app-c/",
		"",
	}

	got := collectProjectNames(keys)
	want := []string{"app-a", "app-b", "app-c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("collectProjectNames() = %#v, want %#v", got, want)
	}
}

func TestFilterCopySourceKeys(t *testing.T) {
	sourcePrefix := "plugin/app-a/V2.5.1"
	keys := []string{
		"",
		sourcePrefix,
		sourcePrefix + "/",
		sourcePrefix + "/config.yaml",
		sourcePrefix + "/nested/item.yaml",
	}

	got := filterCopySourceKeys(keys, sourcePrefix)
	want := []string{
		sourcePrefix,
		sourcePrefix + "/config.yaml",
		sourcePrefix + "/nested/item.yaml",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("filterCopySourceKeys() = %#v, want %#v", got, want)
	}
}
