package capsules

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	ErrInvalidCapsule   = errors.New("invalid capsule")
	ErrDuplicateCapsule = errors.New("duplicate capsule")
	ErrUnknownCapsule   = errors.New("unknown capsule")
	ErrBreakingChange   = errors.New("breaking capsule change")
	ErrDuplicateImport  = errors.New("duplicate capsule import")
	ErrUnknownImport    = errors.New("unknown capsule import")
	ErrAccessDenied     = errors.New("capsule access denied")
	ErrVersionMismatch  = errors.New("capsule version mismatch")
)

type Visibility string

const (
	VisibilityPrivate           Visibility = "private"
	VisibilityImportedReposOnly Visibility = "imported-repos-only"
	VisibilityPublic            Visibility = "public"
)

type CapsuleKind string

const (
	KindProduct CapsuleKind = "product"
	KindScreen  CapsuleKind = "screen"
	KindAPI     CapsuleKind = "api"
	KindEvent   CapsuleKind = "event"
	KindCI      CapsuleKind = "ci"
	KindContext CapsuleKind = "context"
	KindRepo    CapsuleKind = "repository-interface"
)

type ImpactPolicy string

const (
	ImpactCreateDraftPR ImpactPolicy = "create-generated-draft-pr"
	ImpactCreateTask    ImpactPolicy = "create-maintenance-task"
	ImpactBlock         ImpactPolicy = "block-until-owner-approval"
)

type ImportMode string

const (
	ImportRequired       ImportMode = "required"
	ImportValidationOnly ImportMode = "validation-only"
	ImportOptional       ImportMode = "optional"
)

type FreshnessStatus string

const (
	FreshnessUnknown FreshnessStatus = "unknown"
	FreshnessFresh   FreshnessStatus = "fresh"
	FreshnessStale   FreshnessStatus = "stale"
)

type OwnerSet struct {
	Product   string
	Technical string
	QA        string
	Security  string
}

type FreshnessPolicy struct {
	MaxAgeDays int
}

type CapsuleManifest struct {
	ID          string
	Kind        CapsuleKind
	OwnerRepo   string
	Version     string
	Visibility  Visibility
	Exports     []string
	Consumers   []string
	Impact      ImpactPolicy
	Description string
	Owners      OwnerSet
	Freshness   FreshnessPolicy
}

type CapsuleImport struct {
	CapsuleID            string
	ConsumerRepo         string
	RequiredVersionRange string
	Mode                 ImportMode
	MaterializedRevision string
	MaterializedAtUnix   int64
	Freshness            FreshnessStatus
}

type MaterializedSnapshot struct {
	CapsuleID    string
	ConsumerRepo string
	Version      string
	Revision     string
	Files        []string
	ContextPath  string
	Digest       string
}

type MaintenanceTask struct {
	ID           string
	CapsuleID    string
	Repository   string
	Policy       ImpactPolicy
	Reason       string
	DraftPRTitle string
	Blocked      bool
}

type DraftPR struct {
	Repository string
	Title      string
	Branch     string
	TaskID     string
	Generated  bool
}

type ImpactReport struct {
	CapsuleID       string
	Compatible      bool
	Breaking        bool
	AffectedRepos   []string
	RequiredActions map[string]ImpactPolicy
	Tasks           []MaintenanceTask
	DraftPRs        []DraftPR
}

type Registry struct {
	capsules map[string]CapsuleManifest
	imports  map[string]CapsuleImport
}

func NewRegistry() *Registry {
	return &Registry{capsules: map[string]CapsuleManifest{}, imports: map[string]CapsuleImport{}}
}

func (r *Registry) Publish(manifest CapsuleManifest) error {
	if err := ValidateManifest(manifest); err != nil {
		return err
	}
	if _, exists := r.capsules[manifest.ID]; exists {
		return ErrDuplicateCapsule
	}
	r.capsules[manifest.ID] = normalize(manifest)
	return nil
}

func (r *Registry) Upsert(manifest CapsuleManifest, compatible bool) (ImpactReport, error) {
	if _, exists := r.capsules[manifest.ID]; !exists {
		return ImpactReport{}, r.Publish(manifest)
	}
	return r.Update(manifest, compatible)
}

