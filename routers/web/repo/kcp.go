// Copyright 2026 The KYBa Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"net/http"
	"sort"

	"code.gitea.io/gitea/modules/templates"
	"code.gitea.io/gitea/services/context"
)

const tplRepoKCPPage templates.TplName = "repo/kcp/page"

type repoKCPNavItem struct {
	Key    string
	Title  string
	Href   string
	Active bool
}

type repoKCPFileRow struct {
	Path        string
	CapsuleID   string
	Mode        string
	SourceRepo  string
	TargetRepo  string
	Selected    bool
	Description string
}

type repoKCPCapsuleRow struct {
	ID          string
	Kind        string
	Version     string
	OwnerRepo   string
	Visibility  string
	Description string
	Files       []repoKCPFileRow
}

type repoKCPImpactRow struct {
	CapsuleID  string
	Repository string
	Status     string
	Task       string
	DraftPR    string
}

type repoKCPPageData struct {
	SubPage       string
	Nav           []repoKCPNavItem
	Exports       []repoKCPCapsuleRow
	Imports       []repoKCPCapsuleRow
	ExportFiles   []repoKCPFileRow
	ImportedFiles []repoKCPFileRow
	Impact        []repoKCPImpactRow
	SelectedCount int
	HelpText      string
}

// KCPRepoOverview renders the repository-native KYBa KCP overview.
func KCPRepoOverview(ctx *context.Context) { renderRepoKCP(ctx, "overview") }

// KCPRepoImports renders materialized imported capsule files in repository context.
func KCPRepoImports(ctx *context.Context) { renderRepoKCP(ctx, "imports") }

// KCPRepoExports renders exported repository interfaces and capsule file selection.
func KCPRepoExports(ctx *context.Context) { renderRepoKCP(ctx, "exports") }

// KCPRepoImpact renders impact tasks and generated draft PRs for this repository.
func KCPRepoImpact(ctx *context.Context) { renderRepoKCP(ctx, "impact") }

// KCPRepoExportFilesPost handles the native repository export-file selection form.
// Persistence is intentionally left to the Gitea model/service integration slice;
// this handler proves the repository-native UI and request contract.
func KCPRepoExportFilesPost(ctx *context.Context) {
	ctx.Flash.Success("KYBa KCP export file selection accepted for preview. Persisted storage is handled by the KCP repository-interface service slice.")
	renderRepoKCP(ctx, "exports")
}

func renderRepoKCP(ctx *context.Context, subPage string) {
	model := buildRepoKCPPageData(ctx.Repo.Repository.Name, ctx.Repo.RepoLink, subPage)
	ctx.Data["Title"] = "KYBa KCP - " + ctx.Repo.Repository.FullName()
	ctx.Data["PageIsRepoKCP"] = true
	ctx.Data["KCPRepo"] = model
	ctx.HTML(http.StatusOK, tplRepoKCPPage)
}

func buildRepoKCPPageData(repoName, repoLink, subPage string) repoKCPPageData {
	if subPage == "" {
		subPage = "overview"
	}
	base := repoLink + "/kcp"
	model := repoKCPPageData{SubPage: subPage}
	model.Nav = []repoKCPNavItem{
		{Key: "overview", Title: "Overview", Href: base},
		{Key: "imports", Title: "Imported files", Href: base + "/imports"},
		{Key: "exports", Title: "Exported files", Href: base + "/exports"},
		{Key: "impact", Title: "Impact", Href: base + "/impact"},
	}
	for i := range model.Nav {
		model.Nav[i].Active = model.Nav[i].Key == subPage
	}

	model.Exports = repoKCPDemoExports(repoName)
	model.Imports = repoKCPDemoImports(repoName)
	for _, capsule := range model.Exports {
		model.ExportFiles = append(model.ExportFiles, capsule.Files...)
	}
	for _, capsule := range model.Imports {
		model.ImportedFiles = append(model.ImportedFiles, capsule.Files...)
	}
	sort.Slice(model.ExportFiles, func(i, j int) bool { return model.ExportFiles[i].Path < model.ExportFiles[j].Path })
	sort.Slice(model.ImportedFiles, func(i, j int) bool { return model.ImportedFiles[i].Path < model.ImportedFiles[j].Path })
	for _, file := range model.ExportFiles {
		if file.Selected {
			model.SelectedCount++
		}
	}
	model.Impact = []repoKCPImpactRow{
		{CapsuleID: "kyba.backend.task-workflow.api.v1", Repository: "kyba-desktop", Status: "compatible", Task: "Maintain imported backend API capsule", DraftPR: "Update generated desktop client"},
		{CapsuleID: "kyba.product.task-workspace.v1", Repository: repoName, Status: "context-only", Task: "Refresh materialized context capsule", DraftPR: "Refresh .kyba/imported-capsules snapshot"},
	}
	model.HelpText = "Repository KCP is intentionally embedded in the native repository UI. Imports, exports, file selection and impact are scoped to the current repository so agents do not need broad sibling-repository access."
	return model
}

