package main

import (
	"encoding/json"
	"testing"
)

func TestBuildAddUserPayload(t *testing.T) {
	payload := buildVLESSAddUserPayload("vless-in", 443, "1a078af0-1bb6-498b-9896-4651db5cbaf4", "token-1")

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	var got struct {
		Inbounds []struct {
			Tag      string `json:"tag"`
			Listen   string `json:"listen"`
			Port     int    `json:"port"`
			Protocol string `json:"protocol"`
			Settings struct {
				Clients []struct {
					ID    string `json:"id"`
					Email string `json:"email"`
					Flow  string `json:"flow"`
				} `json:"clients"`
				Decryption string `json:"decryption"`
			} `json:"settings"`
		} `json:"inbounds"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	if len(got.Inbounds) != 1 {
		t.Fatalf("expected one inbound, got %d", len(got.Inbounds))
	}
	inbound := got.Inbounds[0]
	if inbound.Tag != "vless-in" || inbound.Listen != "0.0.0.0" || inbound.Port != 443 || inbound.Protocol != "vless" {
		t.Fatalf("unexpected inbound: %+v", inbound)
	}
	if inbound.Settings.Decryption != "none" {
		t.Fatalf("unexpected decryption: %q", inbound.Settings.Decryption)
	}
	if len(inbound.Settings.Clients) != 1 {
		t.Fatalf("expected one client, got %d", len(inbound.Settings.Clients))
	}
	client := inbound.Settings.Clients[0]
	if client.ID != "1a078af0-1bb6-498b-9896-4651db5cbaf4" || client.Email != "token-1" || client.Flow != "xtls-rprx-vision" {
		t.Fatalf("unexpected client: %+v", client)
	}
}

func TestBuildTrojanAddUserPayload(t *testing.T) {
	payload := buildTrojanAddUserPayload("trojan-in", 8443, "7eb14c94-37e7-4a17-9af5-40535c927af9", "token-1")

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	var got struct {
		Inbounds []struct {
			Tag      string `json:"tag"`
			Listen   string `json:"listen"`
			Port     int    `json:"port"`
			Protocol string `json:"protocol"`
			Settings struct {
				Clients []struct {
					Password string `json:"password"`
					Email    string `json:"email"`
				} `json:"clients"`
			} `json:"settings"`
		} `json:"inbounds"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	if len(got.Inbounds) != 1 {
		t.Fatalf("expected one inbound, got %d", len(got.Inbounds))
	}
	inbound := got.Inbounds[0]
	if inbound.Tag != "trojan-in" || inbound.Listen != "0.0.0.0" || inbound.Port != 8443 || inbound.Protocol != "trojan" {
		t.Fatalf("unexpected inbound: %+v", inbound)
	}
	if len(inbound.Settings.Clients) != 1 {
		t.Fatalf("expected one client, got %d", len(inbound.Settings.Clients))
	}
	client := inbound.Settings.Clients[0]
	if client.Password != "7eb14c94-37e7-4a17-9af5-40535c927af9" || client.Email != "token-1" {
		t.Fatalf("unexpected client: %+v", client)
	}
}
