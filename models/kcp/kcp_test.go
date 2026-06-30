// Copyright 2026 The KYBa Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package kcp

import "testing"

func TestKCPModelTableNames(t *testing.T) {
	cases := map[string]string{
		new(RepositoryInterface).TableName():       "kcp_repository_interface",
		new(RepositoryInterfaceImport).TableName(): "kcp_repository_interface_import",
		new(CapsuleImpactTask).TableName():         "kcp_capsule_impact_task",
		new(ArchiveExportRun).TableName():          "kcp_archive_export_run",
		new(RepositoryInterfaceFile).TableName():   "kcp_repository_interface_file",
		new(PermissionGrant).TableName():           "kcp_permission_grant",
	}
	for actual, expected := range cases {
		if actual != expected {
			t.Fatalf("unexpected table name: got %s want %s", actual, expected)
		}
	}
}

func TestKCPPermissionNames(t *testing.T) {
	for _, permission := range []Permission{PermissionRead, PermissionExportWrite, PermissionImportWrite, PermissionImpactRead, PermissionImpactManage, PermissionAdmin} {
		if permission == "" {
			t.Fatalf("permission constant must not be empty")
		}
	}
}
