package config

import "testing"

// TestProfile_Clone_NoAliasing verifies that Clone produces a deep copy whose
// slice-backed fields share no storage with the original. Mutating the clone in
// place must never reach back and corrupt the source profile.
func TestProfile_Clone_NoAliasing(t *testing.T) {
	source := Profile{
		Name: "src",
		Chain: []ServerConfig{
			{Address: "chain-orig"},
		},
		Tags: []string{"tag-orig", "tag2"},
		Routing: ProfileRouting{
			Bypass: []string{"bypass-orig"},
			Block:  []string{"block-orig"},
			DNSRules: []DNSRule{
				{
					Server:    "1.1.1.1",
					Domains:   []string{"domain-orig"},
					ExpectIPs: []string{"ip-orig"},
				},
			},
		},
	}

	clone := source.Clone()

	clone.Chain[0].Address = "MUT"
	clone.Tags[0] = "MUT"
	clone.Tags = append(clone.Tags, "x")
	clone.Routing.Bypass[0] = "MUT"
	clone.Routing.Block[0] = "MUT"
	clone.Routing.DNSRules[0].Domains[0] = "MUT"
	clone.Routing.DNSRules[0].ExpectIPs = append(clone.Routing.DNSRules[0].ExpectIPs, "y")

	if got := source.Chain[0].Address; got != "chain-orig" {
		t.Errorf("source.Chain[0].Address mutated via clone: got %q, want %q", got, "chain-orig")
	}
	if got := source.Tags[0]; got != "tag-orig" {
		t.Errorf("source.Tags[0] mutated via clone: got %q, want %q", got, "tag-orig")
	}
	if got := len(source.Tags); got != 2 {
		t.Errorf("source.Tags length changed via clone append: got %d, want %d", got, 2)
	}
	if got := source.Routing.Bypass[0]; got != "bypass-orig" {
		t.Errorf("source.Routing.Bypass[0] mutated via clone: got %q, want %q", got, "bypass-orig")
	}
	if got := source.Routing.Block[0]; got != "block-orig" {
		t.Errorf("source.Routing.Block[0] mutated via clone: got %q, want %q", got, "block-orig")
	}
	if got := source.Routing.DNSRules[0].Domains[0]; got != "domain-orig" {
		t.Errorf("source.Routing.DNSRules[0].Domains[0] mutated via clone: got %q, want %q", got, "domain-orig")
	}
	if got := len(source.Routing.DNSRules[0].ExpectIPs); got != 1 {
		t.Errorf("source.Routing.DNSRules[0].ExpectIPs length changed via clone append: got %d, want %d", got, 1)
	}
	if got := source.Routing.DNSRules[0].ExpectIPs[0]; got != "ip-orig" {
		t.Errorf("source.Routing.DNSRules[0].ExpectIPs[0] mutated via clone: got %q, want %q", got, "ip-orig")
	}
}

// TestProfile_Clone_PreservesNil verifies that cloning a zero-value Profile
// keeps all slice-backed fields nil, so the YAML round-trip (omitempty) is
// unchanged and nil never becomes an empty non-nil slice.
func TestProfile_Clone_PreservesNil(t *testing.T) {
	var source Profile

	clone := source.Clone()

	if clone.Chain != nil {
		t.Errorf("clone.Chain: got %v, want nil", clone.Chain)
	}
	if clone.Tags != nil {
		t.Errorf("clone.Tags: got %v, want nil", clone.Tags)
	}
	if clone.Routing.Bypass != nil {
		t.Errorf("clone.Routing.Bypass: got %v, want nil", clone.Routing.Bypass)
	}
	if clone.Routing.Block != nil {
		t.Errorf("clone.Routing.Block: got %v, want nil", clone.Routing.Block)
	}
	if clone.Routing.DNSRules != nil {
		t.Errorf("clone.Routing.DNSRules: got %v, want nil", clone.Routing.DNSRules)
	}
}
