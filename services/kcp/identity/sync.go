// Copyright 2026 The KYBa Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package identity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	auth_model "code.gitea.io/gitea/models/auth"
	user_model "code.gitea.io/gitea/models/user"
	kcpidentity "code.gitea.io/gitea/modules/kcp/identity"
	"code.gitea.io/gitea/modules/optional"
	user_service "code.gitea.io/gitea/services/user"
)

// EnsureShadowUser maps a KYBa identity-service principal to a local Gitea shadow user.
// The identity service remains the authentication source of truth; the local user exists
// only so normal Gitea sessions, repository permissions and UI rendering continue to work.
func EnsureShadowUser(ctx context.Context, p kcpidentity.Principal, remoteAddr, userAgent string) (*user_model.User, error) {
	name, err := shadowUserName(p)
	if err != nil {
		return nil, err
	}
	if u, err := user_model.GetIndividualUserByName(ctx, name); err == nil {
		return syncShadowUser(ctx, u, p)
	} else if !user_model.IsErrUserNotExist(err) {
		return nil, err
	}

	email := p.Email
	if strings.TrimSpace(email) == "" {
		email = fmt.Sprintf("%s@identity.local", name)
	}
	fullName := p.DisplayName
	if fullName == "" {
		fullName = name
	}

	u := &user_model.User{
		Name:        name,
		FullName:    fullName,
		Email:       email,
		LoginType:   auth_model.Plain,
		LoginSource: 0,
		Passwd:      disabledLocalPassword(p),
		IsActive:    true,
	}
	meta := &user_model.Meta{InitialIP: remoteAddr, InitialUserAgent: userAgent}
	overwrite := &user_model.CreateUserOverwriteOptions{IsActive: optional.Some(true)}
	if err := user_model.CreateUser(ctx, u, meta, overwrite); err != nil {
		// Race-safe fallback if another request created the same shadow user.
		if user_model.IsErrUserAlreadyExist(err) {
			return user_model.GetIndividualUserByName(ctx, name)
		}
		// Identity Service may return an email that already belongs to a shadow user.
		if user_model.IsErrEmailAlreadyUsed(err) && email != "" {
			if existing, getErr := user_model.GetUserByEmail(ctx, email); getErr == nil {
				return syncShadowUser(ctx, existing, p)
			}
		}
		return nil, err
	}
	return syncShadowUser(ctx, u, p)
}

func syncShadowUser(ctx context.Context, u *user_model.User, p kcpidentity.Principal) (*user_model.User, error) {
	opts := &user_service.UpdateOptions{IsActive: optional.Some(true), SetLastLogin: true}
	if p.DisplayName != "" && p.DisplayName != u.FullName {
		opts.FullName = optional.Some(p.DisplayName)
	}
	if err := user_service.UpdateUser(ctx, u, opts); err != nil {
		return nil, err
	}
	return u, nil
}

func shadowUserName(p kcpidentity.Principal) (string, error) {
	candidate := firstNonEmpty(p.Username, p.Email, p.Phone, p.Subject)
	if candidate == "" {
		return "", fmt.Errorf("identity principal does not contain username, email, phone or subject")
	}
	name, err := user_model.NormalizeUserName(candidate)
	if err != nil {
		return "", err
	}
	name = strings.Trim(name, "-")
	if name == "" || len(name) < 3 {
		digest := sha256.Sum256([]byte(firstNonEmpty(p.Subject, p.Phone, p.Email, p.Username)))
		name = "identity-" + hex.EncodeToString(digest[:])[:12]
	}
	return name, nil
}

func disabledLocalPassword(p kcpidentity.Principal) string {
	seed := firstNonEmpty(p.Subject, p.Phone, p.Email, p.Username)
	if seed == "" {
		seed = "kyba-identity-shadow-user"
	}
	digest := sha256.Sum256([]byte("kyba-identity-shadow-user:" + seed))
	return hex.EncodeToString(digest[:])
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
