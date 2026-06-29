package exporter

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var (
	ErrInvalidManifest = errors.New("invalid export manifest")
	ErrAmbiguousFile   = errors.New("ambiguous export file")
	ErrUnassignedFile  = errors.New("unassigned export file")
)

var forbiddenTargets = map[string]struct{}{
	"kyba-context-capsules": {},
	"kyba-contracts":        {},
	"kyba-infra":            {},
}

type Target struct {
	Name            string
	Description     string
	Include         []string
	Exclude         []string
	DependsOn       []string
	GeneratedReadme bool
}

type Manifest struct {
	SchemaVersion int
	Targets       []Target
	ImportOrder   []string
	FailAmbiguous bool
}

type Assignment struct {
	Path   string
	Target string
}

type TargetSummary struct {
	Target         string
	Description    string
	Files          int
	DependsOn      []string
	GeneratedFiles int
}

type Plan struct {
	Assignments []Assignment
	Ambiguous   map[string][]string
	Unassigned  []string
	ImportOrder []string
	Summaries   []TargetSummary
}

func ValidateManifest(manifest Manifest) error {
	if manifest.SchemaVersion != 1 {
		return fmt.Errorf("%w: schema version", ErrInvalidManifest)
	}
	if len(manifest.Targets) == 0 {
		return fmt.Errorf("%w: no targets", ErrInvalidManifest)
	}
	seen := map[string]struct{}{}
	for _, target := range manifest.Targets {
		if !validTargetName(target.Name) {
			return fmt.Errorf("%w: target name %q", ErrInvalidManifest, target.Name)
		}
		if _, forbidden := forbiddenTargets[target.Name]; forbidden {
			return fmt.Errorf("%w: standalone target %q is not allowed", ErrInvalidManifest, target.Name)
		}
		if _, exists := seen[target.Name]; exists {
			return fmt.Errorf("%w: duplicate target %q", ErrInvalidManifest, target.Name)
		}
		seen[target.Name] = struct{}{}
		if len(target.Include) == 0 {
			return fmt.Errorf("%w: target %q has no include rules", ErrInvalidManifest, target.Name)
		}
		for _, pattern := range append(append([]string{}, target.Include...), target.Exclude...) {
			if !validPattern(pattern) {
				return fmt.Errorf("%w: invalid pattern %q", ErrInvalidManifest, pattern)
			}
		}
	}
	for _, target := range manifest.Targets {
		for _, dependency := range uniqueSorted(target.DependsOn) {
			if dependency == target.Name {
				return fmt.Errorf("%w: target %q depends on itself", ErrInvalidManifest, target.Name)
			}
			if _, exists := seen[dependency]; !exists {
				return fmt.Errorf("%w: target %q references unknown dependency %q", ErrInvalidManifest, target.Name, dependency)
			}
		}
	}
	if len(manifest.ImportOrder) > 0 {
		orderSeen := map[string]struct{}{}
		for _, name := range manifest.ImportOrder {
			if _, exists := seen[name]; !exists {
				return fmt.Errorf("%w: import order references unknown target %q", ErrInvalidManifest, name)
			}
			if _, exists := orderSeen[name]; exists {
				return fmt.Errorf("%w: duplicate import order target %q", ErrInvalidManifest, name)
			}
			orderSeen[name] = struct{}{}
		}
		for name := range seen {
			if _, exists := orderSeen[name]; !exists {
				return fmt.Errorf("%w: import order omits target %q", ErrInvalidManifest, name)
			}
		}
	}
	return nil
}

func BuildPlan(manifest Manifest, files []string) (Plan, error) {
	if err := ValidateManifest(manifest); err != nil {
		return Plan{}, err
	}
	plan := Plan{Ambiguous: map[string][]string{}, ImportOrder: normalizeOrder(manifest)}
	for _, file := range uniqueSorted(files) {
		if !validPath(file) {
			plan.Unassigned = append(plan.Unassigned, file)
			continue
		}
		owners := Classify(file, manifest.Targets)
		switch len(owners) {
		case 0:
			plan.Unassigned = append(plan.Unassigned, file)
		case 1:
			plan.Assignments = append(plan.Assignments, Assignment{Path: file, Target: owners[0]})
		default:
			plan.Ambiguous[file] = owners
		}
	}
	plan.Summaries = summarizeTargets(manifest.Targets, plan.Assignments)
	if manifest.FailAmbiguous && len(plan.Ambiguous) > 0 {
		return plan, ErrAmbiguousFile
	}
	if manifest.FailAmbiguous && len(plan.Unassigned) > 0 {
		return plan, ErrUnassignedFile
	}
	return plan, nil
}

func Classify(path string, targets []Target) []string {
	path = normalizePath(path)
	owners := make([]string, 0, 1)
	for _, target := range targets {
		if matchesAny(path, target.Exclude) {
			continue
		}
		if matchesAny(path, target.Include) {
			owners = append(owners, target.Name)
		}
	}
	sort.Strings(owners)
	return owners
}

func (p Plan) Ready() bool {
	return len(p.Ambiguous) == 0 && len(p.Unassigned) == 0
}

func (p Plan) CountByTarget() map[string]int {
	counts := map[string]int{}
	for _, assignment := range p.Assignments {
		counts[assignment.Target]++
	}
	return counts
}