func (r *Registry) Import(request CapsuleImport) error {
	if err := ValidateImport(request); err != nil {
		return err
	}
	manifest, exists := r.capsules[request.CapsuleID]
	if !exists {
		return ErrUnknownCapsule
	}
	if !AllowsConsumer(manifest, request.ConsumerRepo) {
		return ErrAccessDenied
	}
	if !VersionAllows(request.RequiredVersionRange, manifest.Version) {
		return ErrVersionMismatch
	}
	key := importKey(request.CapsuleID, request.ConsumerRepo)
	if _, exists := r.imports[key]; exists {
		return ErrDuplicateImport
	}
	if request.Freshness == "" {
		request.Freshness = FreshnessUnknown
	}
	r.imports[key] = request
	return nil
}

func (r *Registry) Materialize(capsuleID, consumerRepo, revision string, nowUnix int64) (MaterializedSnapshot, error) {
	manifest, exists := r.capsules[capsuleID]
	if !exists {
		return MaterializedSnapshot{}, ErrUnknownCapsule
	}
	key := importKey(capsuleID, consumerRepo)
	request, exists := r.imports[key]
	if !exists {
		return MaterializedSnapshot{}, ErrUnknownImport
	}
	request.MaterializedRevision = revision
	request.MaterializedAtUnix = nowUnix
	request.Freshness = FreshnessFresh
	r.imports[key] = request
	files := append([]string{}, manifest.Exports...)
	sort.Strings(files)
	return MaterializedSnapshot{
		CapsuleID:    capsuleID,
		ConsumerRepo: consumerRepo,
		Version:      manifest.Version,
		Revision:     revision,
		Files:        files,
		ContextPath:  ".kyba/imported-capsules/" + capsuleID + "/context-capsule.md",
		Digest:       snapshotDigest(capsuleID, consumerRepo, manifest.Version, revision, files),
	}, nil
}

func (r *Registry) Update(manifest CapsuleManifest, compatible bool) (ImpactReport, error) {
	if err := ValidateManifest(manifest); err != nil {
		return ImpactReport{}, err
	}
	if _, exists := r.capsules[manifest.ID]; !exists {
		return ImpactReport{}, ErrUnknownCapsule
	}
	next := normalize(manifest)
	r.capsules[manifest.ID] = next
	repos := r.affectedRepos(next.ID, next.Consumers)
	report := ImpactReport{
		CapsuleID:       manifest.ID,
		Compatible:      compatible,
		Breaking:        !compatible,
		AffectedRepos:   repos,
		RequiredActions: map[string]ImpactPolicy{},
	}
	for _, repo := range repos {
		policy := ImpactCreateDraftPR
		if !compatible {
			policy = ImpactBlock
		}
		report.RequiredActions[repo] = policy
		task := MaintenanceTask{
			ID:           taskID(manifest.ID, repo),
			CapsuleID:    manifest.ID,
			Repository:   repo,
			Policy:       policy,
			Reason:       reason(compatible),
			DraftPRTitle: fmt.Sprintf("Update imported capsule %s", manifest.ID),
			Blocked:      !compatible,
		}
		report.Tasks = append(report.Tasks, task)
		if compatible {
			report.DraftPRs = append(report.DraftPRs, DraftPR{Repository: repo, Title: task.DraftPRTitle, Branch: "capsule/" + safeID(manifest.ID), TaskID: task.ID, Generated: true})
		}
	}
	if !compatible {
		return report, ErrBreakingChange
	}
	return report, nil
}

func (r *Registry) Get(id string) (CapsuleManifest, bool) {
	manifest, exists := r.capsules[id]
	return manifest, exists
}

