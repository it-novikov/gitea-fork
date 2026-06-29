package kcp

import (
	"errors"
	"testing"
	"time"

	"code.gitea.io/gitea/modules/kcp/capsules"
	"code.gitea.io/gitea/modules/kcp/exporter"
)

func TestExportServiceCreatesReadyRun(t *testing.T) {
	svc := NewExportService()
	manifest := exporter.Manifest{SchemaVersion: 1, FailAmbiguous: true, ImportOrder: []string{"kyba", "kyba-backend"}, Targets: []exporter.Target{
		{Name: "kyba", Description: "Project", Include: []string{"docs/**"}, GeneratedReadme: false},
		{Name: "kyba-backend", Description: "Backend", Include: []string{"kotlin/task-workflow/**"}, DependsOn: []string{"kyba"}, GeneratedReadme: true},
	}}
	run, err := svc.Plan(manifest, []string{"docs/README.md", "kotlin/task-workflow/build.gradle.kts"}, time.Unix(1000, 0))
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if run.Status != ExportStatusReady || run.OwnershipDigest == "" || len(run.Artifacts) != 2 {
		t.Fatalf("unexpected run: %#v", run)
	}
	stored, ok := svc.Get(run.ID)
	if !ok || stored.ID != run.ID {
		t.Fatalf("stored run missing")
	}
}

func TestExportServiceBlocksAmbiguousRun(t *testing.T) {
	svc := NewExportService()
	manifest := exporter.Manifest{SchemaVersion: 1, FailAmbiguous: true, ImportOrder: []string{"kyba", "kyba-backend"}, Targets: []exporter.Target{
		{Name: "kyba", Description: "Project", Include: []string{"kotlin/**"}},
		{Name: "kyba-backend", Description: "Backend", Include: []string{"kotlin/task-workflow/**"}, DependsOn: []string{"kyba"}},
	}}
	run, err := svc.Plan(manifest, []string{"kotlin/task-workflow/build.gradle.kts"}, time.Unix(1000, 0))
	if !errors.Is(err, exporter.ErrAmbiguousFile) {
		t.Fatalf("expected ambiguous error, got %v", err)
	}
	if run.Status != ExportStatusBlocked {
		t.Fatalf("expected blocked run: %#v", run)
	}
}

func TestCapsuleServiceMaterializesAndImpacts(t *testing.T) {
	svc := NewCapsuleService(nil)
	manifest := capsules.CapsuleManifest{ID: "kyba.backend.task-workflow.api.v1", Kind: capsules.KindAPI, OwnerRepo: "kyba-backend", Version: "1.0.0", Visibility: capsules.VisibilityImportedReposOnly, Exports: []string{"proto/a.proto"}, Consumers: []string{"kyba-desktop"}}
	if err := svc.Publish(manifest); err != nil {
		t.Fatal(err)
	}
	if err := svc.Import(capsules.CapsuleImport{CapsuleID: manifest.ID, ConsumerRepo: "kyba-desktop", RequiredVersionRange: "^1.0.0", Mode: capsules.ImportRequired}); err != nil {
		t.Fatal(err)
	}
	snapshot, err := svc.Materialize(manifest.ID, "kyba-desktop", "rev-1", time.Unix(1000, 0))
	if err != nil || snapshot.Digest == "" {
		t.Fatalf("materialize: %#v %v", snapshot, err)
	}
	manifest.Version = "1.1.0"
	report, err := svc.Change(manifest, true)
	if err != nil || len(report.DraftPRs) != 1 || len(report.Tasks) != 1 {
		t.Fatalf("impact: %#v %v", report, err)
	}
}
