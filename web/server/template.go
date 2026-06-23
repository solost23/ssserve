package main

import (
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
)

func renderClash(cfg Config, uuid string) string {
	directRules := renderDirectRules(cfg)
	return fmt.Sprintf(`mixed-port: 7890
allow-lan: false
mode: rule
log-level: info

proxies:
  - name: %s
    type: vless
    server: %s
    port: %d
    uuid: %s
    network: tcp
    udp: true
    tls: true
    flow: xtls-rprx-vision
    servername: %s
    client-fingerprint: chrome
    reality-opts:
      public-key: %s
      short-id: %s
      spider-x: "/"

proxy-groups:
  - name: Proxy
    type: select
    url: https://www.gstatic.com/generate_204
    proxies:
      - %s
      - DIRECT

rules:
%s
  - GEOIP,CN,DIRECT
  - MATCH,Proxy
`,
		yamlQuote(cfg.NodeName),
		yamlQuote(cfg.ServerAddr),
		cfg.XrayPort,
		yamlQuote(uuid),
		yamlQuote(cfg.XrayServerName),
		yamlQuote(cfg.XrayPublicKey),
		yamlQuote(cfg.XrayShortID),
		yamlQuote(cfg.NodeName),
		directRules,
	)
}

func yamlQuote(s string) string {
	return strconv.Quote(s)
}

func renderDirectRules(cfg Config) string {
	hosts := []string{cfg.ServerAddr}

	seen := make(map[string]bool, len(hosts))
	lines := make([]string, 0, len(hosts))
	for _, raw := range hosts {
		host := normalizeRuleHost(raw)
		if host == "" || seen[host] {
			continue
		}
		seen[host] = true
		if addr, err := netip.ParseAddr(host); err == nil {
			if addr.Is4() {
				lines = append(lines, fmt.Sprintf("  - IP-CIDR,%s/32,DIRECT,no-resolve", host))
			} else {
				lines = append(lines, fmt.Sprintf("  - IP-CIDR6,%s/128,DIRECT,no-resolve", host))
			}
			continue
		}
		lines = append(lines, fmt.Sprintf("  - DOMAIN,%s,DIRECT", host))
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

func normalizeRuleHost(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.Contains(raw, "://") {
		if u, err := url.Parse(raw); err == nil {
			raw = u.Host
		}
	}
	if host, _, err := net.SplitHostPort(raw); err == nil {
		raw = host
	}
	raw = strings.Trim(raw, "[]")
	raw = strings.TrimSuffix(raw, ".")
	return strings.ToLower(raw)
}