func (r *Registry) List() []CapsuleManifest {
	result := make([]CapsuleManifest, 0, len(r.capsules))
	for _, manifest := range r.capsules {
		result = append(result, manifest)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func (r *Registry) ImportsForConsumer(consumerRepo string) []CapsuleImport {
	result := []CapsuleImport{}
	for _, item := range r.imports {
		if item.ConsumerRepo == consumerRepo {
			result = append(result, item)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CapsuleID < result[j].CapsuleID })
	return result
}

func (r *Registry) ImportsForCapsule(capsuleID string) []CapsuleImport {
	result := []CapsuleImport{}
	for _, item := range r.imports {
		if item.CapsuleID == capsuleID {
			result = append(result, item)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ConsumerRepo < result[j].ConsumerRepo })
	return result
}

func (r *Registry) MarkStale(nowUnix int64) []CapsuleImport {
	stale := []CapsuleImport{}
	for key, item := range r.imports {
		manifest := r.capsules[item.CapsuleID]
		maxAge := int64(manifest.Freshness.MaxAgeDays) * 24 * 60 * 60
		if maxAge <= 0 || item.MaterializedAtUnix == 0 {
			continue
		}
		if nowUnix-item.MaterializedAtUnix > maxAge {
			item.Freshness = FreshnessStale
			r.imports[key] = item
			stale = append(stale, item)
		}
	}
	sort.Slice(stale, func(i, j int) bool {
		return importKey(stale[i].CapsuleID, stale[i].ConsumerRepo) < importKey(stale[j].CapsuleID, stale[j].ConsumerRepo)
	})
	return stale
}

func (r *Registry) CanRead(repo, capsuleID string) bool {
	manifest, exists := r.capsules[capsuleID]
	if !exists {
		return false
	}
	if repo == manifest.OwnerRepo || manifest.Visibility == VisibilityPublic {
		return true
	}
	if manifest.Visibility == VisibilityPrivate {
		return false
	}
	if AllowsConsumer(manifest, repo) {
		return true
	}
	_, imported := r.imports[importKey(capsuleID, repo)]
	return imported
}

func (r *Registry) affectedRepos(capsuleID string, manifestConsumers []string) []string {
	seen := map[string]struct{}{}
	for _, repo := range manifestConsumers {
		repo = strings.TrimSpace(repo)
		if repo != "" {
			seen[repo] = struct{}{}
		}
	}
	for _, item := range r.imports {
		if item.CapsuleID == capsuleID {
			seen[item.ConsumerRepo] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for repo := range seen {
		result = append(result, repo)
	}
	sort.Strings(result)
	return result
}

func ValidateManifest(manifest CapsuleManifest) error {
	if !validID(manifest.ID) {
		return fmt.Errorf("%w: id", ErrInvalidCapsule)
	}
	if !validKind(manifest.Kind) {
		return fmt.Errorf("%w: kind", ErrInvalidCapsule)
	}
	if !validRepo(manifest.OwnerRepo) {
		return fmt.Errorf("%w: owner repo", ErrInvalidCapsule)
	}
	if isForbiddenStandaloneRepo(manifest.OwnerRepo) {
		return fmt.Errorf("%w: standalone capsule repository is not allowed", ErrInvalidCapsule)
	}
	if manifest.Version == "" || !validSemver(manifest.Version) {
		return fmt.Errorf("%w: version", ErrInvalidCapsule)
	}
	if !validVisibility(manifest.Visibility) {
		return fmt.Errorf("%w: visibility", ErrInvalidCapsule)
	}
	if len(manifest.Exports) == 0 {
		return fmt.Errorf("%w: exports", ErrInvalidCapsule)
	}
	for _, export := range manifest.Exports {
		if strings.TrimSpace(export) == "" || strings.Contains(export, "..") || strings.HasPrefix(export, "/") {
			return fmt.Errorf("%w: export path", ErrInvalidCapsule)
		}
	}
	for _, consumer := range manifest.Consumers {
		if !validRepo(consumer) || consumer == manifest.OwnerRepo {
			return fmt.Errorf("%w: consumer", ErrInvalidCapsule)
		}
	}
	if manifest.Freshness.MaxAgeDays < 0 {
		return fmt.Errorf("%w: freshness", ErrInvalidCapsule)
	}
	return nil
}

func ValidateImport(request CapsuleImport) error {
	if !validID(request.CapsuleID) {
		return fmt.Errorf("%w: import capsule id", ErrInvalidCapsule)
	}
	if !validRepo(request.ConsumerRepo) {
		return fmt.Errorf("%w: import consumer", ErrInvalidCapsule)
	}
	if request.RequiredVersionRange == "" {
		return fmt.Errorf("%w: import version range", ErrInvalidCapsule)
	}
	if !validImportMode(request.Mode) {
		return fmt.Errorf("%w: import mode", ErrInvalidCapsule)
	}
	return nil
}

func AllowsConsumer(manifest CapsuleManifest, repo string) bool {
	for _, consumer := range manifest.Consumers {
		if consumer == repo {
			return true
		}
	}
	return false
}

func VersionAllows(requiredRange, actual string) bool {
	requiredRange = strings.TrimSpace(requiredRange)
	actual = strings.TrimSpace(actual)
	if requiredRange == "*" || requiredRange == actual {
		return true
	}
	if strings.HasPrefix(requiredRange, "^") {
		base := strings.TrimPrefix(requiredRange, "^")
		baseParts, ok1 := semverParts(base)
		actualParts, ok2 := semverParts(actual)
		return ok1 && ok2 && actualParts[0] == baseParts[0] && compareSemver(actualParts, baseParts) >= 0
	}
	return false
}

func normalize(manifest CapsuleManifest) CapsuleManifest {
	manifest.Exports = uniqueSorted(manifest.Exports)
	manifest.Consumers = uniqueSorted(manifest.Consumers)
	if manifest.Impact == "" {
		manifest.Impact = ImpactCreateTask
	}
	return manifest
}

func uniqueSorted(values []string) []string {
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			seen[trimmed] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func validID(id string) bool {
	return regexp.MustCompile(`^kyba\.[a-z0-9][a-z0-9.-]*\.v[0-9]+$`).MatchString(id)
}

func validRepo(repo string) bool {
	return regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`).MatchString(repo) && !strings.Contains(repo, "..")
}

func isForbiddenStandaloneRepo(repo string) bool {
	switch repo {
	case "kyba-context-capsules", "kyba-contracts", "kyba-infra":
		return true
	default:
		return false
	}
}

func validKind(kind CapsuleKind) bool {
	switch kind {
	case KindProduct, KindScreen, KindAPI, KindEvent, KindCI, KindContext, KindRepo:
		return true
	default:
		return false
	}
}

func validVisibility(visibility Visibility) bool {
	switch visibility {
	case VisibilityPrivate, VisibilityImportedReposOnly, VisibilityPublic:
		return true
	default:
		return false
	}
}

func validImportMode(mode ImportMode) bool {
	switch mode {
	case ImportRequired, ImportValidationOnly, ImportOptional:
		return true
	default:
		return false
	}
}

func validSemver(version string) bool {
	_, ok := semverParts(version)
	return ok
}

func semverParts(version string) ([3]int, bool) {
	var result [3]int
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return result, false
	}
	for i, part := range parts {
		value, err := strconv.Atoi(part)
		if err != nil || value < 0 {
			return result, false
		}
		result[i] = value
	}
	return result, true
}

func compareSemver(left, right [3]int) int {
	for i := 0; i < 3; i++ {
		if left[i] > right[i] {
			return 1
		}
		if left[i] < right[i] {
			return -1
		}
	}
	return 0
}

func importKey(capsuleID, consumerRepo string) string {
	return capsuleID + "\x00" + consumerRepo
}

func taskID(capsuleID, repo string) string {
	return "capsule-maintenance:" + safeID(capsuleID) + ":" + repo
}

func safeID(value string) string {
	value = strings.ReplaceAll(value, ".", "-")
	value = strings.ReplaceAll(value, "/", "-")
	return value
}

func snapshotDigest(capsuleID, consumerRepo, version, revision string, files []string) string {
	payload := capsuleID + "\n" + consumerRepo + "\n" + version + "\n" + revision + "\n" + strings.Join(uniqueSorted(files), "\n")
	// A short deterministic digest is enough for UI and test fixtures. The CI layer may store full hashes.
	var sum uint32
	for _, r := range payload {
		sum = sum*33 + uint32(r)
	}
	return fmt.Sprintf("%08x", sum)
}

func reason(compatible bool) string {
	if compatible {
		return "compatible capsule change; generated update can be proposed"
	}
	return "breaking capsule change; owner approval and manual migration plan required"
}
