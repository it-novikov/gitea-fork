// Copyright 2026 The KYBa Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package identity

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMaskPhone(t *testing.T) {
	if got := maskPhone("+12345678944"); got != "+•• ••• •• 44" {
		t.Fatalf("unexpected masked phone: %s", got)
	}
}

func TestSafeEmail(t *testing.T) {
	if got := safeEmail("Ada Lovelace"); got != "ada-lovelace@example.invalid" {
		t.Fatalf("unexpected safe email: %s", got)
	}
}

func TestRegisterSendsInvitationAndClientID(t *testing.T) {
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/register" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"userId":"id-1","phoneVerified":true,"roles":["member"]}`))
	}))
	defer server.Close()

	client := Client{
		settings: Settings{
			Enabled:             true,
			BaseURL:             server.URL,
			RegisterPath:        "/register",
			RegistrationEnabled: true,
			Timeout:             time.Second,
		},
		http: server.Client(),
	}
	_, err := client.Register(t.Context(), RegisterRequest{
		DisplayName:    "Ada Lovelace",
		Phone:          "+12345678944",
		ChallengeID:    "challenge-1",
		Code:           "123456",
		InvitationCode: "invite-1",
		ClientID:       "gitea-fork",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	for key, want := range map[string]string{
		"phone":          "+12345678944",
		"displayName":    "Ada Lovelace",
		"challengeId":    "challenge-1",
		"code":           "123456",
		"invitationCode": "invite-1",
		"clientId":       "gitea-fork",
	} {
		if got, _ := payload[key].(string); got != want {
			t.Fatalf("payload[%s]=%q, want %q", key, got, want)
		}
	}
}