func (p Plan) FilesForTarget(target string) []string {
	files := []string{}
	for _, assignment := range p.Assignments {
		if assignment.Target == target {
			files = append(files, assignment.Path)
		}
	}
	sort.Strings(files)
	return files
}

func (p Plan) OwnershipDigest() string {
	lines := make([]string, 0, len(p.Assignments))
	for _, assignment := range p.Assignments {
		lines = append(lines, assignment.Path+"\t"+assignment.Target)
	}
	sort.Strings(lines)
	digest := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(digest[:])
}

func GenerateReadme(target Target) string {
	return fmt.Sprintf(`# %s

Status: generated repository export.

## Purpose

%s

## Source of truth

Project-level product truth remains in the KYBa knowledge cell. This repository owns only the local implementation and documentation assigned by the export manifest.

## Repository dependencies

%s

## Update rule

When local behavior changes, update this repository and exported/imported repository interfaces affected by the change.
`, target.Name, target.Description, bulletList(target.DependsOn))
}

func GenerateValidation(target Target) string {
	return fmt.Sprintf(`# Validation for %s

Status: generated export note.

Run repository-local validation after import. Do not treat archive generation as implementation validation.

Declared dependencies:

%s
`, target.Name, bulletList(target.DependsOn))
}

func GenerateRepositoryInterface(target Target) string {
	return fmt.Sprintf(`id: kyba.repo.%s.interface.v1
kind: repository-interface
owner_repo: %s
version: 0.1.0
visibility: imported-repos-only
description: %q
dependencies:
%s
`, target.Name, target.Name, target.Description, yamlList(target.DependsOn))
}

func GenerateImportedCapsulesNote(target Target) string {
	return fmt.Sprintf(`# Imported Capsules

Status: generated placeholder.

This directory is reserved for materialized repository-interface/context capsules imported through Gitea.

Declared dependencies:

%s

Rules:

1. Imported capsule snapshots are generated artifacts.
2. Agents may read local files and these imported capsules.
3. Agents must not read sibling repositories directly unless a task explicitly authorizes it.
4. Capsule freshness and compatibility are checked by kyba-ci and surfaced by Gitea.
`, bulletList(target.DependsOn))
}

func summarizeTargets(targets []Target, assignments []Assignment) []TargetSummary {
	counts := map[string]int{}
	for _, assignment := range assignments {
		counts[assignment.Target]++
	}
	summaries := make([]TargetSummary, 0, len(targets))
	for _, target := range targets {
		summaries = append(summaries, TargetSummary{Target: target.Name, Description: target.Description, Files: counts[target.Name], DependsOn: uniqueSorted(target.DependsOn), GeneratedFiles: generatedFileCount(target)})
	}
	sort.Slice(summaries, func(i, j int) bool { return summaries[i].Target < summaries[j].Target })
	return summaries
}

func generatedFileCount(target Target) int {
	if target.GeneratedReadme {
		return 5
	}
	return 0
}

func normalizeOrder(manifest Manifest) []string {
	if len(manifest.ImportOrder) > 0 {
		return append([]string{}, manifest.ImportOrder...)
	}
	order := make([]string, 0, len(manifest.Targets))
	for _, target := range manifest.Targets {
		order = append(order, target.Name)
	}
	sort.Strings(order)
	return order
}

func validTargetName(name string) bool {
	if name == "" || strings.ContainsAny(name, " /\\") {
		return false
	}
	return regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`).MatchString(name)
}

func validPattern(pattern string) bool {
	pattern = normalizePath(pattern)
	return pattern != "" && validPath(pattern)
}

func validPath(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" || strings.HasPrefix(path, "/") || strings.Contains(path, "\x00") {
		return false
	}
	for _, part := range strings.Split(strings.ReplaceAll(path, "\\", "/"), "/") {
		if part == ".." {
			return false
		}
	}
	return true
}

func matchesAny(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if matchPattern(path, pattern) {
			return true
		}
	}
	return false
}

func matchPattern(path, pattern string) bool {
	pattern = normalizePath(pattern)
	path = normalizePath(path)
	if pattern == path {
		return true
	}
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return path == prefix || strings.HasPrefix(path, prefix+"/")
	}
	regex := regexp.QuoteMeta(pattern)
	regex = strings.ReplaceAll(regex, `\*\*`, `.*`)
	regex = strings.ReplaceAll(regex, `\*`, `[^/]*`)
	matched, _ := regexp.MatchString("^"+regex+"$", path)
	return matched
}

func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "\\", "/")
	path = strings.TrimPrefix(path, "./")
	return path
}

func uniqueSorted(values []string) []string {
	seen := map[string]struct{}{}
	for _, value := range values {
		value = normalizePath(value)
		if value != "" {
			seen[value] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func bulletList(values []string) string {
	values = uniqueSorted(values)
	if len(values) == 0 {
		return "- none"
	}
	lines := make([]string, 0, len(values))
	for _, value := range values {
		lines = append(lines, "- `"+value+"`")
	}
	return strings.Join(lines, "\n")
}

func yamlList(values []string) string {
	values = uniqueSorted(values)
	if len(values) == 0 {
		return "  []"
	}
	lines := make([]string, 0, len(values))
	for _, value := range values {
		lines = append(lines, "  - "+value)
	}
	return strings.Join(lines, "\n")
}
