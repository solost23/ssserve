package main

import (
	"net/url"
	"strings"
	"testing"
)

func testConfig() Config {
	return Config{
		ServerAddr:     "202.182.111.110",
		NodeName:       "Tokyo",
		XrayPort:       443,
		XrayPublicKey:  "public-key",
		XrayShortID:    "f919438ba90e7ae3",
		XrayServerName: "www.cloudflare.com",
	}
}

func TestVLESSURLIncludesRealityOptions(t *testing.T) {
	raw := testConfig().VLESSURL("1a078af0-1bb6-498b-9896-4651db5cbaf4")
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}

	q := parsed.Query()
	for key, want := range map[string]string{
		"encryption": "none",
		"flow":       "xtls-rprx-vision",
		"security":   "reality",
		"sni":        "www.cloudflare.com",
		"fp":         "chrome",
		"pbk":        "public-key",
		"sid":        "f919438ba90e7ae3",
		"spx":        "/",
		"type":       "tcp",
	} {
		if got := q.Get(key); got != want {
			t.Fatalf("query %s = %q, want %q", key, got, want)
		}
	}
}

func TestRenderClashIncludesRealityOptions(t *testing.T) {
	yaml := renderClash(testConfig(), "1a078af0-1bb6-498b-9896-4651db5cbaf4")

	for _, want := range []string{
		`servername: "www.cloudflare.com"`,
		`public-key: "public-key"`,
		`short-id: "f919438ba90e7ae3"`,
		`spider-x: "/"`,
	} {
		if !strings.Contains(yaml, want) {
			t.Fatalf("clash yaml missing %q:\n%s", want, yaml)
		}
	}
}

func TestRenderClashDoesNotSetProxyGroupTestURL(t *testing.T) {
	yaml := renderClash(testConfig(), "1a078af0-1bb6-498b-9896-4651db5cbaf4")

	if strings.Contains(yaml, "url:") {
		t.Fatalf("clash yaml should let clients choose test targets:\n%s", yaml)
	}
}

func TestRenderClashDoesNotSetExternalController(t *testing.T) {
	yaml := renderClash(testConfig(), "1a078af0-1bb6-498b-9896-4651db5cbaf4")

	if strings.Contains(yaml, "external-controller:") {
		t.Fatalf("clash yaml should not reserve a controller port:\n%s", yaml)
	}
}

func TestRenderClashIncludesFakeIPFilter(t *testing.T) {
	yaml := renderClash(testConfig(), "1a078af0-1bb6-498b-9896-4651db5cbaf4")

	for _, want := range []string{
		`  fake-ip-filter:`,
		`    - "*.lan"`,
		`    - "*.local"`,
		`    - connectivitycheck.gstatic.com`,
		`    - captive.apple.com`,
		`    - "*.pool.ntp.org"`,
	} {
		if !strings.Contains(yaml, want) {
			t.Fatalf("clash yaml missing fake-ip filter %q:\n%s", want, yaml)
		}
	}
}

func TestRenderClashIncludesDNSPolicy(t *testing.T) {
	yaml := renderClash(testConfig(), "1a078af0-1bb6-498b-9896-4651db5cbaf4")

	for _, want := range []string{
		`  nameserver:`,
		`    - https://doh.pub/dns-query`,
		`    - https://dns.alidns.com/dns-query`,
		`  nameserver-policy:`,
		`    "geosite:geolocation-!cn":`,
		`      - https://cloudflare-dns.com/dns-query`,
		`      - https://dns.google/dns-query`,
	} {
		if !strings.Contains(yaml, want) {
			t.Fatalf("clash yaml missing dns policy %q:\n%s", want, yaml)
		}
	}
	if strings.Contains(yaml, `"geosite:cn":`) {
		t.Fatalf("clash yaml should use default nameservers for cn domains instead of duplicate policy:\n%s", yaml)
	}
}

func TestRenderClashDirectsServerHostsBeforeCatchAll(t *testing.T) {
	cfg := testConfig()
	cfg.ServerAddr = "sub.example.com"

	yaml := renderClash(cfg, "1a078af0-1bb6-498b-9896-4651db5cbaf4")
	for _, want := range []string{
		`  - DOMAIN,sub.example.com,DIRECT`,
	} {
		if !strings.Contains(yaml, want) {
			t.Fatalf("clash yaml missing direct rule %q:\n%s", want, yaml)
		}
	}

	directIdx := strings.Index(yaml, `  - DOMAIN,sub.example.com,DIRECT`)
	matchIdx := strings.Index(yaml, `  - MATCH,Proxy`)
	if directIdx == -1 || matchIdx == -1 || directIdx > matchIdx {
		t.Fatalf("direct rule must appear before MATCH:\n%s", yaml)
	}
}

func TestRenderClashDirectsPrivateNetworksBeforeCatchAll(t *testing.T) {
	yaml := renderClash(testConfig(), "1a078af0-1bb6-498b-9896-4651db5cbaf4")

	for _, want := range []string{
		`  - DOMAIN,localhost,DIRECT`,
		`  - DOMAIN-SUFFIX,local,DIRECT`,
		`  - IP-CIDR,10.0.0.0/8,DIRECT,no-resolve`,
		`  - IP-CIDR,172.16.0.0/12,DIRECT,no-resolve`,
		`  - IP-CIDR,192.168.0.0/16,DIRECT,no-resolve`,
		`  - IP-CIDR6,fc00::/7,DIRECT,no-resolve`,
	} {
		directIdx := strings.Index(yaml, want)
		matchIdx := strings.Index(yaml, `  - MATCH,Proxy`)
		if directIdx == -1 || matchIdx == -1 || directIdx > matchIdx {
			t.Fatalf("private direct rule %q must appear before MATCH:\n%s", want, yaml)
		}
	}
}

func TestRenderClashDirectsChinaSitesBeforeCatchAll(t *testing.T) {
	yaml := renderClash(testConfig(), "1a078af0-1bb6-498b-9896-4651db5cbaf4")

	geositeIdx := strings.Index(yaml, `  - GEOSITE,CN,DIRECT`)
	geoipIdx := strings.Index(yaml, `  - GEOIP,CN,DIRECT`)
	matchIdx := strings.Index(yaml, `  - MATCH,Proxy`)
	if geositeIdx == -1 || geoipIdx == -1 || matchIdx == -1 {
		t.Fatalf("clash yaml missing china direct rules:\n%s", yaml)
	}
	if geositeIdx > geoipIdx || geoipIdx > matchIdx {
		t.Fatalf("china direct rules must appear before MATCH and GEOSITE before GEOIP:\n%s", yaml)
	}
}

func TestRenderDirectRulesNormalizesIPAndDeduplicates(t *testing.T) {
	cfg := testConfig()
	cfg.ServerAddr = "http://[2001:db8::1]:8080/"

	rules := renderDirectRules(cfg)
	if want := "  - IP-CIDR6,2001:db8::1/128,DIRECT,no-resolve\n"; rules != want {
		t.Fatalf("rules = %q, want %q", rules, want)
	}
}
