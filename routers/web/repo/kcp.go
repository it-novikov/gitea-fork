// Copyright 2026 The KYBa Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

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
	Partial     bool
	IsDir       bool
	Depth       int
	FileCount   int
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

type repoKCPTreeEntry struct {
	Path  string
	IsDir bool
}

type repoKCPSelection struct {
	Submitted bool
	Files     map[string]struct{}
	Dirs      map[string]struct{}
}

// KCPRepoOverview renders the repository-native KYBa KCP overview.
func KCPRepoOverview(ctx *context.Context) { renderRepoKCP(ctx, "overview") }

// KCPRepoImports renders materialized imported capsule files in repository context.
func KCPRepoImports(ctx *context.Context) { renderRepoKCP(ctx, "imports") }

// KCPRepoExports renders exported repository interfaces and capsule file selection.
func KCPRepoExports(ctx *context.Context) { renderRepoKCP(ctx, "exports") }

// KCPRepoImpact renders impact tasks and generated draft PRs for this repository.
func KCPRepoImpact(ctx *context.Context) { renderRepoKCP(ctx, "impact") }

// KCPRepoExportFilesPost handles the native repository export-file selection form
// and renders the submitted preview back into the repository UI.
func KCPRepoExportFilesPost(ctx *context.Context) {
	selection := repoKCPSelectionFromForm(ctx)
	ctx.Flash.Success("KYBa KCP export selection preview updated.")
	renderRepoKCPWithSelection(ctx, "exports", selection)
}

func renderRepoKCP(ctx *context.Context, subPage string) {
	renderRepoKCPWithSelection(ctx, subPage, repoKCPSelection{})
}

func renderRepoKCPWithSelection(ctx *context.Context, subPage string, selection repoKCPSelection) {
	treeEntries, err := repoKCPTreeEntries(ctx)
	if err != nil {
		ctx.ServerError("repoKCPTreeEntries", err)
		return
	}
	model := buildRepoKCPPageData(ctx.Repo.Repository.Name, ctx.Repo.RepoLink, subPage, treeEntries, selection)
	ctx.Data["Title"] = "KYBa KCP - " + ctx.Repo.Repository.FullName()
	ctx.Data["PageIsRepoKCP"] = true
	ctx.Data["KCPRepo"] = model
	ctx.HTML(http.StatusOK, tplRepoKCPPage)
}

func buildRepoKCPPageData(repoName, repoLink, subPage string, treeEntries []repoKCPTreeEntry, selection repoKCPSelection) repoKCPPageData {
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

	model.ImportedFiles = repoKCPImportedRows(repoName, treeEntries)
	model.Imports = repoKCPGroupCapsuleRows(model.ImportedFiles)
	model.ExportFiles = repoKCPExportRows(repoName, treeEntries, selection)
	model.Exports = repoKCPGroupCapsuleRows(model.ExportFiles)
	for _, file := range model.ExportFiles {
		if !file.IsDir && file.Selected {
			model.SelectedCount++
		}
	}
	model.Impact = repoKCPImpactRows(repoName, model.ImportedFiles, model.ExportFiles)
	model.HelpText = "Repository KCP is intentionally embedded in the native repository UI. Imports, exports, file selection and impact are scoped to the current repository so agents do not need broad sibling-repository access."
	return model
}

func repoKCPSelectionFromForm(ctx *context.Context) repoKCPSelection {
	selection := repoKCPSelection{Submitted: true, Files: map[string]struct{}{}, Dirs: map[string]struct{}{}}
	for _, value := range ctx.FormStrings("export_files") {
		if path := repoKCPNormalizePath(value); path != "" {
			selection.Files[path] = struct{}{}
		}
	}
	for _, value := range ctx.FormStrings("export_dirs") {
		if path := repoKCPNormalizePath(value); path != "" {
			selection.Dirs[path] = struct{}{}
		}
	}
	return selection
}

func repoKCPTreeEntries(ctx *context.Context) ([]repoKCPTreeEntry, error) {
	if ctx.Repo.Repository.IsEmpty || ctx.Repo.Repository.IsBroken() {
		return nil, nil
	}
	commit := ctx.Repo.Commit
	if commit == nil && ctx.Repo.GitRepo != nil {
		var err error
		commit, err = ctx.Repo.GitRepo.GetBranchCommit(ctx.Repo.Repository.DefaultBranch)
		if err != nil {
			return nil, err
		}
	}
	if commit == nil {
		return nil, nil
	}
	entries, err := commit.Tree.ListEntriesRecursiveFast()
	if err != nil {
		return nil, err
	}
	rows := make([]repoKCPTreeEntry, 0, len(entries))
	for _, entry := range entries {
		path := repoKCPNormalizePath(entry.Name())
		if path == "" {
			continue
		}
		rows = append(rows, repoKCPTreeEntry{Path: path, IsDir: entry.IsDir()})
	}
	return rows, nil
}

