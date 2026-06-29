// Copyright 2026 The KYBa Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"code.gitea.io/gitea/modules/setting"
)

var ErrDisabled = errors.New("kyba identity service is disabled")

type Settings struct {
	Enabled               bool
	Mock                  bool
	BaseURL               string
	ClientID              string
	ClientSecret          string
	Timeout               time.Duration
	ChallengeStartPath    string
	ChallengeVerifyPath   string
	RegistrationStartPath string
	RegisterPath          string
	RecoveryStartPath     string
	AutoCreateUsers       bool
	RegistrationEnabled   bool
	RecoveryEnabled       bool
}

func LoadSettings() Settings {
	cfg := Settings{
		Timeout:               10 * time.Second,
		ChallengeStartPath:    "/v1/auth/challenges",
		ChallengeVerifyPath:   "/v1/auth/complete",
		RegistrationStartPath: "/v1/registration/challenges",
		RegisterPath:          "/v1/registration/complete",
		RecoveryStartPath:     "/v1/account-recovery/challenges",
		AutoCreateUsers:       true,
		RegistrationEnabled:   true,
		RecoveryEnabled:       true,
	}
	if setting.CfgProvider == nil {
		return cfg
	}
	sec := setting.CfgProvider.Section("kyba.identity")
	cfg.Enabled = sec.Key("ENABLED").MustBool(false)
	cfg.Mock = sec.Key("MOCK").MustBool(false)
	cfg.BaseURL = strings.TrimRight(sec.Key("BASE_URL").MustString(""), "/")
	cfg.ClientID = sec.Key("CLIENT_ID").MustString("gitea-fork")
	cfg.ClientSecret = sec.Key("CLIENT_SECRET").MustString("")
	cfg.ChallengeStartPath = sec.Key("CHALLENGE_START_PATH").MustString(cfg.ChallengeStartPath)
	cfg.ChallengeVerifyPath = sec.Key("CHALLENGE_VERIFY_PATH").MustString(cfg.ChallengeVerifyPath)
	cfg.RegistrationStartPath = sec.Key("REGISTRATION_START_PATH").MustString(cfg.RegistrationStartPath)
	cfg.RegisterPath = sec.Key("REGISTER_PATH").MustString(cfg.RegisterPath)
	cfg.RecoveryStartPath = sec.Key("RECOVERY_START_PATH").MustString(cfg.RecoveryStartPath)
	cfg.AutoCreateUsers = sec.Key("AUTO_CREATE_USERS").MustBool(cfg.AutoCreateUsers)
	cfg.RegistrationEnabled = sec.Key("REGISTRATION_ENABLED").MustBool(cfg.RegistrationEnabled)
	cfg.RecoveryEnabled = sec.Key("RECOVERY_ENABLED").MustBool(cfg.RecoveryEnabled)
	if timeout := sec.Key("TIMEOUT").MustString(""); timeout != "" {
		if parsed, err := time.ParseDuration(timeout); err == nil {
			cfg.Timeout = parsed
		}
	}
	return cfg
}

func Enabled() bool {
	return LoadSettings().Enabled
}

type Client struct {
	settings Settings
	http     *http.Client
}

func NewClient() Client {
	cfg := LoadSettings()
	return Client{settings: cfg, http: &http.Client{Timeout: cfg.Timeout}}
}

type Principal struct {
	Subject     string   `json:"subject"`
	Username    string   `json:"username"`
	DisplayName string   `json:"display_name"`
	Email       string   `json:"email"`
	Phone       string   `json:"phone"`
	Roles       []string `json:"roles"`
	Grants      []string `json:"grants"`
}

type ChallengeStartRequest struct {
	Phone      string `json:"phone"`
	Delivery   string `json:"delivery"`
	Channel    string `json:"channel,omitempty"`
	RedirectTo string `json:"redirect_to,omitempty"`
	ClientID   string `json:"client_id,omitempty"`
	UserAgent  string `json:"user_agent,omitempty"`
	RemoteAddr string `json:"remote_addr,omitempty"`
}

