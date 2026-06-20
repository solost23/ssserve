package main

import (
	"fmt"
	"strconv"
)

func renderClash(cfg Config, uuid string) string {
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
    proxies:
      - %s
      - DIRECT

rules:
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
	)
}

func yamlQuote(s string) string {
	return strconv.Quote(s)
}
