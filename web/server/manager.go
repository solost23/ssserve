package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	handlercmd "github.com/xtls/xray-core/app/proxyman/command"
	statscmd "github.com/xtls/xray-core/app/stats/command"
	"github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/proxy/vless"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const xrayAPITimeout = 3 * time.Second

type Manager struct {
	apiAddr      string
	vlessInbound string

	mu            sync.Mutex
	conn          *grpc.ClientConn
	handlerClient handlercmd.HandlerServiceClient
	statsClient   statscmd.StatsServiceClient
}

type ManagerConfig struct {
	APIAddr      string
	VLESSInbound string
}

func NewManager(cfg ManagerConfig) *Manager {
	return &Manager{
		apiAddr:      cfg.APIAddr,
		vlessInbound: cfg.VLESSInbound,
	}
}

func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.conn == nil {
		return nil
	}
	err := m.conn.Close()
	m.conn = nil
	m.handlerClient = nil
	m.statsClient = nil
	return err
}

func (m *Manager) AddUser(uuid, email string) error {
	client, err := m.handler()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), xrayAPITimeout)
	defer cancel()

	_, err = client.AlterInbound(ctx, buildVLESSAddUserRequest(m.vlessInbound, uuid, email))
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return nil
		}
		return fmt.Errorf("xray add user: %w", err)
	}
	return nil
}

func buildVLESSAddUserRequest(inboundTag, uuid, email string) *handlercmd.AlterInboundRequest {
	return &handlercmd.AlterInboundRequest{
		Tag: inboundTag,
		Operation: serial.ToTypedMessage(&handlercmd.AddUserOperation{
			User: &protocol.User{
				Email: email,
				Account: serial.ToTypedMessage(&vless.Account{
					Id:         uuid,
					Flow:       "xtls-rprx-vision",
					Encryption: "none",
				}),
			},
		}),
	}
}

func (m *Manager) RemoveUser(email string) error {
	client, err := m.handler()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), xrayAPITimeout)
	defer cancel()

	_, err = client.AlterInbound(ctx, buildRemoveUserRequest(m.vlessInbound, email))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil
		}
		return fmt.Errorf("xray remove user: %w", err)
	}
	return nil
}

func buildRemoveUserRequest(inboundTag, email string) *handlercmd.AlterInboundRequest {
	return &handlercmd.AlterInboundRequest{
		Tag:       inboundTag,
		Operation: serial.ToTypedMessage(&handlercmd.RemoveUserOperation{Email: email}),
	}
}

func (m *Manager) Stats() (map[string]int64, error) {
	client, err := m.stats()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), xrayAPITimeout)
	defer cancel()

	resp, err := client.QueryStats(ctx, &statscmd.QueryStatsRequest{Pattern: "user>>>"})
	if err != nil {
		return nil, fmt.Errorf("xray stats: %w", err)
	}
	stats := make(map[string]int64)
	for _, s := range resp.GetStat() {
		addStat(stats, s.GetName(), s.GetValue())
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

func (m *Manager) handler() (handlercmd.HandlerServiceClient, error) {
	if err := m.connect(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.handlerClient, nil
}

func (m *Manager) stats() (statscmd.StatsServiceClient, error) {
	if err := m.connect(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.statsClient, nil
}

func (m *Manager) connect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.conn != nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), xrayAPITimeout)
	defer cancel()
	conn, err := grpc.DialContext(
		ctx,
		m.apiAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return fmt.Errorf("xray api connect: %w", err)
	}
	m.conn = conn
	m.handlerClient = handlercmd.NewHandlerServiceClient(conn)
	m.statsClient = statscmd.NewStatsServiceClient(conn)
	return nil
}
