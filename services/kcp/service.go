package kcp

import (
	"fmt"
	"sort"
	"time"

	"code.gitea.io/gitea/modules/kcp/capsules"
	"code.gitea.io/gitea/modules/kcp/exporter"
)

type ExportStatus string

const (
	ExportStatusPlanned ExportStatus = "planned"
	ExportStatusReady   ExportStatus = "ready"
	ExportStatusBlocked ExportStatus = "blocked"
)

type ExportArtifact struct {
	Target         string
	ArchiveName    string
	FileCount      int
	GeneratedFiles int
	Digest         string
}

type ExportRun struct {
	ID              string
	Status          ExportStatus
	Plan            exporter.Plan
	Artifacts       []ExportArtifact
	CreatedAt       time.Time
	OwnershipDigest string
}

type ExportService struct {
	runs map[string]ExportRun
}

func NewExportService() *ExportService {
	return &ExportService{runs: map[string]ExportRun{}}
}

func (s *ExportService) Plan(manifest exporter.Manifest, files []string, now time.Time) (ExportRun, error) {
	plan, err := exporter.BuildPlan(manifest, files)
	status := ExportStatusReady
	if err != nil || !plan.Ready() {
		status = ExportStatusBlocked
	}
	run := ExportRun{
		ID:              runID(now),
		Status:          status,
		Plan:            plan,
		CreatedAt:       now.UTC(),
		OwnershipDigest: plan.OwnershipDigest(),
	}
	for _, summary := range plan.Summaries {
		run.Artifacts = append(run.Artifacts, ExportArtifact{Target: summary.Target, ArchiveName: summary.Target + ".zip", FileCount: summary.Files, GeneratedFiles: summary.GeneratedFiles, Digest: run.OwnershipDigest[:12] + ":" + summary.Target})
	}
	sort.Slice(run.Artifacts, func(i, j int) bool { return run.Artifacts[i].Target < run.Artifacts[j].Target })
	s.runs[run.ID] = run
	return run, err
}

func (s *ExportService) Get(id string) (ExportRun, bool) {
	run, ok := s.runs[id]
	return run, ok
}

type CapsuleService struct {
	Registry *capsules.Registry
}

func NewCapsuleService(registry *capsules.Registry) *CapsuleService {
	if registry == nil {
		registry = capsules.NewRegistry()
	}
	return &CapsuleService{Registry: registry}
}

func (s *CapsuleService) Publish(manifest capsules.CapsuleManifest) error {
	return s.Registry.Publish(manifest)
}

func (s *CapsuleService) Import(request capsules.CapsuleImport) error {
	return s.Registry.Import(request)
}

func (s *CapsuleService) Materialize(capsuleID, consumerRepo, revision string, now time.Time) (capsules.MaterializedSnapshot, error) {
	return s.Registry.Materialize(capsuleID, consumerRepo, revision, now.UTC().Unix())
}

func (s *CapsuleService) Change(manifest capsules.CapsuleManifest, compatible bool) (capsules.ImpactReport, error) {
	return s.Registry.Update(manifest, compatible)
}

func runID(now time.Time) string {
	return fmt.Sprintf("kcp-export-%d", now.UTC().Unix())
}
