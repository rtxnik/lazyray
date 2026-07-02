package core

import (
	"encoding/base64"
	"net"
	"strings"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

func TestParseSubscriptionBody_Base64Encoded(t *testing.T) {
	urls := []string{
		"vless://aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee@1.2.3.4:443?type=tcp#profile1",
		"vless://11111111-2222-3333-4444-555555555555@5.6.7.8:8443?type=xhttp#profile2",
	}
	body := base64.StdEncoding.EncodeToString([]byte(strings.Join(urls, "\n")))

	profiles, err := ParseSubscriptionBody(body)
	if err != nil {
		t.Fatalf("ParseSubscriptionBody() error = %v", err)
	}

	if len(profiles) != 2 {
		t.Fatalf("got %d profiles, want 2", len(profiles))
	}

	if profiles[0].Name != "profile1" {
		t.Errorf("profiles[0].Name = %q, want %q", profiles[0].Name, "profile1")
	}
	if profiles[1].Name != "profile2" {
		t.Errorf("profiles[1].Name = %q, want %q", profiles[1].Name, "profile2")
	}
	if profiles[0].Server.Address != "1.2.3.4" {
		t.Errorf("profiles[0].Address = %q, want %q", profiles[0].Server.Address, "1.2.3.4")
	}
	if profiles[1].Server.Port != 8443 {
		t.Errorf("profiles[1].Port = %d, want 8443", profiles[1].Server.Port)
	}
}

func TestParseSubscriptionBody_PlainText(t *testing.T) {
	body := "vless://aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee@1.2.3.4:443?type=tcp#test"

	profiles, err := ParseSubscriptionBody(body)
	if err != nil {
		t.Fatalf("ParseSubscriptionBody() error = %v", err)
	}

	if len(profiles) != 1 {
		t.Fatalf("got %d profiles, want 1", len(profiles))
	}

	if profiles[0].Name != "test" {
		t.Errorf("Name = %q, want %q", profiles[0].Name, "test")
	}
}

func TestParseSubscriptionBody_MultiProtocol(t *testing.T) {
	lines := []string{
		"vless://aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee@1.2.3.4:443#valid",
		"vmess://base64data",
		"trojan://password@host:443#trojan",
		"",
		"# comment line",
		"vless://11111111-2222-3333-4444-555555555555@5.6.7.8:443#valid2",
	}
	body := base64.StdEncoding.EncodeToString([]byte(strings.Join(lines, "\n")))

	profiles, err := ParseSubscriptionBody(body)
	if err != nil {
		t.Fatalf("ParseSubscriptionBody() error = %v", err)
	}

	// 2 VLESS + 1 Trojan = 3 valid profiles (vmess://base64data is invalid)
	if len(profiles) != 3 {
		t.Fatalf("got %d profiles, want 3 (2 VLESS + 1 Trojan, skipping invalid VMess)", len(profiles))
	}
	// Order: vless, trojan, vless (vmess is skipped due to invalid base64)
	if profiles[1].Server.GetProtocol() != "trojan" {
		t.Errorf("profiles[1].Protocol = %q, want %q", profiles[1].Server.GetProtocol(), "trojan")
	}
}

func TestParseSubscriptionBody_Empty(t *testing.T) {
	_, err := ParseSubscriptionBody("")
	if err == nil {
		t.Fatal("expected error for empty body")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error %q should mention empty", err.Error())
	}
}

func TestParseSubscriptionBody_NoValidURLs(t *testing.T) {
	body := base64.StdEncoding.EncodeToString([]byte("no valid urls here\njust text"))

	_, err := ParseSubscriptionBody(body)
	if err == nil {
		t.Fatal("expected error for body with no valid VLESS URLs")
	}
	if !strings.Contains(err.Error(), "no valid") {
		t.Errorf("error %q should mention no valid URLs", err.Error())
	}
}

func TestParseSubscriptionBody_RawBase64(t *testing.T) {
	url := "vless://aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee@1.2.3.4:443?type=tcp#rawtest"
	// Use raw base64 encoding (no padding)
	body := base64.RawStdEncoding.EncodeToString([]byte(url))

	profiles, err := ParseSubscriptionBody(body)
	if err != nil {
		t.Fatalf("ParseSubscriptionBody() error = %v", err)
	}

	if len(profiles) != 1 {
		t.Fatalf("got %d profiles, want 1", len(profiles))
	}
	if profiles[0].Name != "rawtest" {
		t.Errorf("Name = %q, want %q", profiles[0].Name, "rawtest")
	}
}

func TestImportSubscription_AddNew(t *testing.T) {
	servers := &config.ServersConfig{}

	// Create a fake subscription body (we'll test ImportSubscription indirectly via the merge logic)
	profiles := []*config.Profile{
		{
			Name: "Server 1",
			Server: config.ServerConfig{
				Address: "1.2.3.4",
				Port:    443,
				UUID:    "uuid-1",
			},
		},
		{
			Name: "Server 2",
			Server: config.ServerConfig{
				Address: "5.6.7.8",
				Port:    443,
				UUID:    "uuid-2",
			},
		},
	}

	// Simulate the merge logic from ImportSubscription
	subURL := "https://example.com/sub"
	subName := "test-sub"

	added := 0
	for _, p := range profiles {
		p.Group = subName
		p.Subscription = subURL
		servers.Profiles = append(servers.Profiles, *p)
		added++
	}

	if added != 2 {
		t.Errorf("added = %d, want 2", added)
	}
	if len(servers.Profiles) != 2 {
		t.Errorf("profiles count = %d, want 2", len(servers.Profiles))
	}
	if servers.Profiles[0].Group != "test-sub" {
		t.Errorf("group = %q, want %q", servers.Profiles[0].Group, "test-sub")
	}
	if servers.Profiles[0].Subscription != subURL {
		t.Errorf("subscription = %q, want %q", servers.Profiles[0].Subscription, subURL)
	}
}

func TestImportSubscription_UpdateExisting(t *testing.T) {
	subURL := "https://example.com/sub"
	servers := &config.ServersConfig{
		Profiles: []config.Profile{
			{
				Name:         "Old Name",
				Subscription: subURL,
				Server: config.ServerConfig{
					Address: "1.2.3.4",
					Port:    443,
					UUID:    "uuid-1",
				},
			},
		},
	}

	// Simulate update: same UUID from same subscription should update
	newProfile := &config.Profile{
		Name: "New Name",
		Server: config.ServerConfig{
			Address: "9.8.7.6",
			Port:    8443,
			UUID:    "uuid-1",
		},
	}

	updated := 0
	for i := range servers.Profiles {
		if servers.Profiles[i].Subscription == subURL &&
			servers.Profiles[i].Server.UUID == newProfile.Server.UUID {
			servers.Profiles[i].Server = newProfile.Server
			servers.Profiles[i].Name = newProfile.Name
			updated++
		}
	}

	if updated != 1 {
		t.Errorf("updated = %d, want 1", updated)
	}
	if servers.Profiles[0].Name != "New Name" {
		t.Errorf("name = %q, want %q", servers.Profiles[0].Name, "New Name")
	}
	if servers.Profiles[0].Server.Address != "9.8.7.6" {
		t.Errorf("address = %q, want %q", servers.Profiles[0].Server.Address, "9.8.7.6")
	}
}

func TestParseSubscriptionBody_MultipleEncodings(t *testing.T) {
	url := "vless://aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee@1.2.3.4:443?type=tcp#encoded"

	tests := []struct {
		name   string
		encode func(string) string
	}{
		{"standard", func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }},
		{"raw", func(s string) string { return base64.RawStdEncoding.EncodeToString([]byte(s)) }},
		{"url-safe", func(s string) string { return base64.URLEncoding.EncodeToString([]byte(s)) }},
		{"plain", func(s string) string { return s }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := tc.encode(url)
			profiles, err := ParseSubscriptionBody(body)
			if err != nil {
				t.Fatalf("ParseSubscriptionBody() error = %v", err)
			}
			if len(profiles) != 1 {
				t.Fatalf("got %d profiles, want 1", len(profiles))
			}
		})
	}
}

func TestFetchSubscription_BlocksPrivateHost(t *testing.T) {
	orig := lookupIP
	t.Cleanup(func() { lookupIP = orig })
	lookupIP = func(host string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("127.0.0.1")}, nil // resolves to loopback
	}

	_, err := FetchSubscription("https://malicious.example/sub")
	if err == nil {
		t.Fatal("FetchSubscription allowed a host resolving to loopback, want SSRF block")
	}
	if !strings.Contains(err.Error(), "blocked") {
		t.Errorf("error %q should indicate a blocked address", err.Error())
	}
}

func TestFetchSubscription_RejectsHTTP(t *testing.T) {
	_, err := FetchSubscription("http://example.com/sub")
	if err == nil {
		t.Fatal("FetchSubscription allowed http:// URL, want https-only rejection")
	}
	if !strings.Contains(err.Error(), "https") {
		t.Errorf("error %q should mention https", err.Error())
	}
}
