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
log-level: warning
tcp-concurrent: true
ipv6: false

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
    tfo: true
    reality-opts:
      public-key: %s
      short-id: %s
      spider-x: "/"

proxy-groups:
  - name: Proxy
    type: select
    url: http://cp.cloudflare.com/generate_204
    interval: 300
    proxies:
      - %s
      - DIRECT

rules:
  - DOMAIN,localhost,DIRECT
  - DOMAIN-SUFFIX,local,DIRECT
  - IP-CIDR,10.0.0.0/8,DIRECT,no-resolve
  - IP-CIDR,100.64.0.0/10,DIRECT,no-resolve
  - IP-CIDR,127.0.0.0/8,DIRECT,no-resolve
  - IP-CIDR,169.254.0.0/16,DIRECT,no-resolve
  - IP-CIDR,172.16.0.0/12,DIRECT,no-resolve
  - IP-CIDR,192.168.0.0/16,DIRECT,no-resolve
  - IP-CIDR,224.0.0.0/4,DIRECT,no-resolve
  - IP-CIDR6,::1/128,DIRECT,no-resolve
  - IP-CIDR6,fc00::/7,DIRECT,no-resolve
  - IP-CIDR6,fe80::/10,DIRECT,no-resolve
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
