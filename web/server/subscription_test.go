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
		TrojanEnabled:  true,
		TrojanDomain:   "202.182.111.110.sslip.io",
		TrojanPort:     8443,
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

func TestTrojanURL(t *testing.T) {
	raw := testConfig().TrojanURL("7eb14c94-37e7-4a17-9af5-40535c927af9")
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}

	if parsed.Scheme != "trojan" {
		t.Fatalf("scheme = %q, want trojan", parsed.Scheme)
	}
	if parsed.User.Username() != "7eb14c94-37e7-4a17-9af5-40535c927af9" {
		t.Fatalf("username = %q", parsed.User.Username())
	}
	if parsed.Host != "202.182.111.110.sslip.io:8443" {
		t.Fatalf("host = %q", parsed.Host)
	}
	if got := parsed.Query().Get("sni"); got != "202.182.111.110.sslip.io" {
		t.Fatalf("sni = %q", got)
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
	cfg.TrojanDomain = "202.182.111.110.sslip.io"

	yaml := renderClash(cfg, "1a078af0-1bb6-498b-9896-4651db5cbaf4")
	for _, want := range []string{
		`  - DOMAIN,sub.example.com,DIRECT`,
		`  - DOMAIN,202.182.111.110.sslip.io,DIRECT`,
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

func TestRenderDirectRulesNormalizesIPAndDeduplicates(t *testing.T) {
	cfg := testConfig()
	cfg.ServerAddr = "http://[2001:db8::1]:8080/"
	cfg.TrojanDomain = "HTTP://[2001:db8::1]:8443/"

	rules := renderDirectRules(cfg)
	if want := "  - IP-CIDR6,2001:db8::1/128,DIRECT,no-resolve\n"; rules != want {
		t.Fatalf("rules = %q, want %q", rules, want)
	}
}
