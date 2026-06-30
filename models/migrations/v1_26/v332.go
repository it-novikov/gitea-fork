// Copyright 2026 The KYBa Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_26

import (
	"context"

	"xorm.io/xorm"
)

type kcpRepositoryInterfaceFile struct {
	ID          int64  `xorm:"pk autoincr"`
	RepoID      int64  `xorm:"UNIQUE(repo_interface_file) INDEX NOT NULL"`
	InterfaceID string `xorm:"UNIQUE(repo_interface_file) INDEX NOT NULL"`
	Path        string `xorm:"UNIQUE(repo_interface_file) NOT NULL"`
	Mode        string `xorm:"INDEX NOT NULL"`
	Selected    bool   `xorm:"INDEX NOT NULL"`
	CreatedUnix int64  `xorm:"INDEX created"`
	UpdatedUnix int64  `xorm:"INDEX updated"`
}

func (*kcpRepositoryInterfaceFile) TableName() string { return "kcp_repository_interface_file" }

type kcpPermissionGrant struct {
	ID          int64  `xorm:"pk autoincr"`
	RepoID      int64  `xorm:"UNIQUE(kcp_permission_grant) INDEX NOT NULL"`
	SubjectType string `xorm:"UNIQUE(kcp_permission_grant) INDEX NOT NULL"`
	SubjectID   int64  `xorm:"UNIQUE(kcp_permission_grant) INDEX NOT NULL"`
	Permission  string `xorm:"UNIQUE(kcp_permission_grant) INDEX NOT NULL"`
	CreatedByID int64  `xorm:"INDEX NOT NULL"`
	CreatedUnix int64  `xorm:"INDEX created"`
	UpdatedUnix int64  `xorm:"INDEX updated"`
}

func (*kcpPermissionGrant) TableName() string { return "kcp_permission_grant" }

// AddKYBaKCPRepositoryFileSelectionsAndPermissions creates repository-native
// KCP file selection persistence and explicit KCP permission grants.
func AddKYBaKCPRepositoryFileSelectionsAndPermissions(ctx context.Context, x *xorm.Engine) error {
	return x.Sync2(
		new(kcpRepositoryInterfaceFile),
		new(kcpPermissionGrant),
	)
}