func repoKCPDemoExports(repoName string) []repoKCPCapsuleRow {
	switch repoName {
	case "kyba-backend":
		return []repoKCPCapsuleRow{{ID: "kyba.backend.task-workflow.api.v1", Kind: "api-capsule", Version: "1.4.0", OwnerRepo: "kyba-backend", Visibility: "imported-repos-only", Description: "Backend API and context exported to dependent clients.", Files: []repoKCPFileRow{
			{Path: "proto/kyba/task_workflow/v1/task_workflow.proto", CapsuleID: "kyba.backend.task-workflow.api.v1", Mode: "contract", SourceRepo: "kyba-backend", TargetRepo: "kyba-desktop", Selected: true, Description: "Canonical task workflow service contract."},
			{Path: "docs/context-capsules/task-workflow.md", CapsuleID: "kyba.backend.task-workflow.api.v1", Mode: "context", SourceRepo: "kyba-backend", TargetRepo: "kyba-desktop", Selected: true, Description: "Agent-readable backend context capsule."},
			{Path: "generated/typescript/task-workflow", CapsuleID: "kyba.backend.task-workflow.api.v1", Mode: "generated", SourceRepo: "kyba-backend", TargetRepo: "kyba-desktop", Selected: true, Description: "Generated TypeScript client output."},
		}}}
	case "kyba":
		return []repoKCPCapsuleRow{{ID: "kyba.product.governance.v1", Kind: "product-capsule", Version: "1.0.0", OwnerRepo: "kyba", Visibility: "imported-repos-only", Description: "Project truth, roadmap and source-of-truth rules for child repositories.", Files: []repoKCPFileRow{
			{Path: "docs/knowledge-base/source-of-truth.md", CapsuleID: "kyba.product.governance.v1", Mode: "context", SourceRepo: "kyba", TargetRepo: "all", Selected: true, Description: "Canonical source-of-truth policy."},
			{Path: "docs/roadmap/repository-bridge-map.md", CapsuleID: "kyba.product.governance.v1", Mode: "context", SourceRepo: "kyba", TargetRepo: "all", Selected: true, Description: "Repository bridge and ownership rules."},
		}}}
	default:
		return []repoKCPCapsuleRow{{ID: "kyba.repo." + repoName + ".status.v1", Kind: "repository-status-capsule", Version: "0.1.0", OwnerRepo: repoName, Visibility: "imported-repos-only", Description: "Repository-local implementation/status capsule.", Files: []repoKCPFileRow{
			{Path: ".kyba/repository-interface.yaml", CapsuleID: "kyba.repo." + repoName + ".status.v1", Mode: "interface", SourceRepo: repoName, TargetRepo: "kyba", Selected: true, Description: "Repository exported interface manifest."},
			{Path: "VALIDATION.md", CapsuleID: "kyba.repo." + repoName + ".status.v1", Mode: "validation", SourceRepo: repoName, TargetRepo: "kyba-ci", Selected: true, Description: "Repository-local validation contract."},
		}}}
	}
}

func repoKCPDemoImports(repoName string) []repoKCPCapsuleRow {
	imports := []repoKCPCapsuleRow{
		{
			ID:          "kyba.product.governance.v1",
			Kind:        "product-capsule",
			Version:     "1.0.0",
			OwnerRepo:   "kyba",
			Visibility:  "materialized",
			Description: "Imported project governance context.",
			Files: []repoKCPFileRow{
				{Path: ".kyba/imported-capsules/kyba.product.governance.v1/context.md", CapsuleID: "kyba.product.governance.v1", Mode: "context", SourceRepo: "kyba", TargetRepo: repoName, Description: "Bounded product and source-of-truth context."},
				{Path: ".kyba/imported-capsules/kyba.product.governance.v1/repository-bridge.md", CapsuleID: "kyba.product.governance.v1", Mode: "context", SourceRepo: "kyba", TargetRepo: repoName, Description: "Repository ownership and dependency map."},
			},
		},
	}
	if repoName == "kyba-desktop" || repoName == "kyba-ci" {
		imports = append(imports, repoKCPCapsuleRow{
			ID:          "kyba.backend.task-workflow.api.v1",
			Kind:        "api-capsule",
			Version:     "1.4.0",
			OwnerRepo:   "kyba-backend",
			Visibility:  "materialized",
			Description: "Imported backend API capsule.",
			Files: []repoKCPFileRow{
				{Path: ".kyba/imported-capsules/kyba.backend.task-workflow.api.v1/proto/task_workflow.proto", CapsuleID: "kyba.backend.task-workflow.api.v1", Mode: "contract", SourceRepo: "kyba-backend", TargetRepo: repoName, Description: "Materialized backend proto contract."},
				{Path: ".kyba/imported-capsules/kyba.backend.task-workflow.api.v1/generated/typescript", CapsuleID: "kyba.backend.task-workflow.api.v1", Mode: "generated", SourceRepo: "kyba-backend", TargetRepo: repoName, Description: "Generated TypeScript client snapshot."},
			},
		})
	}
	return imports
}
