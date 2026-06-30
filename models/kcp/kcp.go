// Copyright 2026 The KYBa Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package kcp

import (
	"context"
	"strings"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/timeutil"
)

type Permission string

const (
	PermissionRead         Permission = "kcp.read"
	PermissionExportWrite  Permission = "kcp.export.write"
	PermissionImportWrite  Permission = "kcp.import.write"
	PermissionImpactRead   Permission = "kcp.impact.read"
	PermissionImpactManage Permission = "kcp.impact.manage"
	PermissionAdmin        Permission = "kcp.admin"
)

type PermissionSubjectType string

const (
	PermissionSubjectUser PermissionSubjectType = "user"
	PermissionSubjectTeam PermissionSubjectType = "team"
	PermissionSubjectOrg  PermissionSubjectType = "org"
)

type RepositoryInterface struct {
	ID          int64              `xorm:"pk autoincr"`
	RepoID      int64              `xorm:"INDEX NOT NULL"`
	InterfaceID string             `xorm:"UNIQUE NOT NULL"`
	Kind        string             `xorm:"INDEX NOT NULL"`
	Version     string             `xorm:"NOT NULL"`
	Visibility  string             `xorm:"INDEX NOT NULL"`
	Manifest    string             `xorm:"LONGTEXT"`
	CreatedUnix timeutil.TimeStamp `xorm:"created INDEX"`
	UpdatedUnix timeutil.TimeStamp `xorm:"updated INDEX"`
}

func (*RepositoryInterface) TableName() string { return "kcp_repository_interface" }

type RepositoryInterfaceImport struct {
	ID                   int64  `xorm:"pk autoincr"`
	ConsumerRepoID       int64  `xorm:"INDEX NOT NULL"`
	InterfaceID          string `xorm:"INDEX NOT NULL"`
	RequiredVersionRange string `xorm:"NOT NULL"`
	Mode                 string `xorm:"INDEX NOT NULL"`
	MaterializedRevision string
	MaterializedDigest   string
	Freshness            string             `xorm:"INDEX NOT NULL"`
	CreatedUnix          timeutil.TimeStamp `xorm:"created INDEX"`
	UpdatedUnix          timeutil.TimeStamp `xorm:"updated INDEX"`
}

func (*RepositoryInterfaceImport) TableName() string { return "kcp_repository_interface_import" }

type CapsuleImpactTask struct {
	ID            int64  `xorm:"pk autoincr"`
	CapsuleID     string `xorm:"INDEX NOT NULL"`
	RepositoryID  int64  `xorm:"INDEX NOT NULL"`
	Policy        string `xorm:"INDEX NOT NULL"`
	Reason        string `xorm:"LONGTEXT"`
	DraftPRBranch string
	DraftPRTitle  string
	Blocked       bool               `xorm:"INDEX NOT NULL"`
	Status        string             `xorm:"INDEX NOT NULL"`
	CreatedUnix   timeutil.TimeStamp `xorm:"created INDEX"`
	UpdatedUnix   timeutil.TimeStamp `xorm:"updated INDEX"`
}

func (*CapsuleImpactTask) TableName() string { return "kcp_capsule_impact_task" }

type ArchiveExportRun struct {
	ID              int64              `xorm:"pk autoincr"`
	RunID           string             `xorm:"UNIQUE NOT NULL"`
	Status          string             `xorm:"INDEX NOT NULL"`
	OwnershipDigest string             `xorm:"INDEX NOT NULL"`
	Plan            string             `xorm:"LONGTEXT"`
	Artifacts       string             `xorm:"LONGTEXT"`
	CreatedUnix     timeutil.TimeStamp `xorm:"created INDEX"`
	UpdatedUnix     timeutil.TimeStamp `xorm:"updated INDEX"`
}

func (*ArchiveExportRun) TableName() string { return "kcp_archive_export_run" }

type RepositoryInterfaceFile struct {
	ID          int64              `xorm:"pk autoincr"`
	RepoID      int64              `xorm:"UNIQUE(repo_interface_file) INDEX NOT NULL"`
	InterfaceID string             `xorm:"UNIQUE(repo_interface_file) INDEX NOT NULL"`
	Path        string             `xorm:"UNIQUE(repo_interface_file) NOT NULL"`
	Mode        string             `xorm:"INDEX NOT NULL"`
	Selected    bool               `xorm:"INDEX NOT NULL"`
	CreatedUnix timeutil.TimeStamp `xorm:"created INDEX"`
	UpdatedUnix timeutil.TimeStamp `xorm:"updated INDEX"`
}

func (*RepositoryInterfaceFile) TableName() string { return "kcp_repository_interface_file" }

type PermissionGrant struct {
	ID          int64              `xorm:"pk autoincr"`
	RepoID      int64              `xorm:"UNIQUE(kcp_permission_grant) INDEX NOT NULL"`
	SubjectType string             `xorm:"UNIQUE(kcp_permission_grant) INDEX NOT NULL"`
	SubjectID   int64              `xorm:"UNIQUE(kcp_permission_grant) INDEX NOT NULL"`
	Permission  string             `xorm:"UNIQUE(kcp_permission_grant) INDEX NOT NULL"`
	CreatedByID int64              `xorm:"INDEX NOT NULL"`
	CreatedUnix timeutil.TimeStamp `xorm:"created INDEX"`
	UpdatedUnix timeutil.TimeStamp `xorm:"updated INDEX"`
}

func (*PermissionGrant) TableName() string { return "kcp_permission_grant" }

