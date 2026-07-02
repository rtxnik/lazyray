package modals

import "fmt"

// wizardMethod is the add-a-connection method the user picked in the wizard.
type wizardMethod int

const (
	methodURL wizardMethod = iota
	methodSubscription
)

// Nudge is the single "what next" hint shown on the wizard's Done step. It is a
// struct (not a bare string) so future state arms can add fields without
// changing callers — exactly one Nudge is returned, so "at most one nudge at a
// time" holds by construction.
type Nudge struct {
	Text string
}

// nudgeInput is the wizard's terminal state feeding the resolver. On first run
// the proxy is always freshly-stopped, so method is the only discriminator; the
// struct leaves room for running/connected arms later.
type nudgeInput struct {
	method   wizardMethod
	startKey string // display key for "start proxy" (e.g. "s"), already KeyDisplay-resolved
}

// nextNudge maps the wizard's terminal state to exactly one contextual hint,
// inlining the injected start key in [bracket] form.
func nextNudge(in nudgeInput) Nudge {
	key := in.startKey
	if key == "" {
		key = "s" // defensive fallback; the app always injects the real key
	}
	switch in.method {
	case methodSubscription:
		return Nudge{Text: fmt.Sprintf("Servers are importing — they'll appear in Profiles, then press [%s] to start.", key)}
	default: // methodURL
		return Nudge{Text: fmt.Sprintf("Press [%s] to start the proxy.", key)}
	}
}
