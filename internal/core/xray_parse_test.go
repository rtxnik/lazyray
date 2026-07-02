package core

import (
	"testing"
	"time"
)

func TestParseAllTrafficStats(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantUp   int64
		wantDown int64
	}{
		// === Protobuf text format (older xray-core) ===
		{
			name: "protobuf text format",
			input: `stat: {
  name: "outbound>>>proxy>>>traffic>>>uplink"
  value: 12345
}
stat: {
  name: "outbound>>>proxy>>>traffic>>>downlink"
  value: 67890
}`,
			wantUp:   12345,
			wantDown: 67890,
		},
		{
			name: "protobuf quoted values",
			input: `stat: {
  name: "outbound>>>proxy>>>traffic>>>uplink"
  value: "11111"
}
stat: {
  name: "outbound>>>proxy>>>traffic>>>downlink"
  value: "22222"
}`,
			wantUp:   11111,
			wantDown: 22222,
		},
		{
			name: "protobuf mixed stats including non-proxy",
			input: `stat: {
  name: "inbound>>>socks-in>>>traffic>>>uplink"
  value: 100
}
stat: {
  name: "outbound>>>proxy>>>traffic>>>uplink"
  value: 9876543210
}
stat: {
  name: "outbound>>>proxy>>>traffic>>>downlink"
  value: 1073741824
}
stat: {
  name: "outbound>>>direct>>>traffic>>>uplink"
  value: 200
}`,
			wantUp:   9876543210,
			wantDown: 1073741824,
		},
		{
			name: "protobuf only uplink present",
			input: `stat: {
  name: "outbound>>>proxy>>>traffic>>>uplink"
  value: 42
}`,
			wantUp:   42,
			wantDown: 0,
		},

		// === JSON format (modern xray-core 1.8+) ===
		{
			name:     "json format with string values",
			input:    `{"stat":[{"name":"outbound>>>proxy>>>traffic>>>uplink","value":"12345"},{"name":"outbound>>>proxy>>>traffic>>>downlink","value":"67890"}]}`,
			wantUp:   12345,
			wantDown: 67890,
		},
		{
			name:     "json format with numeric values",
			input:    `{"stat":[{"name":"outbound>>>proxy>>>traffic>>>uplink","value":12345},{"name":"outbound>>>proxy>>>traffic>>>downlink","value":67890}]}`,
			wantUp:   12345,
			wantDown: 67890,
		},
		{
			name: "json format pretty-printed",
			input: `{
  "stat": [
    {
      "name": "outbound>>>proxy>>>traffic>>>uplink",
      "value": "9876543210"
    },
    {
      "name": "outbound>>>proxy>>>traffic>>>downlink",
      "value": "1073741824"
    }
  ]
}`,
			wantUp:   9876543210,
			wantDown: 1073741824,
		},
		{
			name: "json mixed stats",
			input: `{
  "stat": [
    {"name": "inbound>>>socks-in>>>traffic>>>uplink", "value": "100"},
    {"name": "outbound>>>proxy>>>traffic>>>uplink", "value": "55555"},
    {"name": "outbound>>>proxy>>>traffic>>>downlink", "value": "88888"},
    {"name": "outbound>>>direct>>>traffic>>>uplink", "value": "200"}
  ]
}`,
			wantUp:   55555,
			wantDown: 88888,
		},
		{
			name:     "json empty stat array",
			input:    `{"stat":[]}`,
			wantUp:   0,
			wantDown: 0,
		},

		// === Edge cases ===
		{
			name:     "empty output",
			input:    "",
			wantUp:   0,
			wantDown: 0,
		},
		{
			name:     "no matching stats text",
			input:    "stat: {}\n",
			wantUp:   0,
			wantDown: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseAllTrafficStats(tc.input)
			if got.Uplink != tc.wantUp {
				t.Errorf("Uplink = %d, want %d", got.Uplink, tc.wantUp)
			}
			if got.Downlink != tc.wantDown {
				t.Errorf("Downlink = %d, want %d", got.Downlink, tc.wantDown)
			}
		})
	}
}

func TestParseStatsJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantNil  bool
		wantUp   int64
		wantDown int64
	}{
		{
			name:     "valid json with string values",
			input:    `{"stat":[{"name":"outbound>>>proxy>>>traffic>>>uplink","value":"42"},{"name":"outbound>>>proxy>>>traffic>>>downlink","value":"99"}]}`,
			wantUp:   42,
			wantDown: 99,
		},
		{
			name:     "valid json with numeric values",
			input:    `{"stat":[{"name":"outbound>>>proxy>>>traffic>>>uplink","value":42},{"name":"outbound>>>proxy>>>traffic>>>downlink","value":99}]}`,
			wantUp:   42,
			wantDown: 99,
		},
		{
			name:    "invalid json returns nil",
			input:   "not json at all",
			wantNil: true,
		},
		{
			name:    "protobuf text returns nil",
			input:   "stat: {\n  name: \"foo\"\n  value: 1\n}",
			wantNil: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseStatsJSON(tc.input)
			if tc.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil, got nil")
			}
			if got.Uplink != tc.wantUp {
				t.Errorf("Uplink = %d, want %d", got.Uplink, tc.wantUp)
			}
			if got.Downlink != tc.wantDown {
				t.Errorf("Downlink = %d, want %d", got.Downlink, tc.wantDown)
			}
		})
	}
}

func TestParseStatsText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantUp   int64
		wantDown int64
	}{
		{
			name: "standard format",
			input: `stat: {
  name: "outbound>>>proxy>>>traffic>>>uplink"
  value: 100
}
stat: {
  name: "outbound>>>proxy>>>traffic>>>downlink"
  value: 200
}`,
			wantUp:   100,
			wantDown: 200,
		},
		{
			name:     "empty",
			input:    "",
			wantUp:   0,
			wantDown: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseStatsText(tc.input)
			if got.Uplink != tc.wantUp {
				t.Errorf("Uplink = %d, want %d", got.Uplink, tc.wantUp)
			}
			if got.Downlink != tc.wantDown {
				t.Errorf("Downlink = %d, want %d", got.Downlink, tc.wantDown)
			}
		})
	}
}

func TestFormatUptime(t *testing.T) {
	tests := []struct {
		name string
		dur  time.Duration
		want string
	}{
		{"zero", 0, "-"},
		{"one minute", time.Minute, "1m"},
		{"30 minutes", 30 * time.Minute, "30m"},
		{"59 minutes", 59 * time.Minute, "59m"},
		{"one hour", time.Hour, "1h 0m"},
		{"1h30m", 90 * time.Minute, "1h 30m"},
		{"24 hours", 24 * time.Hour, "24h 0m"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatUptime(tc.dur)
			if got != tc.want {
				t.Errorf("FormatUptime(%v) = %q, want %q", tc.dur, got, tc.want)
			}
		})
	}
}
