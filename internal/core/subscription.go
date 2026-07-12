package core

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rtxnik/lazyray/internal/config"
)

// FetchSubscription downloads a subscription URL and parses VLESS profiles from it.
// The subscription body is expected to be base64-encoded, containing one VLESS URL per line.
func FetchSubscription(subURL string) ([]*config.Profile, error) {
	client := directClient(15 * time.Second)
	resp, err := safeGet(context.Background(), client, subURL, 1<<20)
	if err != nil {
		return nil, fmt.Errorf("fetching subscription: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("subscription returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body) // body already capped at 1 MB by safeGet
	if err != nil {
		return nil, fmt.Errorf("reading subscription body: %w", err)
	}

	return ParseSubscriptionBody(string(body))
}

// ParseSubscriptionBody decodes a base64-encoded subscription body and parses proxy URLs.
// Supports VLESS, VMess, and Trojan URLs.
func ParseSubscriptionBody(body string) ([]*config.Profile, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, fmt.Errorf("empty subscription body")
	}

	// Decode base64 (standard or URL-safe, padded or raw). Genuine plaintext
	// bodies always fail every decoder — proxy URLs contain ':', which no
	// base64 alphabet includes — and fall through unchanged.
	decoded, err := decodeBase64Any(body)
	if err != nil {
		decoded = []byte(body)
	}

	lines := strings.Split(strings.TrimSpace(string(decoded)), "\n")
	var profiles []*config.Profile

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Support all protocol URLs
		profile, err := ParseProxyURL(line)
		if err != nil {
			continue
		}
		if err := ValidateProfile(profile); err != nil {
			continue
		}
		profiles = append(profiles, profile)
	}

	if len(profiles) == 0 {
		return nil, fmt.Errorf("no valid proxy URLs found in subscription")
	}

	return profiles, nil
}

// ImportSubscription fetches a subscription URL and merges profiles into servers config.
// Existing profiles from the same subscription are updated by UUID match; new ones are added.
// Returns the number of added and updated profiles.
func ImportSubscription(subURL, subName string, servers *config.ServersConfig) (added, updated int, err error) {
	profiles, err := FetchSubscription(subURL)
	if err != nil {
		return 0, 0, err
	}

	groupName := subName
	if groupName == "" {
		groupName = "subscription"
	}

	for _, newProfile := range profiles {
		newProfile.Group = groupName
		newProfile.Subscription = subURL

		// Try to find existing profile from same subscription by UUID
		found := false
		for i := range servers.Profiles {
			if servers.Profiles[i].Subscription == subURL &&
				servers.Profiles[i].Server.UUID == newProfile.Server.UUID {
				// Update existing profile, preserve local settings
				servers.Profiles[i].Server = newProfile.Server
				servers.Profiles[i].Name = newProfile.Name
				updated++
				found = true
				break
			}
		}

		if !found {
			servers.Profiles = append(servers.Profiles, *newProfile)
			added++
		}
	}

	return added, updated, nil
}
