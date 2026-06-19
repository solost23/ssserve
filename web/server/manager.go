package main

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"
)

type Manager struct {
	addr string
}

func NewManager(addr string) *Manager {
	return &Manager{addr: addr}
}

func (m *Manager) AddServer(port int, password, method string, speedLimitKbps int64) error {
	params := map[string]any{
		"server_port": port,
		"password":    password,
		"method":      method,
		"mode":        "tcp_and_udp",
	}
	if speedLimitKbps > 0 {
		params["speed_limit"] = speedLimitKbps * 1024
	}
	payload, _ := json.Marshal(params)
	resp, err := m.send("add: " + string(payload))
	if err != nil {
		return err
	}
	if strings.TrimSpace(resp) != "ok" {
		return fmt.Errorf("manager add: %s", resp)
	}
	return nil
}

func (m *Manager) RemoveServer(port int) error {
	payload, _ := json.Marshal(map[string]any{"server_port": port})
	resp, err := m.send("remove: " + string(payload))
	if err != nil {
		return err
	}
	if strings.TrimSpace(resp) != "ok" {
		return fmt.Errorf("manager remove: %s", resp)
	}
	return nil
}

// Stats returns cumulative bytes per port since ssserver last started.
func (m *Manager) Stats() (map[int]int64, error) {
	conn, err := net.Dial("udp", m.addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(3 * time.Second))

	if _, err := conn.Write([]byte("ping")); err != nil {
		return nil, err
	}

	buf := make([]byte, 65535)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}

	stats := make(map[int]int64)
	s := string(buf[:n])
	if strings.HasPrefix(s, "stat: ") {
		var raw map[string]int64
		if jsonErr := json.Unmarshal([]byte(strings.TrimPrefix(s, "stat: ")), &raw); jsonErr == nil {
			for k, v := range raw {
				var port int
				fmt.Sscanf(k, "%d", &port)
				stats[port] = v
			}
		}
	}
	return stats, nil
}

func (m *Manager) send(cmd string) (string, error) {
	conn, err := net.Dial("udp", m.addr)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(3 * time.Second))
	if _, err := conn.Write([]byte(cmd)); err != nil {
		return "", err
	}
	buf := make([]byte, 65535)
	n, err := conn.Read(buf)
	if err != nil {
		return "", err
	}
	return string(buf[:n]), nil
}