func init() {
	db.RegisterModel(new(RepositoryInterface))
	db.RegisterModel(new(RepositoryInterfaceImport))
	db.RegisterModel(new(CapsuleImpactTask))
	db.RegisterModel(new(ArchiveExportRun))
	db.RegisterModel(new(RepositoryInterfaceFile))
	db.RegisterModel(new(PermissionGrant))
}

type RepositoryInterfaceFileSpec struct {
	Path     string
	Mode     string
	Selected bool
}

func UpsertRepositoryInterface(ctx context.Context, item *RepositoryInterface) error {
	item.InterfaceID = strings.TrimSpace(item.InterfaceID)
	if item.InterfaceID == "" {
		return nil
	}
	has, err := db.GetEngine(ctx).Where("interface_id = ?", item.InterfaceID).Get(new(RepositoryInterface))
	if err != nil {
		return err
	}
	if has {
		_, err = db.GetEngine(ctx).Where("interface_id = ?", item.InterfaceID).Cols("repo_id", "kind", "version", "visibility", "manifest", "updated_unix").Update(item)
		return err
	}
	_, err = db.GetEngine(ctx).Insert(item)
	return err
}

func ReplaceRepositoryInterfaceFiles(ctx context.Context, repoID int64, interfaceID string, files []RepositoryInterfaceFileSpec) error {
	return db.WithTx(ctx, func(ctx context.Context) error {
		if _, err := db.GetEngine(ctx).Where("repo_id = ? AND interface_id = ?", repoID, interfaceID).Delete(new(RepositoryInterfaceFile)); err != nil {
			return err
		}
		if len(files) == 0 {
			return nil
		}
		beans := make([]*RepositoryInterfaceFile, 0, len(files))
		for _, file := range files {
			path := strings.TrimSpace(file.Path)
			if path == "" {
				continue
			}
			mode := strings.TrimSpace(file.Mode)
			if mode == "" {
				mode = "source"
			}
			beans = append(beans, &RepositoryInterfaceFile{RepoID: repoID, InterfaceID: interfaceID, Path: path, Mode: mode, Selected: file.Selected})
		}
		if len(beans) == 0 {
			return nil
		}
		_, err := db.GetEngine(ctx).Insert(beans)
		return err
	})
}

func ListRepositoryInterfaceFiles(ctx context.Context, repoID int64, interfaceID string) ([]*RepositoryInterfaceFile, error) {
	files := make([]*RepositoryInterfaceFile, 0)
	err := db.GetEngine(ctx).Where("repo_id = ? AND interface_id = ?", repoID, interfaceID).Asc("path").Find(&files)
	return files, err
}

func ListRepositoryInterfaces(ctx context.Context, repoID int64) ([]*RepositoryInterface, error) {
	interfaces := make([]*RepositoryInterface, 0)
	err := db.GetEngine(ctx).Where("repo_id = ?", repoID).Asc("interface_id").Find(&interfaces)
	return interfaces, err
}

func ListImportsForRepo(ctx context.Context, repoID int64) ([]*RepositoryInterfaceImport, error) {
	imports := make([]*RepositoryInterfaceImport, 0)
	err := db.GetEngine(ctx).Where("consumer_repo_id = ?", repoID).Asc("interface_id").Find(&imports)
	return imports, err
}

func UpsertImportsForRepo(ctx context.Context, repoID int64, imports []*RepositoryInterfaceImport) error {
	return db.WithTx(ctx, func(ctx context.Context) error {
		if _, err := db.GetEngine(ctx).Where("consumer_repo_id = ?", repoID).Delete(new(RepositoryInterfaceImport)); err != nil {
			return err
		}
		if len(imports) == 0 {
			return nil
		}
		_, err := db.GetEngine(ctx).Insert(imports)
		return err
	})
}

func ListImpactTasksForRepo(ctx context.Context, repoID int64) ([]*CapsuleImpactTask, error) {
	tasks := make([]*CapsuleImpactTask, 0)
	err := db.GetEngine(ctx).Where("repository_id = ?", repoID).Desc("updated_unix").Find(&tasks)
	return tasks, err
}

func ReplaceImpactTasksForRepo(ctx context.Context, repoID int64, tasks []*CapsuleImpactTask) error {
	return db.WithTx(ctx, func(ctx context.Context) error {
		if _, err := db.GetEngine(ctx).Where("repository_id = ?", repoID).Delete(new(CapsuleImpactTask)); err != nil {
			return err
		}
		if len(tasks) == 0 {
			return nil
		}
		_, err := db.GetEngine(ctx).Insert(tasks)
		return err
	})
}

func GrantRepoPermission(ctx context.Context, repoID int64, subjectType PermissionSubjectType, subjectID int64, permission Permission, createdByID int64) error {
	grant := &PermissionGrant{RepoID: repoID, SubjectType: string(subjectType), SubjectID: subjectID, Permission: string(permission), CreatedByID: createdByID}
	has, err := db.GetEngine(ctx).Where("repo_id = ? AND subject_type = ? AND subject_id = ? AND permission = ?", repoID, subjectType, subjectID, permission).Get(new(PermissionGrant))
	if err != nil || has {
		return err
	}
	_, err = db.GetEngine(ctx).Insert(grant)
	return err
}

func HasRepoPermissionGrant(ctx context.Context, repoID, userID int64, permission Permission) (bool, error) {
	if userID <= 0 {
		return false, nil
	}
	return db.GetEngine(ctx).Where(
		"repo_id = ? AND subject_type = ? AND subject_id = ? AND permission IN (?, ?)",
		repoID,
		string(PermissionSubjectUser),
		userID,
		string(permission),
		string(PermissionAdmin),
	).Exist(new(PermissionGrant))
}
