// Copyright 2026 The KYBa Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package identity

import (
	"strings"
	"testing"

	kcpidentity "code.gitea.io/gitea/modules/kcp/identity"
)

func TestShadowUserNameFallsBackToSubjectHash(t *testing.T) {
	name, err := shadowUserName(kcpidentity.Principal{Subject: "sub"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(name, "identity-") {
		t.Fatalf("expected identity fallback, got %s", name)
	}
}

func TestShadowUserNameNormalizesEmail(t *testing.T) {
	name, err := shadowUserName(kcpidentity.Principal{Email: "Ada.Lovelace@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if name != "Ada.Lovelace" {
		t.Fatalf("unexpected normalized name: %s", name)
	}
}
