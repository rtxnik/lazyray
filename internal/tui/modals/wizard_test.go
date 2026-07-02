package modals

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/tui/theme"
)

// wzKey builds a KeyMsg for the wizard tests (no shared helper exists in this package).
func wzKey(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func TestWizard_MethodMenuRoutesToURL(t *testing.T) {
	m := NewWizardModal(120, 40)
	m.Update(wzKey("enter")) // default selection = Paste URL
	if m.step != WizardStepURL {
		t.Fatalf("enter on default should route to WizardStepURL, got step %d", m.step)
	}
}

func TestWizard_MethodMenuRoutesToSubscription(t *testing.T) {
	m := NewWizardModal(120, 40)
	m.Update(wzKey("down")) // move to "Import a subscription"
	m.Update(wzKey("enter"))
	if m.step != WizardStepSubURL {
		t.Fatalf("down+enter should route to WizardStepSubURL, got step %d", m.step)
	}
}

func TestWizard_MethodMenuNumberShortcut(t *testing.T) {
	m := NewWizardModal(120, 40)
	m.Update(wzKey("2")) // shortcut to subscription
	if m.step != WizardStepSubURL {
		t.Fatalf("'2' should jump to WizardStepSubURL, got step %d", m.step)
	}
}

func TestWizard_URLPathProducesProfileAndVisibleDone(t *testing.T) {
	m := NewWizardModal(120, 40)
	m.Update(wzKey("enter")) // -> URL step
	m.urlInput.SetValue("vless://aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee@1.2.3.4:443?type=tcp#profile1")
	m.Update(wzKey("enter")) // parse -> Name step
	if m.step != WizardStepName {
		t.Fatalf("expected WizardStepName after parse, got step %d", m.step)
	}
	m.Update(wzKey("enter")) // accept name -> Done step (visible, not closed)
	if m.step != WizardStepDone {
		t.Fatalf("expected WizardStepDone after name, got step %d", m.step)
	}
	if m.Done {
		t.Fatal("Done must be false while the what-next card is visible")
	}
	if m.Profile == nil || m.Profile.Name != "profile1" {
		t.Fatalf("expected parsed Profile named profile1, got %+v", m.Profile)
	}
	m.Update(wzKey("enter")) // close the Done card
	if !m.Done || m.Skipped {
		t.Fatalf("enter on Done must commit (Done=true, Skipped=false), got Done=%v Skipped=%v", m.Done, m.Skipped)
	}
}

func TestWizard_SubscriptionPathCapturesURLAndName(t *testing.T) {
	m := NewWizardModal(120, 40)
	m.Update(wzKey("2")) // subscription
	m.subURLInput.SetValue("https://example.com/sub")
	m.Update(wzKey("enter")) // -> SubName
	if m.step != WizardStepSubName {
		t.Fatalf("expected WizardStepSubName, got step %d", m.step)
	}
	m.subNameInput.SetValue("my-sub")
	m.Update(wzKey("enter")) // -> Done
	if m.step != WizardStepDone || m.Done {
		t.Fatalf("expected visible Done (step=Done, Done=false), got step %d Done %v", m.step, m.Done)
	}
	if m.SubURL != "https://example.com/sub" || m.SubName != "my-sub" {
		t.Fatalf("subscription result not captured: SubURL=%q SubName=%q", m.SubURL, m.SubName)
	}
	if m.Profile != nil {
		t.Fatalf("subscription path must not set Profile, got %+v", m.Profile)
	}
}

func TestWizard_SubscriptionRejectsNonHTTP(t *testing.T) {
	m := NewWizardModal(120, 40)
	m.Update(wzKey("2"))
	m.subURLInput.SetValue("ftp://example.com/sub")
	m.Update(wzKey("enter"))
	if m.step != WizardStepSubURL {
		t.Fatalf("non-http subscription URL must keep us on SubURL, got step %d", m.step)
	}
	if m.err == "" {
		t.Fatal("expected an inline error for a non-http subscription URL")
	}
}

func TestWizard_DoneNudgeUsesInjectedStartKey(t *testing.T) {
	m := NewWizardModal(120, 40)
	m.StartKey = "z"
	m.method = methodURL
	m.Profile = &config.Profile{Name: "p"}
	m.step = WizardStepDone
	if !strings.Contains(m.View(), "[z]") {
		t.Errorf("Done view must cite the injected start key [z], view:\n%s", m.View())
	}
}

func TestWizard_EscAtMethodSkips(t *testing.T) {
	m := NewWizardModal(120, 40)
	m.Update(wzKey("esc"))
	if !m.Done || !m.Skipped {
		t.Fatalf("esc at the method step must skip, got Done=%v Skipped=%v", m.Done, m.Skipped)
	}
}

func TestWizard_EscFromURLGoesBackToMethod(t *testing.T) {
	m := NewWizardModal(120, 40)
	m.Update(wzKey("enter")) // -> URL
	m.Update(wzKey("esc"))   // back -> Method
	if m.step != WizardStepMethod || m.Done {
		t.Fatalf("esc from URL must return to Method without closing, got step %d Done %v", m.step, m.Done)
	}
}

func TestWizard_RendersUnderAllThemes(t *testing.T) {
	steps := []WizardStep{WizardStepMethod, WizardStepURL, WizardStepSubURL, WizardStepSubName}
	for _, name := range theme.Names() {
		theme.Set(name)
		for _, s := range steps {
			m := NewWizardModal(120, 40)
			m.step = s
			if strings.TrimSpace(m.View()) == "" {
				t.Errorf("wizard step %d renders empty under theme %q", s, name)
			}
		}
	}
	theme.Set("gruvbox-dark") // restore default for other tests
}