func repoKCPExportRows(repoName string, entries []repoKCPTreeEntry, selection repoKCPSelection) []repoKCPFileRow {
	files, folders := repoKCPFilesAndFolders(entries, false)
	capsuleID := "kyba.repo." + repoName + ".interface.v1"
	rows := make([]repoKCPFileRow, 0, len(files)+len(folders))
	for _, folder := range folders {
		count := countDescendantFiles(folder, files)
		if count == 0 {
			continue
		}
		rows = append(rows, repoKCPFileRow{
			Path:        folder,
			CapsuleID:   capsuleID,
			Mode:        "folder",
			SourceRepo:  repoName,
			TargetRepo:  "repository-interface",
			IsDir:       true,
			Depth:       repoKCPPathDepth(folder),
			FileCount:   count,
			Description: fmt.Sprintf("Folder export selector covering %d repository files.", count),
		})
	}
	for _, file := range files {
		rows = append(rows, repoKCPFileRow{
			Path:        file,
			CapsuleID:   capsuleID,
			Mode:        repoKCPModeForPath(file, false),
			SourceRepo:  repoName,
			TargetRepo:  "repository-interface",
			Selected:    repoKCPFileSelected(file, selection),
			Depth:       repoKCPPathDepth(file),
			Description: "Repository file available for bounded KCP export.",
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Path == rows[j].Path {
			return rows[i].IsDir && !rows[j].IsDir
		}
		return rows[i].Path < rows[j].Path
	})
	repoKCPApplyFolderSelection(rows)
	return rows
}

func repoKCPImportedRows(repoName string, entries []repoKCPTreeEntry) []repoKCPFileRow {
	files, _ := repoKCPFilesAndFolders(entries, true)
	rows := make([]repoKCPFileRow, 0, len(files))
	for _, file := range files {
		capsuleID := repoKCPCapsuleIDFromImportedPath(file)
		if capsuleID == "" {
			continue
		}
		rows = append(rows, repoKCPFileRow{
			Path:        file,
			CapsuleID:   capsuleID,
			Mode:        repoKCPModeForPath(file, true),
			SourceRepo:  repoKCPSourceRepoFromCapsuleID(capsuleID),
			TargetRepo:  repoName,
			Depth:       repoKCPPathDepth(file),
			Description: "Materialized imported capsule file visible inside this repository.",
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Path < rows[j].Path })
	return rows
}

func repoKCPFilesAndFolders(entries []repoKCPTreeEntry, importedOnly bool) ([]string, []string) {
	fileSet := map[string]struct{}{}
	folderSet := map[string]struct{}{}
	for _, entry := range entries {
		path := repoKCPNormalizePath(entry.Path)
		if path == "" {
			continue
		}
		imported := repoKCPIsImportedPath(path)
		if importedOnly != imported {
			continue
		}
		if entry.IsDir {
			folderSet[path] = struct{}{}
			continue
		}
		fileSet[path] = struct{}{}
		for parent := repoKCPParentPath(path); parent != ""; parent = repoKCPParentPath(parent) {
			if importedOnly || !repoKCPIsImportedPath(parent) {
				folderSet[parent] = struct{}{}
			}
		}
	}
	files := repoKCPSortedKeys(fileSet)
	folders := repoKCPSortedKeys(folderSet)
	return files, folders
}

func repoKCPGroupCapsuleRows(files []repoKCPFileRow) []repoKCPCapsuleRow {
	grouped := map[string]*repoKCPCapsuleRow{}
	for _, file := range files {
		row := grouped[file.CapsuleID]
		if row == nil {
			row = &repoKCPCapsuleRow{
				ID:          file.CapsuleID,
				Kind:        repoKCPKindForCapsule(file.CapsuleID),
				Version:     "working-tree",
				OwnerRepo:   file.SourceRepo,
				Visibility:  "repository-scoped",
				Description: "Repository-scoped KCP capsule derived from current Git tree.",
			}
			grouped[file.CapsuleID] = row
		}
		row.Files = append(row.Files, file)
	}
	capsules := make([]repoKCPCapsuleRow, 0, len(grouped))
	for _, row := range grouped {
		capsules = append(capsules, *row)
	}
	sort.Slice(capsules, func(i, j int) bool { return capsules[i].ID < capsules[j].ID })
	return capsules
}

func repoKCPImpactRows(repoName string, importedRows, exportRows []repoKCPFileRow) []repoKCPImpactRow {
	rows := make([]repoKCPImpactRow, 0)
	importCounts := map[string]int{}
	for _, row := range importedRows {
		importCounts[row.CapsuleID]++
	}
	for _, capsuleID := range repoKCPSortedIntKeys(importCounts) {
		rows = append(rows, repoKCPImpactRow{
			CapsuleID:  capsuleID,
			Repository: repoName,
			Status:     "imported",
			Task:       fmt.Sprintf("Refresh %d materialized imported files", importCounts[capsuleID]),
			DraftPR:    "Refresh .kyba/imported-capsules/" + capsuleID,
		})
	}
	selected := 0
	for _, row := range exportRows {
		if !row.IsDir && row.Selected {
			selected++
		}
	}
	if selected > 0 {
		capsuleID := "kyba.repo." + repoName + ".interface.v1"
		rows = append(rows, repoKCPImpactRow{
			CapsuleID:  capsuleID,
			Repository: "declared consumers",
			Status:     "export-preview",
			Task:       fmt.Sprintf("Review %d selected exported files", selected),
			DraftPR:    "Persist repository-interface selection",
		})
	}
	return rows
}

func repoKCPApplyFolderSelection(rows []repoKCPFileRow) {
	for i := range rows {
		if !rows[i].IsDir {
			continue
		}
		total := 0
		selected := 0
		for _, candidate := range rows {
			if candidate.IsDir || !repoKCPPathUnder(candidate.Path, rows[i].Path) {
				continue
			}
			total++
			if candidate.Selected {
				selected++
			}
		}
		rows[i].Selected = total > 0 && total == selected
		rows[i].Partial = selected > 0 && selected < total
	}
}

func repoKCPFileSelected(path string, selection repoKCPSelection) bool {
	if !selection.Submitted {
		return false
	}
	if _, ok := selection.Files[path]; ok {
		return true
	}
	for dir := range selection.Dirs {
		if repoKCPPathUnder(path, dir) {
			return true
		}
	}
	return false
}

func repoKCPModeForPath(path string, imported bool) string {
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, "repository-interface.yaml") || strings.HasSuffix(lower, "repository-interface.yml"):
		return "interface"
	case strings.Contains(lower, "/generated/") || strings.Contains(lower, "/dist/"):
		return "generated"
	case strings.HasSuffix(lower, ".proto") || strings.HasSuffix(lower, ".openapi.yaml") || strings.HasSuffix(lower, ".openapi.yml") || strings.HasSuffix(lower, ".graphql"):
		return "contract"
	case strings.HasSuffix(lower, "_test.go") || strings.HasSuffix(lower, ".test.ts") || strings.HasSuffix(lower, ".test.tsx") || strings.HasSuffix(lower, ".spec.ts") || strings.HasSuffix(lower, ".spec.tsx"):
		return "test"
	case strings.HasPrefix(lower, "docs/") || strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".mdx"):
		if imported {
			return "context"
		}
		return "docs"
	case strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml") || strings.HasSuffix(lower, ".json") || strings.HasSuffix(lower, ".toml") || strings.HasSuffix(lower, ".ini"):
		return "config"
	default:
		return "source"
	}
}

