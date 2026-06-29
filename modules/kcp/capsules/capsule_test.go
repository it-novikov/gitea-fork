package capsules

import (
	"errors"
	"testing"
)

func TestPublishAndGetCapsule(t *testing.T) {
	registry := NewRegistry()
	manifest := CapsuleManifest{
		ID:         "kyba.backend.task-workflow.api.v1",
		Kind:       KindAPI,
		OwnerRepo:  "kyba-backend",
		Version:    "1.0.0",
		Visibility: VisibilityImportedReposOnly,
		Exports:    []string{"proto/kyba/task_workflow/v1/service.proto"},
		Consumers:  []string{"kyba-desktop", "kyba-ci"},
		Impact:     ImpactCreateDraftPR,
	}
	if err := registry.Publish(manifest); err != nil {
		t.Fatalf("publish: %v", err)
	}
	stored, ok := registry.Get(manifest.ID)
	if !ok {
		t.Fatalf("capsule missing")
	}
	if stored.Consumers[0] != "kyba-ci" || stored.Consumers[1] != "kyba-desktop" {
		t.Fatalf("consumers not normalized: %#v", stored.Consumers)
	}
}

func TestCompatibleUpdateCreatesDraftPRActions(t *testing.T) {
	registry := NewRegistry()
	manifest := CapsuleManifest{ID: "kyba.backend.task-workflow.api.v1", Kind: KindAPI, OwnerRepo: "kyba-backend", Version: "1.0.0", Visibility: VisibilityImportedReposOnly, Exports: []string{"proto/a.proto"}, Consumers: []string{"kyba-desktop"}, Impact: ImpactCreateDraftPR}
	if err := registry.Publish(manifest); err != nil {
		t.Fatal(err)
	}
	manifest.Version = "1.1.0"
	report, err := registry.Update(manifest, true)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if !report.Compatible || report.RequiredActions["kyba-desktop"] != ImpactCreateDraftPR {
		t.Fatalf("unexpected report: %#v", report)
	}
}

func TestBreakingUpdateBlocksConsumers(t *testing.T) {
	registry := NewRegistry()
	manifest := CapsuleManifest{ID: "kyba.backend.task-workflow.api.v1", Kind: KindAPI, OwnerRepo: "kyba-backend", Version: "1.0.0", Visibility: VisibilityImportedReposOnly, Exports: []string{"proto/a.proto"}, Consumers: []string{"kyba-desktop"}, Impact: ImpactCreateDraftPR}
	if err := registry.Publish(manifest); err != nil {
		t.Fatal(err)
	}
	manifest.Version = "2.0.0"
	report, err := registry.Update(manifest, false)
	if !errors.Is(err, ErrBreakingChange) {
		t.Fatalf("expected breaking change, got %v", err)
	}
	if report.RequiredActions["kyba-desktop"] != ImpactBlock {
		t.Fatalf("breaking change did not block consumer: %#v", report)
	}
}

func TestInvalidManifestRequiresExports(t *testing.T) {
	manifest := CapsuleManifest{ID: "kyba.backend.task-workflow.api.v1", Kind: KindAPI, OwnerRepo: "kyba-backend", Version: "1.0.0", Visibility: VisibilityImportedReposOnly}
	if !errors.Is(ValidateManifest(manifest), ErrInvalidCapsule) {
		t.Fatalf("expected invalid capsule")
	}
}

func TestImportAndMaterializeCapsuleSnapshot(t *testing.T) {
	registry := NewRegistry()
	manifest := CapsuleManifest{ID: "kyba.backend.task-workflow.api.v1", Kind: KindAPI, OwnerRepo: "kyba-backend", Version: "1.0.0", Visibility: VisibilityImportedReposOnly, Exports: []string{"proto/a.proto", "context.md"}, Consumers: []string{"kyba-desktop"}, Impact: ImpactCreateDraftPR}
	if err := registry.Publish(manifest); err != nil {
		t.Fatal(err)
	}
	if err := registry.Import(CapsuleImport{CapsuleID: manifest.ID, ConsumerRepo: "kyba-desktop", RequiredVersionRange: "^1.0.0", Mode: ImportRequired}); err != nil {
		t.Fatalf("import: %v", err)
	}
	snapshot, err := registry.Materialize(manifest.ID, "kyba-desktop", "rev-1", 1000)
	if err != nil {
		t.Fatalf("materialize: %v", err)
	}
	if snapshot.ContextPath == "" || len(snapshot.Files) != 2 {
		t.Fatalf("unexpected snapshot: %#v", snapshot)
	}
	imports := registry.ImportsForConsumer("kyba-desktop")
	if len(imports) != 1 || imports[0].Freshness != FreshnessFresh {
		t.Fatalf("unexpected imports: %#v", imports)
	}
}