type ChallengeStartResponse struct {
	ChallengeID      string `json:"challenge_id"`
	MaskedTarget     string `json:"masked_target"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}

type ChallengeVerifyRequest struct {
	Phone       string `json:"phone,omitempty"`
	ChallengeID string `json:"challenge_id"`
	Channel     string `json:"channel,omitempty"`
	Code        string `json:"code"`
	Remember    bool   `json:"remember"`
}

type ChallengeVerifyResponse struct {
	Principal Principal `json:"principal"`
}

type RegisterRequest struct {
	DisplayName    string `json:"display_name"`
	Phone          string `json:"phone"`
	ChallengeID    string `json:"challenge_id,omitempty"`
	Code           string `json:"code,omitempty"`
	InvitationCode string `json:"invitation_code"`
	ClientID       string `json:"client_id,omitempty"`
}

type RegisterResponse struct {
	Principal Principal `json:"principal"`
}

type RecoveryStartRequest struct {
	Phone      string `json:"phone"`
	ClientID   string `json:"client_id,omitempty"`
	RemoteAddr string `json:"remote_addr,omitempty"`
	UserAgent  string `json:"user_agent,omitempty"`
}

type RecoveryStartResponse struct {
	ChallengeID      string `json:"challenge_id"`
	MaskedTarget     string `json:"masked_target"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}

type challengeDispatchResponse struct {
	ChallengeID          string `json:"challengeId"`
	Channel              string `json:"channel"`
	ExpiresAt            string `json:"expiresAt"`
	NextRequestAllowedAt string `json:"nextRequestAllowedAt"`
}

type userResponse struct {
	UserID        string   `json:"userId"`
	PhoneVerified bool     `json:"phoneVerified"`
	Roles         []string `json:"roles"`
}

func (c Client) StartChallenge(ctx context.Context, req ChallengeStartRequest) (ChallengeStartResponse, error) {
	if err := c.ensureEnabled(); err != nil {
		return ChallengeStartResponse{}, err
	}
	if c.settings.Mock {
		return ChallengeStartResponse{ChallengeID: "mock-challenge", MaskedTarget: maskPhone(req.Phone), ExpiresInSeconds: 90}, nil
	}
	input := struct {
		Phone   string `json:"phone"`
		Channel string `json:"channel"`
	}{
		Phone:   req.Phone,
		Channel: normalizeChannel(req.Delivery, req.Channel),
	}
	var out challengeDispatchResponse
	if err := c.post(ctx, c.settings.ChallengeStartPath, input, &out); err != nil {
		return ChallengeStartResponse{}, err
	}
	return ChallengeStartResponse{
		ChallengeID:      out.ChallengeID,
		MaskedTarget:     maskPhone(req.Phone),
		ExpiresInSeconds: 90,
	}, nil
}

func (c Client) VerifyChallenge(ctx context.Context, req ChallengeVerifyRequest) (ChallengeVerifyResponse, error) {
	if err := c.ensureEnabled(); err != nil {
		return ChallengeVerifyResponse{}, err
	}
	if c.settings.Mock {
		return ChallengeVerifyResponse{Principal: Principal{Subject: "mock-identity", Username: "kyba-user", DisplayName: "KYBa User", Email: "kyba-user@example.invalid", Phone: "+000000000"}}, nil
	}
	input := struct {
		Phone       string `json:"phone"`
		ChallengeID string `json:"challengeId"`
		Channel     string `json:"channel"`
		Code        string `json:"code"`
	}{
		Phone:       req.Phone,
		ChallengeID: req.ChallengeID,
		Channel:     normalizeChannel("", req.Channel),
		Code:        req.Code,
	}
	var out userResponse
	if err := c.post(ctx, c.settings.ChallengeVerifyPath, input, &out); err != nil {
		return ChallengeVerifyResponse{}, err
	}
	return ChallengeVerifyResponse{
		Principal: Principal{
			Subject:  out.UserID,
			Username: req.Phone,
			Phone:    req.Phone,
			Roles:    out.Roles,
		},
	}, nil
}

func (c Client) StartRegistration(ctx context.Context, req ChallengeStartRequest) (ChallengeStartResponse, error) {
	if err := c.ensureEnabled(); err != nil {
		return ChallengeStartResponse{}, err
	}
	if !c.settings.RegistrationEnabled {
		return ChallengeStartResponse{}, errors.New("kyba identity registration is disabled")
	}
	if c.settings.Mock {
		return ChallengeStartResponse{ChallengeID: "mock-registration", MaskedTarget: maskPhone(req.Phone), ExpiresInSeconds: 90}, nil
	}
	input := struct {
		Phone string `json:"phone"`
	}{
		Phone: req.Phone,
	}
	var out challengeDispatchResponse
	if err := c.post(ctx, c.settings.RegistrationStartPath, input, &out); err != nil {
		return ChallengeStartResponse{}, err
	}
	return ChallengeStartResponse{
		ChallengeID:      out.ChallengeID,
		MaskedTarget:     maskPhone(req.Phone),
		ExpiresInSeconds: 90,
	}, nil
}

