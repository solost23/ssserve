package main

import (
	"testing"

	handlercmd "github.com/xtls/xray-core/app/proxyman/command"
	"github.com/xtls/xray-core/proxy/vless"
)

func TestBuildVLESSAddUserRequest(t *testing.T) {
	req := buildVLESSAddUserRequest("vless-in", "1a078af0-1bb6-498b-9896-4651db5cbaf4", "token-1")

	if req.Tag != "vless-in" {
		t.Fatalf("tag = %q, want vless-in", req.Tag)
	}
	msg, err := req.Operation.GetInstance()
	if err != nil {
		t.Fatalf("decode operation: %v", err)
	}
	op, ok := msg.(*handlercmd.AddUserOperation)
	if !ok {
		t.Fatalf("operation = %T, want *AddUserOperation", msg)
	}
	if op.User.GetEmail() != "token-1" {
		t.Fatalf("email = %q, want token-1", op.User.GetEmail())
	}

	accountMsg, err := op.User.GetAccount().GetInstance()
	if err != nil {
		t.Fatalf("decode account: %v", err)
	}
	account, ok := accountMsg.(*vless.Account)
	if !ok {
		t.Fatalf("account = %T, want *vless.Account", accountMsg)
	}
	if account.Id != "1a078af0-1bb6-498b-9896-4651db5cbaf4" {
		t.Fatalf("id = %q", account.Id)
	}
	if account.Flow != "xtls-rprx-vision" {
		t.Fatalf("flow = %q", account.Flow)
	}
	if account.Encryption != "none" {
		t.Fatalf("encryption = %q", account.Encryption)
	}
}

func TestBuildRemoveUserRequest(t *testing.T) {
	req := buildRemoveUserRequest("vless-in", "token-1")

	if req.Tag != "vless-in" {
		t.Fatalf("tag = %q, want vless-in", req.Tag)
	}
	msg, err := req.Operation.GetInstance()
	if err != nil {
		t.Fatalf("decode operation: %v", err)
	}
	op, ok := msg.(*handlercmd.RemoveUserOperation)
	if !ok {
		t.Fatalf("operation = %T, want *RemoveUserOperation", msg)
	}
	if op.Email != "token-1" {
		t.Fatalf("email = %q, want token-1", op.Email)
	}
}