func TestCompatibleUpdateCreatesTaskAndDraftPR(t *testing.T) {
	registry := NewRegistry()
	manifest := CapsuleManifest{ID: "kyba.backend.task-workflow.api.v1", Kind: KindAPI, OwnerRepo: "kyba-backend", Version: "1.0.0", Visibility: VisibilityImportedReposOnly, Exports: []string{"proto/a.proto"}, Consumers: []string{"kyba-desktop", "kyba-ci"}, Impact: ImpactCreateDraftPR}
	if err := registry.Publish(manifest); err != nil {
		t.Fatal(err)
	}
	if err := registry.Import(CapsuleImport{CapsuleID: manifest.ID, ConsumerRepo: "kyba-ci", RequiredVersionRange: "^1.0.0", Mode: ImportValidationOnly}); err != nil {
		t.Fatal(err)
	}
	manifest.Version = "1.1.0"
	report, err := registry.Update(manifest, true)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(report.Tasks) != 2 || len(report.DraftPRs) != 2 {
		t.Fatalf("expected tasks and draft PRs for both consumers: %#v", report)
	}
}

func TestStandaloneCapsuleOwnerIsRejected(t *testing.T) {
	manifest := CapsuleManifest{ID: "kyba.context.bad.v1", Kind: KindContext, OwnerRepo: "kyba-context-capsules", Version: "1.0.0", Visibility: VisibilityImportedReposOnly, Exports: []string{"context.md"}}
	if !errors.Is(ValidateManifest(manifest), ErrInvalidCapsule) {
		t.Fatalf("expected standalone capsule repo rejection")
	}
}

func TestImportRejectsUnauthorizedConsumer(t *testing.T) {
	registry := NewRegistry()
	manifest := CapsuleManifest{ID: "kyba.backend.task-workflow.api.v1", Kind: KindAPI, OwnerRepo: "kyba-backend", Version: "1.0.0", Visibility: VisibilityImportedReposOnly, Exports: []string{"proto/a.proto"}, Consumers: []string{"kyba-desktop"}, Impact: ImpactCreateDraftPR}
	if err := registry.Publish(manifest); err != nil {
		t.Fatal(err)
	}
	err := registry.Import(CapsuleImport{CapsuleID: manifest.ID, ConsumerRepo: "kyba-ci", RequiredVersionRange: "^1.0.0", Mode: ImportValidationOnly})
	if !errors.Is(err, ErrAccessDenied) {
		t.Fatalf("expected access denied, got %v", err)
	}
}

func TestImportRejectsVersionMismatch(t *testing.T) {
	registry := NewRegistry()
	manifest := CapsuleManifest{ID: "kyba.backend.task-workflow.api.v1", Kind: KindAPI, OwnerRepo: "kyba-backend", Version: "2.0.0", Visibility: VisibilityImportedReposOnly, Exports: []string{"proto/a.proto"}, Consumers: []string{"kyba-desktop"}, Impact: ImpactCreateDraftPR}
	if err := registry.Publish(manifest); err != nil {
		t.Fatal(err)
	}
	err := registry.Import(CapsuleImport{CapsuleID: manifest.ID, ConsumerRepo: "kyba-desktop", RequiredVersionRange: "^1.0.0", Mode: ImportRequired})
	if !errors.Is(err, ErrVersionMismatch) {
		t.Fatalf("expected version mismatch, got %v", err)
	}
}

func TestMarkStaleImports(t *testing.T) {
	registry := NewRegistry()
	manifest := CapsuleManifest{ID: "kyba.backend.task-workflow.api.v1", Kind: KindAPI, OwnerRepo: "kyba-backend", Version: "1.0.0", Visibility: VisibilityImportedReposOnly, Exports: []string{"proto/a.proto"}, Consumers: []string{"kyba-desktop"}, Freshness: FreshnessPolicy{MaxAgeDays: 1}}
	if err := registry.Publish(manifest); err != nil {
		t.Fatal(err)
	}
	if err := registry.Import(CapsuleImport{CapsuleID: manifest.ID, ConsumerRepo: "kyba-desktop", RequiredVersionRange: "^1.0.0", Mode: ImportRequired}); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Materialize(manifest.ID, "kyba-desktop", "rev-1", 1000); err != nil {
		t.Fatal(err)
	}
	stale := registry.MarkStale(1000 + 3*24*60*60)
	if len(stale) != 1 || stale[0].Freshness != FreshnessStale {
		t.Fatalf("expected stale import, got %#v", stale)
	}
}

func TestCanReadCapsuleIsBounded(t *testing.T) {
	registry := NewRegistry()
	manifest := CapsuleManifest{ID: "kyba.backend.task-workflow.api.v1", Kind: KindAPI, OwnerRepo: "kyba-backend", Version: "1.0.0", Visibility: VisibilityImportedReposOnly, Exports: []string{"proto/a.proto"}, Consumers: []string{"kyba-desktop"}}
	if err := registry.Publish(manifest); err != nil {
		t.Fatal(err)
	}
	if !registry.CanRead("kyba-backend", manifest.ID) || !registry.CanRead("kyba-desktop", manifest.ID) {
		t.Fatalf("owner and declared consumer should read")
	}
	if registry.CanRead("kyba-ci", manifest.ID) {
		t.Fatalf("undeclared repo should not read")
	}
}
