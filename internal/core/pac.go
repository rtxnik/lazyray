package core

import (
	"fmt"
	"strings"

	"github.com/rtxnik/lazyray/internal/config"
)

// GeneratePAC creates a PAC (Proxy Auto-Configuration) file content.
// The PAC file tells browsers to route traffic through the lazyray SOCKS5/HTTP proxy,
// with optional bypass rules for direct connections.
func GeneratePAC(settings *config.Settings, profile *config.Profile) string {
	listen := settings.Local.Listen
	httpPort := settings.Local.HTTPPort

	proxyString := fmt.Sprintf("PROXY %s:%d; DIRECT", listen, httpPort)

	// Build bypass conditions from profile routing rules
	var bypassConditions []string

	if profile != nil {
		for _, entry := range profile.Routing.Bypass {
			cond := pacConditionFromEntry(entry)
			if cond != "" {
				bypassConditions = append(bypassConditions, cond)
			}
		}
	}

	// Always bypass localhost and private networks
	defaultBypasses := []string{
		`if (isPlainHostName(host)) return "DIRECT";`,
		`if (shExpMatch(host, "localhost")) return "DIRECT";`,
		`if (shExpMatch(host, "127.*")) return "DIRECT";`,
		`if (shExpMatch(host, "10.*")) return "DIRECT";`,
		`if (shExpMatch(host, "172.16.*")) return "DIRECT";`,
		`if (shExpMatch(host, "192.168.*")) return "DIRECT";`,
	}

	var allConditions []string
	allConditions = append(allConditions, defaultBypasses...)
	allConditions = append(allConditions, bypassConditions...)

	conditionsStr := ""
	for _, c := range allConditions {
		conditionsStr += "  " + c + "\n"
	}

	return fmt.Sprintf(`function FindProxyForURL(url, host) {
%s
  return "%s";
}
`, conditionsStr, proxyString)
}

// pacConditionFromEntry converts a routing rule entry to a PAC condition.
func pacConditionFromEntry(entry string) string {
	switch {
	case strings.HasPrefix(entry, "domain:"):
		domain := strings.TrimPrefix(entry, "domain:")
		return fmt.Sprintf(`if (dnsDomainIs(host, "%s") || dnsDomainIs(host, ".%s")) return "DIRECT";`, domain, domain)
	case strings.HasPrefix(entry, "full:"):
		domain := strings.TrimPrefix(entry, "full:")
		return fmt.Sprintf(`if (host === "%s") return "DIRECT";`, domain)
	case strings.HasPrefix(entry, "keyword:"):
		kw := strings.TrimPrefix(entry, "keyword:")
		return fmt.Sprintf(`if (host.indexOf("%s") !== -1) return "DIRECT";`, kw)
	case strings.HasPrefix(entry, "geoip:private"):
		return "" // Already covered by default bypasses
	case strings.HasPrefix(entry, "geosite:"):
		// GeoSite rules can't be directly translated to PAC
		return ""
	case strings.HasPrefix(entry, "geoip:"):
		// GeoIP rules can't be directly translated to PAC
		return ""
	default:
		// Bare domain or IP
		if strings.Contains(entry, ".") && !strings.Contains(entry, "/") {
			if entry[0] >= '0' && entry[0] <= '9' {
				// IP address
				return fmt.Sprintf(`if (isInNet(host, "%s", "255.255.255.255")) return "DIRECT";`, entry)
			}
			// Domain name
			return fmt.Sprintf(`if (dnsDomainIs(host, "%s") || dnsDomainIs(host, ".%s")) return "DIRECT";`, entry, entry)
		}
		return ""
	}
}
