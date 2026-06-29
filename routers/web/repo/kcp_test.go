// Copyright 2026 The KYBa Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import "testing"

func TestRepoKCPExportRowsUseRepositoryTreeAndFolderSelection(t *testing.T) {
	entries := []repoKCPTreeEntry{
		{Path: "README.md"},
		{Path: "docs", IsDir: true},
		{Path: "docs/kcp/README.md"},
		{Path: "docs/kcp/WEB_UI_INTEGRATION.md"},
		{Path: "routers/web/repo/kcp.go"},
		{Path: ".kyba/imported-capsules/kyba.product.governance.v1/context.md"},
	}
	selection := repoKCPSelection{
		Submitted: true,
		Files:     map[string]struct{}{},
		Dirs:      map[string]struct{}{"docs": {}},
	}

	rows := repoKCPExportRows("gitea-fork", entries, selection)
	byPath := map[string]repoKCPFileRow{}
	for _, row := range rows {
		byPath[row.Path] = row
	}

	if _, ok := byPath["README.md"]; !ok {
		t.Fatalf("expected repository file to be exportable")
	}
	if _, ok := byPath["routers/web/repo/kcp.go"]; !ok {
		t.Fatalf("expected nested repository file to be exportable")
	}
	if _, ok := byPath[".kyba/imported-capsules/kyba.product.governance.v1/context.md"]; ok {
		t.Fatalf("imported capsule snapshots must not be re-exported")
	}

	docs := byPath["docs"]
	if !docs.IsDir || docs.FileCount != 2 || !docs.Selected {
		t.Fatalf("docs folder row not selected from folder selection: %#v", docs)
	}
	if !byPath["docs/kcp/README.md"].Selected || !byPath["docs/kcp/WEB_UI_INTEGRATION.md"].Selected {
		t.Fatalf("folder selection did not select descendant files: %#v", byPath)
	}
	if byPath["README.md"].Selected {
		t.Fatalf("folder selection leaked outside selected folder")
	}
}

func TestRepoKCPImportedRowsComeFromMaterializedCapsules(t *testing.T) {
	entries := []repoKCPTreeEntry{
		{Path: "README.md"},
		{Path: ".kyba/imported-capsules/README.md"},
		{Path: ".kyba/imported-capsules/kyba.product.governance.v1/context.md"},
		{Path: ".kyba/imported-capsules/kyba.product.governance.v1/repository-bridge.md"},
	}

	rows := repoKCPImportedRows("gitea-fork", entries)
	if len(rows) != 2 {
		t.Fatalf("expected two imported files, got %d: %#v", len(rows), rows)
	}
	for _, row := range rows {
		if row.CapsuleID != "kyba.product.governance.v1" {
			t.Fatalf("unexpected capsule id: %#v", row)
		}
		if row.SourceRepo != "kyba" {
			t.Fatalf("unexpected source repo: %#v", row)
		}
		if row.TargetRepo != "gitea-fork" {
			t.Fatalf("unexpected target repo: %#v", row)
		}
	}
}

func TestRepoKCPImpactRowsAreNotHardcoded(t *testing.T) {
	model := buildRepoKCPPageData("gitea-fork", "/79028418089/gitea-fork", "impact", []repoKCPTreeEntry{
		{Path: "README.md"},
	}, repoKCPSelection{})
	if len(model.Impact) != 0 {
		t.Fatalf("plain repository should not render demo impact rows: %#v", model.Impact)
	}

	model = buildRepoKCPPageData("gitea-fork", "/79028418089/gitea-fork", "impact", []repoKCPTreeEntry{
		{Path: ".kyba/imported-capsules/README.md"},
	}, repoKCPSelection{})
	if len(model.Impact) != 0 {
		t.Fatalf("imported-capsules README should not render impact rows: %#v", model.Impact)
	}

	model = buildRepoKCPPageData("gitea-fork", "/79028418089/gitea-fork", "impact", []repoKCPTreeEntry{
		{Path: ".kyba/imported-capsules/kyba.product.governance.v1/context.md"},
	}, repoKCPSelection{})
	if len(model.Impact) != 1 {
		t.Fatalf("expected imported capsule impact row, got %#v", model.Impact)
	}
	if model.Impact[0].CapsuleID != "kyba.product.governance.v1" || model.Impact[0].Status != "imported" {
		t.Fatalf("unexpected imported impact row: %#v", model.Impact[0])
	}
}
