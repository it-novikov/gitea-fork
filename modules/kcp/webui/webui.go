package webui

import (
	"bytes"
	"embed"
	"html/template"
	"sort"
	"time"

	"code.gitea.io/gitea/modules/kcp/capsules"
	"code.gitea.io/gitea/modules/kcp/exporter"
	kcpservice "code.gitea.io/gitea/services/kcp"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

type Page string

const (
	PageDashboard Page = "dashboard"
	PageCapsules  Page = "capsules"
	PageImports   Page = "imports"
	PageImpact    Page = "impact"
	PageExport    Page = "export"
)

type NavItem struct {
	Page   Page
	Title  string
	Active bool
}

type CapsuleRow struct {
	ID          string
	Kind        string
	OwnerRepo   string
	Version     string
	Visibility  string
	Consumers   []string
	Exports     []string
	Description string
}

type ImportRow struct {
	CapsuleID            string
	ConsumerRepo         string
	RequiredVersionRange string
	Mode                 string
	Freshness            string
	MaterializedRevision string
}

type ImpactRow struct {
	CapsuleID string
	Repo      string
	Policy    string
	Blocked   bool
	DraftPR   string
	Reason    string
}

type ExportRow struct {
	Target         string
	Files          int
	GeneratedFiles int
	DependsOn      []string
	Description    string
}

type ViewModel struct {
	Page           Page
	Title          string
	Nav            []NavItem
	Capsules       []CapsuleRow
	Imports        []ImportRow
	ImpactRows     []ImpactRow
	ExportRows     []ExportRow
	ExportReady    bool
	ExportDigest   string
	GeneratedAt    string
	EmptyState     string
	UpstreamNotice string
}

type DemoData struct {
	Capsules []capsules.CapsuleManifest
	Imports  []capsules.CapsuleImport
	Impact   capsules.ImpactReport
	Export   exporter.Plan
}

func Render(page Page, data DemoData) (string, error) {
	model := BuildViewModel(page, data)
	tmpl, err := template.ParseFS(templateFS, "templates/layout.tmpl", "templates/*.tmpl")
	if err != nil {
		return "", err
	}
	var buffer bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buffer, "layout", model); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

func DemoDataSet() DemoData {
	registry := capsules.NewRegistry()
	api := capsules.CapsuleManifest{
		ID:          "kyba.backend.task-workflow.api.v1",
		Kind:        capsules.KindAPI,
		OwnerRepo:   "kyba-backend",
		Version:     "1.4.0",
		Visibility:  capsules.VisibilityImportedReposOnly,
		Exports:     []string{"proto/kyba/task_workflow/v1/task_workflow.proto", "context-capsule.md"},
		Consumers:   []string{"kyba-desktop", "kyba-ci"},
		Description: "Task workflow API and context capsule exported by backend.",
		Freshness:   capsules.FreshnessPolicy{MaxAgeDays: 14},
	}
	_ = registry.Publish(api)
	_ = registry.Import(capsules.CapsuleImport{CapsuleID: api.ID, ConsumerRepo: "kyba-desktop", RequiredVersionRange: "^1.4.0", Mode: capsules.ImportRequired})
	_, _ = registry.Materialize(api.ID, "kyba-desktop", "rev-demo", time.Now().UTC().Unix())
	impact, _ := registry.Update(api, true)

	manifest := exporter.Manifest{SchemaVersion: 1, ImportOrder: []string{"gitea-fork", "kyba-ci", "kyba-backend", "kyba-desktop", "kyba"}, FailAmbiguous: true, Targets: []exporter.Target{
		{Name: "gitea-fork", Description: "Gitea fork, repository hierarchy and interface/capsule UI.", Include: []string{"routers/**", "templates/repo/kcp/**", "models/kcp/**", "docs/kcp/**"}, DependsOn: []string{"kyba"}, GeneratedReadme: true},
		{Name: "kyba-ci", Description: "CI gates, export checks and draft PR fan-out.", Include: []string{"ci/**", ".github/**", "deploy/vps/**"}, DependsOn: []string{"kyba", "gitea-fork"}, GeneratedReadme: true},
		{Name: "kyba-backend", Description: "Backend services and exported API capsules.", Include: []string{"src/**", "kotlin/**", "proto/**", "db/**"}, DependsOn: []string{"kyba"}, GeneratedReadme: true},
		{Name: "kyba-desktop", Description: "Electron desktop client and imported capsules.", Include: []string{"electron/**", "src/**", "docs/**"}, DependsOn: []string{"kyba", "kyba-backend"}, GeneratedReadme: true},
		{Name: "kyba", Description: "Project knowledge cell.", Include: []string{"docs/**", "README.md"}, DependsOn: []string{"gitea-fork", "kyba-ci", "kyba-backend", "kyba-desktop"}},
	}}
	plan, _ := exporter.BuildPlan(manifest, []string{"routers/web/repo/kcp.go", "models/kcp/kcp.go", "docs/kcp/README.md", "src/offline/persistent.ts", "src/app.py", "docs/README.md", "README.md"})
	return DemoData{Capsules: registry.List(), Imports: registry.ImportsForConsumer("kyba-desktop"), Impact: impact, Export: plan}
}

// BuildViewModel converts KCP domain data into the view model used by both the
// standalone demo renderer and the integrated Gitea templates.
func BuildViewModel(page Page, data DemoData) ViewModel {
	model := ViewModel{
		Page:           page,
		Title:          titleFor(page),
		GeneratedAt:    time.Now().UTC().Format(time.RFC3339),
		UpstreamNotice: "KYBa KCP is integrated into this Gitea fork as a repository interface, context capsule and archive export control plane.",
	}
	for _, item := range []NavItem{{Page: PageDashboard, Title: "Overview"}, {Page: PageCapsules, Title: "Capsules"}, {Page: PageImports, Title: "Imports"}, {Page: PageImpact, Title: "Impact Queue"}, {Page: PageExport, Title: "Archive Export"}} {
		item.Active = item.Page == page
		model.Nav = append(model.Nav, item)
	}
	for _, manifest := range data.Capsules {
		model.Capsules = append(model.Capsules, CapsuleRow{ID: manifest.ID, Kind: string(manifest.Kind), OwnerRepo: manifest.OwnerRepo, Version: manifest.Version, Visibility: string(manifest.Visibility), Consumers: append([]string{}, manifest.Consumers...), Exports: append([]string{}, manifest.Exports...), Description: manifest.Description})
	}
	sort.Slice(model.Capsules, func(i, j int) bool { return model.Capsules[i].ID < model.Capsules[j].ID })
	for _, item := range data.Imports {
		model.Imports = append(model.Imports, ImportRow{CapsuleID: item.CapsuleID, ConsumerRepo: item.ConsumerRepo, RequiredVersionRange: item.RequiredVersionRange, Mode: string(item.Mode), Freshness: string(item.Freshness), MaterializedRevision: item.MaterializedRevision})
	}
	for _, task := range data.Impact.Tasks {
		row := ImpactRow{CapsuleID: task.CapsuleID, Repo: task.Repository, Policy: string(task.Policy), Blocked: task.Blocked, Reason: task.Reason}
		for _, pr := range data.Impact.DraftPRs {
			if pr.TaskID == task.ID {
				row.DraftPR = pr.Title
			}
		}
		model.ImpactRows = append(model.ImpactRows, row)
	}
	for _, summary := range data.Export.Summaries {
		model.ExportRows = append(model.ExportRows, ExportRow{Target: summary.Target, Files: summary.Files, GeneratedFiles: summary.GeneratedFiles, DependsOn: summary.DependsOn, Description: summary.Description})
	}
	model.ExportReady = data.Export.Ready()
	model.ExportDigest = data.Export.OwnershipDigest()
	if len(model.Capsules) == 0 && len(model.ExportRows) == 0 {
		model.EmptyState = "No KCP data available yet."
	}
	return model
}

func titleFor(page Page) string {
	switch page {
	case PageCapsules:
		return "Capsule Registry"
	case PageImports:
		return "Repository Imports"
	case PageImpact:
		return "Impact Queue"
	case PageExport:
		return "Archive Export"
	default:
		return "KYBa Repository Interfaces"
	}
}

// Compile-time reference to the service layer so the UI model stays aligned with export services.
var _ = kcpservice.ExportStatusReady
