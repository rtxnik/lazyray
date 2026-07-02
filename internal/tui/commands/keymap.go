package commands

import (
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/rtxnik/lazyray/internal/config"
	"gopkg.in/yaml.v3"
)

// KeyMap defines all keybindings.
type KeyMap struct {
	Quit           key.Binding
	Tab            key.Binding
	ShiftTab       key.Binding
	Up             key.Binding
	Down           key.Binding
	Enter          key.Binding
	Escape         key.Binding
	Start          key.Binding
	Restart        key.Binding
	Doctor         key.Binding
	Tunnel         key.Binding
	Import         key.Binding
	Update         key.Binding
	EditConfig     key.Binding
	Export         key.Binding
	Delete         key.Binding
	Rename         key.Binding
	ShiftUp        key.Binding
	ShiftDown      key.Binding
	QRExport       key.Binding
	ConfigDiff     key.Binding
	ToggleLog      key.Binding
	FilterLog      key.Binding
	SearchLog      key.Binding
	Help           key.Binding
	Subscriptions  key.Binding
	FilterGroup    key.Binding
	TestAll        key.Binding
	Duplicate      key.Binding
	ToggleCollapse key.Binding
	RoutingEdit    key.Binding
	Activity       key.Binding
	Palette        key.Binding
	ToggleMetric   key.Binding
}

// KeysConfig represents the keys.yaml file format.
type KeysConfig struct {
	Quit          string `yaml:"quit,omitempty"`
	Start         string `yaml:"start,omitempty"`
	Restart       string `yaml:"restart,omitempty"`
	Doctor        string `yaml:"doctor,omitempty"`
	Health        string `yaml:"health,omitempty"` // back-compat alias for doctor
	Tunnel        string `yaml:"tunnel,omitempty"`
	Import        string `yaml:"import,omitempty"`
	Update        string `yaml:"update,omitempty"`
	EditConfig    string `yaml:"editConfig,omitempty"`
	Export        string `yaml:"export,omitempty"`
	Delete        string `yaml:"delete,omitempty"`
	Rename        string `yaml:"rename,omitempty"`
	QRExport      string `yaml:"qrExport,omitempty"`
	ConfigDiff    string `yaml:"configDiff,omitempty"`
	ToggleLog     string `yaml:"toggleLog,omitempty"`
	FilterLog     string `yaml:"filterLog,omitempty"`
	SearchLog     string `yaml:"searchLog,omitempty"`
	Help          string `yaml:"help,omitempty"`
	Subscriptions string `yaml:"subscriptions,omitempty"`
	FilterGroup   string `yaml:"filterGroup,omitempty"`
	TestAll       string `yaml:"testAll,omitempty"`
	Duplicate     string `yaml:"duplicate,omitempty"`
	RoutingEdit   string `yaml:"routingEdit,omitempty"`
	Activity      string `yaml:"activity,omitempty"`
	Palette       string `yaml:"palette,omitempty"`
	ToggleMetric  string `yaml:"toggleMetric,omitempty"`
}

