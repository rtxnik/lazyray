package core

import (
	"fmt"

	"github.com/rtxnik/lazyray/internal/config"
)

// protocolSpec is the single per-protocol description: how to parse a share
// link, serialize back, build the xray outbound, validate, the minimum xray
// version (if any beyond the global floor), and whether the transport is a
// datagram (UDP/QUIC) transport. Keyed by canonical protocol name.
type protocolSpec struct {
	Parse          func(raw string) (*config.Profile, error)
	ToURL          func(p *config.Profile) string
	BuildOutbound  func(tag string, server config.ServerConfig, proxyTag string) Outbound
	Validate       func(s config.ServerConfig) []string
	MinXrayVersion string
	Datagram       bool
}

var protocols = map[string]protocolSpec{
	"vless":       {Parse: ParseVLESS, ToURL: ToVLESSURL, BuildOutbound: buildVLESSOutbound, Validate: validateGeneric},
	"vmess":       {Parse: ParseVMess, ToURL: ToVMessURL, BuildOutbound: buildVMessOutbound, Validate: validateGeneric},
	"trojan":      {Parse: ParseTrojan, ToURL: ToTrojanURL, BuildOutbound: buildTrojanOutbound, Validate: validateTrojan},
	"shadowsocks": {Parse: ParseShadowsocks, ToURL: ToShadowsocksURL, BuildOutbound: buildShadowsocksOutbound, Validate: validateShadowsocks},
	"hysteria2":   {Parse: ParseHysteria2, ToURL: ToHysteria2URL, BuildOutbound: buildHysteria2Outbound, Validate: validateHysteria2, MinXrayVersion: MinXrayVersionHysteria2, Datagram: true},
}

// schemeToProtocol maps a share-link URL scheme to a canonical protocol name.
var schemeToProtocol = map[string]string{
	"vless":     "vless",
	"vmess":     "vmess",
	"trojan":    "trojan",
	"ss":        "shadowsocks",
	"hysteria2": "hysteria2",
	"hy2":       "hysteria2",
}

// protocolFor returns the registry entry for a canonical protocol name.
func protocolFor(name string) (protocolSpec, bool) {
	spec, ok := protocols[name]
	return spec, ok
}

// isDatagramTransport reports whether a protocol rides UDP/QUIC and therefore
// cannot be liveness-checked by a TCP dial. Keyed by transport class, not a
// one-off protocol literal, so a future UDP transport (TUIC) adds one entry.
func isDatagramTransport(protocol string) bool {
	return protocols[protocol].Datagram
}

// validateGeneric covers vless/vmess (and unknown): a UUID is required.
func validateGeneric(s config.ServerConfig) []string {
	if s.UUID == "" {
		return []string{"UUID is empty"}
	}
	return nil
}

func validateTrojan(s config.ServerConfig) []string {
	if s.UUID == "" {
		return []string{"password is empty"}
	}
	return nil
}

func validateShadowsocks(s config.ServerConfig) []string {
	var errs []string
	if s.UUID == "" {
		errs = append(errs, "password is empty")
	}
	if s.Encryption == "" {
		errs = append(errs, "encryption method is empty")
	}
	return errs
}

func validateHysteria2(s config.ServerConfig) []string {
	var errs []string
	if s.UUID == "" {
		errs = append(errs, "hysteria2 auth is empty")
	}
	if o := s.Obfs; o != "" && o != "salamander" {
		errs = append(errs, fmt.Sprintf("unsupported hysteria2 obfs %q (only salamander is supported)", o))
	}
	if s.Obfs == "salamander" && s.ObfsPassword == "" {
		errs = append(errs, "salamander obfs requires an obfs-password")
	}
	if pin := s.Security.PinSHA256; pin != "" {
		if err := validatePinSHA256(pin); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if hop := s.PortHopping; hop != "" {
		if err := validatePortHopping(hop); err != nil {
			errs = append(errs, err.Error())
		}
	}
	return errs
}
