package security

import (
	"testing"

	"github.com/edy/ops-platform/internal/models"
)

func TestDiscoverServiceTargetsSeparatesWebAndHostServices(t *testing.T) {
	webTargets, hostTargets := discoverServiceTargets("10.0.0.8", "host-vuln", []NmapPort{
		{PortID: 80, Service: "http", Protocol: "tcp"},
		{PortID: 22, Service: "ssh", Protocol: "tcp"},
		{PortID: 3306, Service: "mysql", Protocol: "tcp"},
		{PortID: 9999, Service: "unknown", Protocol: "tcp"},
	})

	if len(webTargets) != 1 || webTargets[0] != "http://10.0.0.8:80" {
		t.Fatalf("unexpected web targets: %#v", webTargets)
	}

	if len(hostTargets) != 2 {
		t.Fatalf("expected 2 host targets, got %#v", hostTargets)
	}

	if hostTargets[0].URL != "10.0.0.8:22" || hostTargets[0].Service != "ssh" {
		t.Fatalf("unexpected first host target: %#v", hostTargets[0])
	}

	if hostTargets[1].URL != "10.0.0.8:3306" || hostTargets[1].Service != "mysql" {
		t.Fatalf("unexpected second host target: %#v", hostTargets[1])
	}
}

func TestCountsTowardTaskRisk(t *testing.T) {
	tests := []struct {
		name     string
		vuln     models.SecurityVulnerability
		expected bool
	}{
		{
			name:     "inventory finding excluded",
			vuln:     models.SecurityVulnerability{FindingFamily: "inventory", FindingSource: "asset-inventory"},
			expected: false,
		},
		{
			name:     "host version match excluded",
			vuln:     models.SecurityVulnerability{Scanner: "vuln-matcher", Confidence: "high", MatchMode: "version-range"},
			expected: false,
		},
		{
			name:     "verified host template counted",
			vuln:     models.SecurityVulnerability{Scanner: "nuclei", FindingSource: "host-template", Severity: "high"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := countsTowardTaskRisk(tt.vuln); got != tt.expected {
				t.Fatalf("countsTowardTaskRisk() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetServiceTagsIncludesExpandedHighPriorityServices(t *testing.T) {
	tests := []struct {
		service string
		want    []string
	}{
		{service: "oracle", want: []string{"oracle", "database"}},
		{service: "memcached", want: []string{"memcached", "database"}},
		{service: "docker", want: []string{"docker", "network"}},
		{service: "kube-apiserver", want: []string{"kubernetes", "network"}},
		{service: "etcd", want: []string{"etcd", "database"}},
		{service: "consul", want: []string{"consul", "network"}},
		{service: "zookeeper", want: []string{"zookeeper", "network"}},
		{service: "kafka", want: []string{"kafka", "network"}},
		{service: "rabbitmq", want: []string{"rabbitmq", "network"}},
		{service: "amqp", want: []string{"rabbitmq", "network"}},
	}

	for _, tt := range tests {
		t.Run(tt.service, func(t *testing.T) {
			got := GetServiceTags(tt.service)
			if len(got) != len(tt.want) {
				t.Fatalf("GetServiceTags(%q) len=%d, want %d (%v)", tt.service, len(got), len(tt.want), got)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("GetServiceTags(%q)[%d]=%q, want %q", tt.service, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestDetectServiceIncludesExpandedFallbackPorts(t *testing.T) {
	tests := []struct {
		port int
		want string
	}{
		{port: 1521, want: "oracle"},
		{port: 2181, want: "zookeeper"},
		{port: 2375, want: "docker"},
		{port: 2379, want: "etcd"},
		{port: 5672, want: "amqp"},
		{port: 6443, want: "kube"},
		{port: 8500, want: "consul"},
		{port: 9092, want: "kafka"},
		{port: 11211, want: "memcached"},
		{port: 15672, want: "rabbitmq"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := detectService(tt.port); got != tt.want {
				t.Fatalf("detectService(%d) = %q, want %q", tt.port, got, tt.want)
			}
		})
	}
}

func TestIsUsefulInventoryHostResultSupportsExpandedServices(t *testing.T) {
	result := NucleiResult{
		TemplateID: "docker-version-detect",
		Severity:   "info",
		Info: NucleiInfo{
			Name: "Docker Version Enumeration",
			Tags: []string{"docker", "version", "enumeration"},
		},
	}

	if !isUsefulInventoryHostResult("docker", result) {
		t.Fatalf("expected docker info enumeration result to be treated as inventory")
	}
	if !isUsefulInventoryHostResult("etcd", NucleiResult{
		TemplateID: "etcd-fingerprint",
		Severity:   "info",
		Info: NucleiInfo{
			Name: "Etcd Fingerprint Detection",
			Tags: []string{"etcd", "fingerprint"},
		},
	}) {
		t.Fatalf("expected etcd fingerprint result to be treated as inventory")
	}
	if isUsefulInventoryHostResult("kafka", NucleiResult{
		TemplateID: "kafka-unauthorized-access",
		Severity:   "medium",
		Info: NucleiInfo{
			Name: "Kafka Unauthorized Access",
			Tags: []string{"kafka", "unauth"},
		},
	}) {
		t.Fatalf("unexpected actionable kafka result to be treated as inventory")
	}
}
