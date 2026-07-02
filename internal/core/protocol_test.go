package core

import "testing"

func TestProtocolRegistry_Complete(t *testing.T) {
	for name, spec := range protocols {
		if spec.Parse == nil || spec.ToURL == nil || spec.BuildOutbound == nil || spec.Validate == nil {
			t.Errorf("protocol %q has a nil function field", name)
		}
	}
	for _, want := range []string{"vless", "vmess", "trojan", "shadowsocks", "hysteria2"} {
		if _, ok := protocols[want]; !ok {
			t.Errorf("registry missing protocol %q", want)
		}
	}
}

func TestProtocolRegistry_DatagramFlag(t *testing.T) {
	if !protocols["hysteria2"].Datagram {
		t.Error("hysteria2 must be Datagram")
	}
	for _, p := range []string{"vless", "vmess", "trojan", "shadowsocks"} {
		if protocols[p].Datagram {
			t.Errorf("%s must not be Datagram", p)
		}
	}
}

func TestSchemeToProtocol(t *testing.T) {
	cases := map[string]string{"vless": "vless", "ss": "shadowsocks", "hy2": "hysteria2", "hysteria2": "hysteria2"}
	for scheme, want := range cases {
		if got := schemeToProtocol[scheme]; got != want {
			t.Errorf("schemeToProtocol[%q] = %q, want %q", scheme, got, want)
		}
	}
}

func TestProtocolMinXrayVersion(t *testing.T) {
	if protocols["hysteria2"].MinXrayVersion != MinXrayVersionHysteria2 {
		t.Errorf("hysteria2 MinXrayVersion = %q, want %q", protocols["hysteria2"].MinXrayVersion, MinXrayVersionHysteria2)
	}
	for _, p := range []string{"vless", "vmess", "trojan", "shadowsocks"} {
		if protocols[p].MinXrayVersion != "" {
			t.Errorf("%s must have no extra version floor, got %q", p, protocols[p].MinXrayVersion)
		}
	}
}