func repoKCPKindForCapsule(capsuleID string) string {
	if strings.Contains(capsuleID, ".repo.") || strings.Contains(capsuleID, ".interface.") {
		return "repository-interface"
	}
	if strings.Contains(capsuleID, ".api.") {
		return "api-capsule"
	}
	return "context-capsule"
}

func repoKCPSourceRepoFromCapsuleID(capsuleID string) string {
	parts := strings.Split(capsuleID, ".")
	if len(parts) < 2 {
		return "unknown"
	}
	switch parts[1] {
	case "backend":
		return "kyba-backend"
	case "product":
		return "kyba"
	case "repo":
		if len(parts) >= 3 {
			return parts[2]
		}
	}
	return parts[1]
}

func repoKCPCapsuleIDFromImportedPath(path string) string {
	rest := strings.TrimPrefix(path, ".kyba/imported-capsules/")
	if rest == path || rest == "" {
		return ""
	}
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) < 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return ""
	}
	return parts[0]
}

func repoKCPIsImportedPath(path string) bool {
	return path == ".kyba/imported-capsules" || strings.HasPrefix(path, ".kyba/imported-capsules/")
}

func repoKCPNormalizePath(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	value = strings.TrimPrefix(value, "./")
	value = strings.Trim(value, "/")
	return value
}

func repoKCPParentPath(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx <= 0 {
		return ""
	}
	return path[:idx]
}

func repoKCPPathDepth(path string) int {
	if path == "" {
		return 0
	}
	return strings.Count(path, "/")
}

func repoKCPPathUnder(path, dir string) bool {
	return path != dir && strings.HasPrefix(path, dir+"/")
}

func countDescendantFiles(dir string, files []string) int {
	count := 0
	for _, file := range files {
		if repoKCPPathUnder(file, dir) {
			count++
		}
	}
	return count
}

func repoKCPSortedKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func repoKCPSortedIntKeys(values map[string]int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
