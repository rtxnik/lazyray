package commands

import "testing"

func TestBaseKeyMapBuildsFullRegistry(t *testing.T) {
	reg := New(BaseKeyMap())
	if len(reg.All()) == 0 {
		t.Fatal("registry is empty")
	}
	for _, c := range reg.All() {
		if KeyDisplay(c.Binding) == "" {
			t.Errorf("command %q has no key binding", c.ID)
		}
	}
}

func TestScopeString(t *testing.T) {
	cases := map[Scope]string{
		ScopeGlobal:   "Global",
		ScopeProfiles: "Profiles",
		ScopeLogs:     "Logs",
		ScopeStatus:   "Status",
	}
	for s, want := range cases {
		if got := s.String(); got != want {
			t.Errorf("Scope(%d).String() = %q, want %q", int(s), got, want)
		}
	}
}