// BaseKeyMap returns the built-in default bindings WITHOUT applying any
// keys.yaml overrides. Documentation tooling uses it so generated output is
// independent of the local environment; runtime code should use DefaultKeyMap.
func BaseKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next panel"),
		),
		ShiftTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev panel"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("up/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("down/j", "down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "close/cancel"),
		),
		Start: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "start/stop"),
		),
		Restart: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "restart"),
		),
		Doctor: key.NewBinding(
			key.WithKeys("h"),
			key.WithHelp("h", "diagnostics"),
		),
		Tunnel: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "SSH tunnel"),
		),
		Import: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "import config"),
		),
		Update: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "update xray"),
		),
		EditConfig: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "edit config"),
		),
		Export: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "export VLESS URL"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete profile"),
		),
		Rename: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "rename profile"),
		),
		ShiftUp: key.NewBinding(
			key.WithKeys("shift+up", "K"),
			key.WithHelp("shift+↑/K", "move profile up"),
		),
		ShiftDown: key.NewBinding(
			key.WithKeys("shift+down", "J"),
			key.WithHelp("shift+↓/J", "move profile down"),
		),
		QRExport: key.NewBinding(
			key.WithKeys("Q"),
			key.WithHelp("Q", "QR code export"),
		),
		ConfigDiff: key.NewBinding(
			key.WithKeys("D"),
			key.WithHelp("D", "config diff"),
		),
		ToggleLog: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "toggle error/access log"),
		),
		FilterLog: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "filter logs"),
		),
		SearchLog: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search logs"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Subscriptions: key.NewBinding(
			key.WithKeys("S"),
			key.WithHelp("S", "subscriptions"),
		),
		FilterGroup: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "filter group"),
		),
		TestAll: key.NewBinding(
			key.WithKeys("T"),
			key.WithHelp("T", "test all latency"),
		),
		Duplicate: key.NewBinding(
			key.WithKeys("Y"),
			key.WithHelp("Y", "duplicate profile"),
		),
		ToggleCollapse: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "collapse/expand group"),
		),
		RoutingEdit: key.NewBinding(
			key.WithKeys("W"),
			key.WithHelp("W", "routing rules"),
		),
		Activity: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "activity log"),
		),
		Palette: key.NewBinding(
			key.WithKeys(":"),
			key.WithHelp(":", "command palette"),
		),
		ToggleMetric: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "toggle dashboard metric"),
		),
	}
}

// DefaultKeyMap returns the default keybindings with any keys.yaml overrides applied.
func DefaultKeyMap() KeyMap {
	km := BaseKeyMap()
	km.loadCustomKeys()
	return km
}

// loadCustomKeys reads keys.yaml and overrides default bindings.
func (km *KeyMap) loadCustomKeys() {
	data, err := os.ReadFile(config.KeysPath())
	if err != nil {
		return // No custom keys file — use defaults
	}

	var kc KeysConfig
	if err := yaml.Unmarshal(data, &kc); err != nil {
		return
	}

	applyKey := func(binding *key.Binding, keys, help string) {
		if keys == "" {
			return
		}
		parts := strings.Split(keys, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		*binding = key.NewBinding(
			key.WithKeys(parts...),
			key.WithHelp(parts[0], help),
		)
	}

	applyKey(&km.Quit, kc.Quit, "quit")
	applyKey(&km.Start, kc.Start, "start/stop")
	applyKey(&km.Restart, kc.Restart, "restart")
	doctorKeys := kc.Doctor
	if doctorKeys == "" {
		doctorKeys = kc.Health // back-compat: `health:` still binds diagnostics
	}
	applyKey(&km.Doctor, doctorKeys, "diagnostics")
	applyKey(&km.Tunnel, kc.Tunnel, "SSH tunnel")
	applyKey(&km.Import, kc.Import, "import config")
	applyKey(&km.Update, kc.Update, "update xray")
	applyKey(&km.EditConfig, kc.EditConfig, "edit config")
	applyKey(&km.Export, kc.Export, "export VLESS URL")
	applyKey(&km.Delete, kc.Delete, "delete profile")
	applyKey(&km.Rename, kc.Rename, "rename profile")
	applyKey(&km.QRExport, kc.QRExport, "QR code export")
	applyKey(&km.ConfigDiff, kc.ConfigDiff, "config diff")
	applyKey(&km.ToggleLog, kc.ToggleLog, "toggle error/access log")
	applyKey(&km.FilterLog, kc.FilterLog, "filter logs")
	applyKey(&km.SearchLog, kc.SearchLog, "search logs")
	applyKey(&km.Help, kc.Help, "help")
	applyKey(&km.Subscriptions, kc.Subscriptions, "subscriptions")
	applyKey(&km.FilterGroup, kc.FilterGroup, "filter group")
	applyKey(&km.TestAll, kc.TestAll, "test all latency")
	applyKey(&km.Duplicate, kc.Duplicate, "duplicate profile")
	applyKey(&km.RoutingEdit, kc.RoutingEdit, "routing rules")
	applyKey(&km.Activity, kc.Activity, "activity log")
	applyKey(&km.Palette, kc.Palette, "command palette")
	applyKey(&km.ToggleMetric, kc.ToggleMetric, "toggle dashboard metric")
}
