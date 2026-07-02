package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSubscriptionEntry_YAML(t *testing.T) {
	entry := SubscriptionEntry{
		Name:     "Test Sub",
		URL:      "https://example.com/sub",
		Interval: 24,
	}

	data, err := yaml.Marshal(entry)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var parsed SubscriptionEntry
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if parsed.Name != "Test Sub" {
		t.Errorf("Name = %q, want %q", parsed.Name, "Test Sub")
	}
	if parsed.URL != "https://example.com/sub" {
		t.Errorf("URL = %q, want expected", parsed.URL)
	}
	if parsed.Interval != 24 {
		t.Errorf("Interval = %d, want 24", parsed.Interval)
	}
}

func TestSettings_WithSubscriptions_YAML(t *testing.T) {
	settings := DefaultSettings()
	settings.Subscriptions = []SubscriptionEntry{
		{Name: "Sub 1", URL: "https://example.com/sub1", Interval: 24},
		{Name: "Sub 2", URL: "https://example.com/sub2", Interval: 12},
	}

	data, err := yaml.Marshal(settings)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var parsed Settings
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if len(parsed.Subscriptions) != 2 {
		t.Fatalf("Subscriptions count = %d, want 2", len(parsed.Subscriptions))
	}
	if parsed.Subscriptions[0].Name != "Sub 1" {
		t.Errorf("Sub[0].Name = %q, want %q", parsed.Subscriptions[0].Name, "Sub 1")
	}
	if parsed.Subscriptions[1].URL != "https://example.com/sub2" {
		t.Errorf("Sub[1].URL = %q, want expected", parsed.Subscriptions[1].URL)
	}
}

func TestDefaultSettings_NoSubscriptions(t *testing.T) {
	s := DefaultSettings()
	if len(s.Subscriptions) != 0 {
		t.Errorf("Default settings should have 0 subscriptions, got %d", len(s.Subscriptions))
	}
}

func TestDefaultSettings_Notifications(t *testing.T) {
	s := DefaultSettings()
	if !s.Notifications.Enabled {
		t.Error("Notifications should be enabled by default")
	}
}

func TestProfile_GroupAndTags_YAML(t *testing.T) {
	profile := Profile{
		Name:  "Test",
		Group: "us-servers",
		Tags:  []string{"fast", "stable"},
		Server: ServerConfig{
			Address: "1.2.3.4",
			Port:    443,
			UUID:    "test-uuid",
		},
		Latency:      45,
		Subscription: "https://example.com/sub",
	}

	data, err := yaml.Marshal(profile)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var parsed Profile
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if parsed.Group != "us-servers" {
		t.Errorf("Group = %q, want %q", parsed.Group, "us-servers")
	}
	if len(parsed.Tags) != 2 {
		t.Fatalf("Tags count = %d, want 2", len(parsed.Tags))
	}
	if parsed.Tags[0] != "fast" {
		t.Errorf("Tags[0] = %q, want %q", parsed.Tags[0], "fast")
	}
	if parsed.Latency != 45 {
		t.Errorf("Latency = %d, want 45", parsed.Latency)
	}
	if parsed.Subscription != "https://example.com/sub" {
		t.Errorf("Subscription = %q, want expected", parsed.Subscription)
	}
}

func TestProfile_EmptyGroupAndTags_OmittedInYAML(t *testing.T) {
	profile := Profile{
		Name: "Test",
		Server: ServerConfig{
			Address: "1.2.3.4",
			Port:    443,
			UUID:    "uuid",
		},
	}

	data, err := yaml.Marshal(profile)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	yamlStr := string(data)
	if containsField(yamlStr, "group:") {
		t.Error("empty group should be omitted in YAML")
	}
	if containsField(yamlStr, "tags:") {
		t.Error("empty tags should be omitted in YAML")
	}
	if containsField(yamlStr, "latency:") {
		t.Error("zero latency should be omitted in YAML")
	}
}

func containsField(yamlStr, field string) bool {
	for _, line := range splitLines(yamlStr) {
		if len(line) > 0 && line[0] != ' ' && len(line) >= len(field) {
			if line[:len(field)] == field {
				return true
			}
		}
	}
	return false
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func TestUpsertSubscription_AppendsNewWithDefaultInterval(t *testing.T) {
	s := &Settings{}
	s.UpsertSubscription("https://x/sub", "primary")
	if len(s.Subscriptions) != 1 {
		t.Fatalf("len = %d, want 1", len(s.Subscriptions))
	}
	got := s.Subscriptions[0]
	if got.URL != "https://x/sub" || got.Name != "primary" || got.Interval != 24 {
		t.Errorf("entry = %+v, want {primary https://x/sub 24}", got)
	}
}

func TestUpsertSubscription_UpdatesNameByURLOnly(t *testing.T) {
	s := &Settings{Subscriptions: []SubscriptionEntry{
		{Name: "old", URL: "https://x/sub", Interval: 6},
	}}
	s.UpsertSubscription("https://x/sub", "renamed")
	if len(s.Subscriptions) != 1 {
		t.Fatalf("upsert must not append when URL matches; len = %d", len(s.Subscriptions))
	}
	if s.Subscriptions[0].Name != "renamed" {
		t.Errorf("Name = %q, want renamed", s.Subscriptions[0].Name)
	}
	if s.Subscriptions[0].Interval != 6 {
		t.Errorf("Interval = %d, want 6 (existing interval preserved)", s.Subscriptions[0].Interval)
	}
}
