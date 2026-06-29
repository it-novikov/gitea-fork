// Copyright 2026 The KYBa Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package identity

import "testing"

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