func (c Client) Register(ctx context.Context, req RegisterRequest) (RegisterResponse, error) {
	if err := c.ensureEnabled(); err != nil {
		return RegisterResponse{}, err
	}
	if !c.settings.RegistrationEnabled {
		return RegisterResponse{}, errors.New("kyba identity registration is disabled")
	}
	if c.settings.Mock {
		return RegisterResponse{Principal: Principal{Subject: "mock-registered-identity", Username: req.DisplayName, DisplayName: req.DisplayName, Email: safeEmail(req.DisplayName), Phone: req.Phone}}, nil
	}
	input := struct {
		Phone       string `json:"phone"`
		ChallengeID string `json:"challengeId"`
		Code        string `json:"code"`
	}{
		Phone:       req.Phone,
		ChallengeID: req.ChallengeID,
		Code:        req.Code,
	}
	var out userResponse
	if err := c.post(ctx, c.settings.RegisterPath, input, &out); err != nil {
		return RegisterResponse{}, err
	}
	return RegisterResponse{
		Principal: Principal{
			Subject:     out.UserID,
			Username:    req.DisplayName,
			DisplayName: req.DisplayName,
			Phone:       req.Phone,
			Roles:       out.Roles,
		},
	}, nil
}

func (c Client) StartRecovery(ctx context.Context, req RecoveryStartRequest) (RecoveryStartResponse, error) {
	if err := c.ensureEnabled(); err != nil {
		return RecoveryStartResponse{}, err
	}
	if !c.settings.RecoveryEnabled {
		return RecoveryStartResponse{}, errors.New("kyba identity recovery is disabled")
	}
	if c.settings.Mock {
		return RecoveryStartResponse{ChallengeID: "mock-recovery", MaskedTarget: maskPhone(req.Phone), ExpiresInSeconds: 90}, nil
	}
	input := struct {
		Phone   string `json:"phone"`
		Channel string `json:"channel"`
	}{
		Phone:   req.Phone,
		Channel: "sms",
	}
	var out challengeDispatchResponse
	if err := c.post(ctx, c.settings.RecoveryStartPath, input, &out); err != nil {
		return RecoveryStartResponse{}, err
	}
	return RecoveryStartResponse{
		ChallengeID:      out.ChallengeID,
		MaskedTarget:     maskPhone(req.Phone),
		ExpiresInSeconds: 90,
	}, nil
}

func (c Client) ensureEnabled() error {
	if !c.settings.Enabled {
		return ErrDisabled
	}
	if !c.settings.Mock && c.settings.BaseURL == "" {
		return errors.New("kyba identity BASE_URL is required when MOCK=false")
	}
	return nil
}

func (c Client) post(ctx context.Context, path string, input, output any) error {
	body, err := json.Marshal(input)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.settings.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.settings.ClientID != "" {
		req.Header.Set("X-KYBa-Client-ID", c.settings.ClientID)
	}
	if c.settings.ClientSecret != "" {
		req.Header.Set("Authorization", "Bearer "+c.settings.ClientSecret)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("identity service returned %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(output)
}

func maskPhone(phone string) string {
	trimmed := strings.TrimSpace(phone)
	if len(trimmed) <= 4 {
		return "+•• ••• ••"
	}
	return "+•• ••• •• " + trimmed[len(trimmed)-2:]
}

func safeEmail(seed string) string {
	seed = strings.TrimSpace(strings.ToLower(seed))
	if seed == "" {
		seed = "kyba-user"
	}
	seed = strings.ReplaceAll(seed, " ", "-")
	return seed + "@example.invalid"
}

func normalizeChannel(delivery, channel string) string {
	value := strings.TrimSpace(strings.ToLower(channel))
	if value == "" {
		value = strings.TrimSpace(strings.ToLower(delivery))
	}
	switch value {
	case "push":
		return "push"
	default:
		return "sms"
	}
}
