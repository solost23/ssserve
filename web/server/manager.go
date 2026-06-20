package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Manager struct {
	apiAddr     string
	inboundTag  string
	inboundPort int
}

func NewManager(apiAddr, inboundTag string, inboundPort int) *Manager {
	return &Manager{apiAddr: apiAddr, inboundTag: inboundTag, inboundPort: inboundPort}
}

func (m *Manager) AddUser(uuid, email string) error {
	cfg, err := os.CreateTemp("", "xray-user-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(cfg.Name())
	defer cfg.Close()

	payload := buildAddUserPayload(m.inboundTag, m.inboundPort, uuid, email)
	if err := json.NewEncoder(cfg).Encode(payload); err != nil {
		return err
	}
	if err := cfg.Close(); err != nil {
		return err
	}

	out, err := m.run("adu", cfg.Name())
	if err != nil {
		if strings.Contains(out, "already exists") {
			return nil
		}
		return fmt.Errorf("xray add user: %w: %s", err, out)
	}
	return nil
}

func buildAddUserPayload(inboundTag string, inboundPort int, uuid, email string) map[string]any {
	return map[string]any{
		"inbounds": []map[string]any{
			{
				"tag":      inboundTag,
				"listen":   "0.0.0.0",
				"port":     inboundPort,
				"protocol": "vless",
				"settings": map[string]any{
					"clients": []map[string]any{
						{
							"id":    uuid,
							"email": email,
							"flow":  "xtls-rprx-vision",
						},
					},
					"decryption": "none",
				},
			},
		},
	}
}

func (m *Manager) RemoveUser(email string) error {
	out, err := m.run("rmu", "-tag="+m.inboundTag, email)
	if err != nil {
		if strings.Contains(out, "not found") {
			return nil
		}
		return fmt.Errorf("xray remove user: %w: %s", err, out)
	}
	return nil
}

func (m *Manager) Stats() (map[string]int64, error) {
	out, err := m.run("statsquery", "-pattern", "user>>>")
	if err != nil {
		return nil, fmt.Errorf("xray stats: %w: %s", err, out)
	}

	stats := make(map[string]int64)
	var resp struct {
		Stat []struct {
			Name  string `json:"name"`
			Value int64  `json:"value"`
		} `json:"stat"`
	}
	if err := json.Unmarshal([]byte(out), &resp); err == nil && len(resp.Stat) > 0 {
		for _, s := range resp.Stat {
			addStat(stats, s.Name, s.Value)
		}
		return stats, nil
	}

	re := regexp.MustCompile(`(?s)name:\s*"([^"]+)".*?value:\s*([0-9]+)`)
	for _, match := range re.FindAllStringSubmatch(out, -1) {
		v, _ := strconv.ParseInt(match[2], 10, 64)
		addStat(stats, match[1], v)
	}
	return stats, nil
}

func addStat(stats map[string]int64, name string, value int64) {
	parts := strings.Split(name, ">>>")
	if len(parts) < 4 || parts[0] != "user" {
		return
	}
	stats[parts[1]] += value
}

func (m *Manager) run(args ...string) (string, error) {
	base := []string{"api", args[0], "--server=" + m.apiAddr, "--timeout=3"}
	base = append(base, args[1:]...)
	cmd := exec.Command("xray", base...)
	var out strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &out
	errCh := make(chan error, 1)
	if err := cmd.Start(); err != nil {
		return out.String(), err
	}
	go func() { errCh <- cmd.Wait() }()
	select {
	case err := <-errCh:
		return out.String(), err
	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()
		return out.String(), fmt.Errorf("timeout")
	}
}
