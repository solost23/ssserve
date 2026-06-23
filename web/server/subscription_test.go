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

func TestRenderDirectRulesNormalizesIPAndDeduplicates(t *testing.T) {
	cfg := testConfig()
	cfg.ServerAddr = "http://[2001:db8::1]:8080/"

	rules := renderDirectRules(cfg)
	if want := "  - IP-CIDR6,2001:db8::1/128,DIRECT,no-resolve\n"; rules != want {
		t.Fatalf("rules = %q, want %q", rules, want)
	}
}
