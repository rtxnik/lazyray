package panels

import (
	"strings"
	"testing"
)

func TestProfilesEmptyKeystoneCard(t *testing.T) {
	p := NewProfilesPanel()
	p.Width = 40
	p.ImportKey = "i"
	p.SubsKey = "S"
	out := p.View()
	for _, want := range []string{"Profiles are saved connections", "[i]", "[S]"} {
		if !strings.Contains(out, want) {
			t.Errorf("empty profiles card missing %q\n%s", want, out)
		}
	}
}

func TestProfilesEmptyCardCitesRebind(t *testing.T) {
	p := NewProfilesPanel()
	p.Width = 40
	p.ImportKey = "x" // rebound import key
	p.SubsKey = "S"
	if out := p.View(); !strings.Contains(out, "[x]") {
		t.Errorf("empty card must cite the rebound import key [x]\n%s", out)
	}
}
