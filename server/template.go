package main

import "fmt"

func renderClash(cfg Config, password string, port int) string {
	return fmt.Sprintf(`mixed-port: 7890
allow-lan: false
mode: rule
log-level: info

proxies:
  - name: %s
    type: ss
    server: %s
    port: %d
    cipher: %s
    password: "%s"
    udp: true

proxy-groups:
  - name: Proxy
    type: select
    proxies:
      - %s
      - DIRECT

rules:
  - GEOIP,CN,DIRECT
  - MATCH,Proxy
`, cfg.NodeName, cfg.Domain, port, cfg.Cipher, password, cfg.NodeName)
}
