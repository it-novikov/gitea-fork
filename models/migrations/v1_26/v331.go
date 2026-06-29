// Copyright 2026 The KYBa Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_26

import (
	"context"

	"xorm.io/xorm"
)

type kcpRepositoryInterface struct {
	ID          int64  `xorm:"pk autoincr"`
	RepoID      int64  `xorm:"INDEX NOT NULL"`
	InterfaceID string `xorm:"UNIQUE NOT NULL"`
	Kind        string `xorm:"INDEX NOT NULL"`
	Version     string `xorm:"NOT NULL"`
	Visibility  string `xorm:"INDEX NOT NULL"`
	Manifest    string `xorm:"LONGTEXT"`
	CreatedUnix int64  `xorm:"INDEX created"`
	UpdatedUnix int64  `xorm:"INDEX updated"`
}

type kcpRepositoryInterfaceImport struct {
	ID                   int64  `xorm:"pk autoincr"`
	ConsumerRepoID       int64  `xorm:"INDEX NOT NULL"`
	InterfaceID          string `xorm:"INDEX NOT NULL"`
	RequiredVersionRange string `xorm:"NOT NULL"`
	Mode                 string `xorm:"INDEX NOT NULL"`
	MaterializedRevision string
	MaterializedDigest   string
	Freshness            string `xorm:"INDEX NOT NULL"`
	CreatedUnix          int64  `xorm:"INDEX created"`
	UpdatedUnix          int64  `xorm:"INDEX updated"`
}

type kcpCapsuleImpactTask struct {
	ID            int64  `xorm:"pk autoincr"`
	CapsuleID     string `xorm:"INDEX NOT NULL"`
	RepositoryID  int64  `xorm:"INDEX NOT NULL"`
	Policy        string `xorm:"INDEX NOT NULL"`
	Reason        string `xorm:"LONGTEXT"`
	DraftPRBranch string
	DraftPRTitle  string
	Blocked       bool   `xorm:"INDEX NOT NULL"`
	Status        string `xorm:"INDEX NOT NULL"`
	CreatedUnix   int64  `xorm:"INDEX created"`
	UpdatedUnix   int64  `xorm:"INDEX updated"`
}

type kcpArchiveExportRun struct {
	ID              int64  `xorm:"pk autoincr"`
	RunID           string `xorm:"UNIQUE NOT NULL"`
	Status          string `xorm:"INDEX NOT NULL"`
	OwnershipDigest string `xorm:"INDEX NOT NULL"`
	Plan            string `xorm:"LONGTEXT"`
	Artifacts       string `xorm:"LONGTEXT"`
	CreatedUnix     int64  `xorm:"INDEX created"`
	UpdatedUnix     int64  `xorm:"INDEX updated"`
}

// AddKYBaKCPRepositoryInterfaces creates the KYBa repository interface, context capsule,
// impact queue and archive export tables.
func AddKYBaKCPRepositoryInterfaces(ctx context.Context, x *xorm.Engine) error {
	return x.Sync2(
		new(kcpRepositoryInterface),
		new(kcpRepositoryInterfaceImport),
		new(kcpCapsuleImpactTask),
		new(kcpArchiveExportRun),
	)
}
