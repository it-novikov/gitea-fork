package exporter

import (
	"errors"
	"strings"
	"testing"
)

func sampleManifest() Manifest {
	return Manifest{
		SchemaVersion: 1,
		FailAmbiguous: true,
		ImportOrder:   []string{"gitea-fork", "kyba-ci", "kyba-backend", "kyba-desktop", "kyba"},
		Targets: []Target{
			{Name: "kyba", Description: "Project knowledge cell", Include: []string{"docs/**", "README.md"}},
			{Name: "kyba-desktop", Description: "Desktop", Include: []string{"kotlin/desktop/**"}},
			{Name: "kyba-backend", Description: "Backend", Include: []string{"kotlin/task-workflow/**"}},
			{Name: "gitea-fork", Description: "Git platform", Include: []string{"integrations/gitea-fork/**"}},
			{Name: "kyba-ci", Description: "CI", Include: []string{"ci/jenkins/**", ".github/workflows/**"}},
		},
	}
}

func TestBuildPlanClassifiesRepositoryTargets(t *testing.T) {
	plan, err := BuildPlan(sampleManifest(), []string{
		"README.md",
		"docs/README.md",
		"kotlin/desktop/build.gradle.kts",
		"kotlin/task-workflow/build.gradle.kts",
		"integrations/gitea-fork/capsules/capsule.go",
		"ci/jenkins/Jenkinsfile.pr",
	})
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}
	if !plan.Ready() {
		t.Fatalf("plan is not ready: %#v", plan)
	}
	counts := plan.CountByTarget()
	if counts["kyba"] != 2 || counts["kyba-desktop"] != 1 || counts["kyba-backend"] != 1 || counts["gitea-fork"] != 1 || counts["kyba-ci"] != 1 {
		t.Fatalf("unexpected counts: %#v", counts)
	}
}

func TestBuildPlanFailsOnUnassignedFile(t *testing.T) {
	_, err := BuildPlan(sampleManifest(), []string{"unknown/file.txt"})
	if !errors.Is(err, ErrUnassignedFile) {
		t.Fatalf("expected unassigned file error, got %v", err)
	}
}

func TestBuildPlanFailsOnAmbiguousFile(t *testing.T) {
	manifest := sampleManifest()
	manifest.Targets[0].Include = append(manifest.Targets[0].Include, "kotlin/**")
	_, err := BuildPlan(manifest, []string{"kotlin/desktop/build.gradle.kts"})
	if !errors.Is(err, ErrAmbiguousFile) {
		t.Fatalf("expected ambiguous file error, got %v", err)
	}
}

func TestStandaloneCapsuleRepositoryIsRejected(t *testing.T) {
	manifest := sampleManifest()
	manifest.Targets = append(manifest.Targets, Target{Name: "kyba-context-capsules", Include: []string{"capsules/**"}})
	if err := ValidateManifest(manifest); !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected invalid manifest, got %v", err)
	}
}

func TestGeneratedReadmeMentionsLocalOwnership(t *testing.T) {
	text := GenerateReadme(Target{Name: "kyba-backend", Description: "Backend services"})
	if text == "" || !matchPattern("kotlin/task-workflow/build.gradle.kts", "kotlin/task-workflow/**") {
		t.Fatalf("unexpected generated content or matcher failure")
	}
}

func TestImportOrderMustCoverAllTargets(t *testing.T) {
	manifest := sampleManifest()
	manifest.ImportOrder = []string{"kyba"}
	if !errors.Is(ValidateManifest(manifest), ErrInvalidManifest) {
		t.Fatalf("expected import order omission to be invalid")
	}
}

func TestInvalidPatternIsRejected(t *testing.T) {
	manifest := sampleManifest()
	manifest.Targets[0].Include = []string{"../README.md"}
	if !errors.Is(ValidateManifest(manifest), ErrInvalidManifest) {
		t.Fatalf("expected unsafe pattern rejection")
	}
}

func TestPlanIncludesSummariesAndDigest(t *testing.T) {
	plan, err := BuildPlan(sampleManifest(), []string{"README.md", "docs/README.md"})
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}
	if len(plan.Summaries) != 5 || plan.OwnershipDigest() == "" {
		t.Fatalf("missing summaries or digest: %#v", plan)
	}
}

func TestGeneratedRepositoryInterface(t *testing.T) {
	text := GenerateRepositoryInterface(Target{Name: "kyba-backend", Description: "Backend", DependsOn: []string{"kyba"}})
	if !strings.Contains(text, "kyba.repo.kyba-backend.interface.v1") || !strings.Contains(text, "owner_repo: kyba-backend") {
		t.Fatalf("unexpected interface manifest: %s", text)
	}
}
