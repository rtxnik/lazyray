package app

import (
	"fmt"

	"github.com/rtxnik/lazyray/internal/config"
)

// DuplicateUUIDError reports an import blocked because a profile with the same
// UUID already exists. Its message matches the TUI's wording exactly; the CLI
// reformats it with the "--force" hint.
type DuplicateUUIDError struct{ ExistingName string }

func (e *DuplicateUUIDError) Error() string {
	return fmt.Sprintf("UUID already used by profile %q", e.ExistingName)
}

// ImportProfile persists an already-parsed profile into servers: reject a
// duplicate UUID unless force, flag the first-ever profile as default, append,
// and save. It mutates servers in place so a stateful caller (the TUI) stays in
// sync, and returns the raw save error unwrapped so each shell keeps its own
// wrapping.
func (s *Service) ImportProfile(servers *config.ServersConfig, p *config.Profile, force bool) (*config.Profile, error) {
	if existingName, exists := servers.HasUUID(p.Server.UUID); exists && !force {
		return nil, &DuplicateUUIDError{ExistingName: existingName}
	}
	if len(servers.Profiles) == 0 {
		p.Default = true
	}
	servers.Profiles = append(servers.Profiles, *p)
	if err := s.saveServers(servers); err != nil {
		return nil, err
	}
	return p, nil
}

// ImportSubscription fetches and merges a subscription into servers (via the
// core seam) and applies the default-if-first guard. Persisting servers and
// upserting the subscription settings entry stay with the caller because their
// error strictness differs per shell. servers is mutated in place; the raw
// fetch/merge error is returned unwrapped for the caller to wrap.
func (s *Service) ImportSubscription(servers *config.ServersConfig, subURL, subName string) (added, updated int, err error) {
	added, updated, err = s.importSubscription(subURL, subName, servers)
	if err != nil {
		return 0, 0, err
	}
	// Preserved verbatim from both shells, including its latent no-op condition:
	// DefaultProfile() returns nil only when there are zero profiles, so after a
	// merge this never flags a Default. Do NOT simplify — behaviour identical.
	if servers.DefaultProfile() == nil && len(servers.Profiles) > 0 {
		servers.Profiles[0].Default = true
	}
	return added, updated, nil
}
