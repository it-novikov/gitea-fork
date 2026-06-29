package hierarchy

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

var (
	ErrNotFound           = errors.New("hierarchy object not found")
	ErrConflict           = errors.New("hierarchy version conflict")
	ErrDuplicateName      = errors.New("duplicate sibling group name")
	ErrCycle              = errors.New("repository group move would create a cycle")
	ErrInvalidDestination = errors.New("invalid hierarchy destination")
)

type Project struct {
	ID      int64
	OwnerID int64
	Name    string
	Slug    string
	Version int64
}

type RepositoryGroup struct {
	ID        int64
	ProjectID int64
	ParentID  *int64
	Name      string
	Version   int64
}

type RepositoryPlacement struct {
	RepositoryID int64
	ProjectID    int64
	GroupID      *int64
	Version      int64
}

type Snapshot struct {
	Projects   []Project
	Groups     []RepositoryGroup
	Placements []RepositoryPlacement
}

// Tree is a domain reference implementation for the Gitea fork overlay. The production fork must
// persist equivalent mutations transactionally in PostgreSQL and use the repository row as the
// source of repository identity, clone URL, issues, pull requests, and permissions.
type Tree struct {
	mu         sync.RWMutex
	projects   map[int64]Project
	groups     map[int64]RepositoryGroup
	placements map[int64]RepositoryPlacement
}

func NewTree() *Tree {
	return &Tree{
		projects:   make(map[int64]Project),
		groups:     make(map[int64]RepositoryGroup),
		placements: make(map[int64]RepositoryPlacement),
	}
}

func (t *Tree) CreateProject(project Project) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if project.ID <= 0 || project.OwnerID <= 0 || !validName(project.Name) || !validSlug(project.Slug) {
		return fmt.Errorf("%w: invalid project", ErrInvalidDestination)
	}
	if _, exists := t.projects[project.ID]; exists {
		return ErrConflict
	}
	for _, existing := range t.projects {
		if existing.OwnerID == project.OwnerID && strings.EqualFold(existing.Slug, project.Slug) {
			return ErrDuplicateName
		}
	}
	project.Version = 1
	t.projects[project.ID] = project
	return nil
}

func (t *Tree) CreateGroup(group RepositoryGroup) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if group.ID <= 0 || !validName(group.Name) {
		return fmt.Errorf("%w: invalid group", ErrInvalidDestination)
	}
	if _, exists := t.groups[group.ID]; exists {
		return ErrConflict
	}
	if _, exists := t.projects[group.ProjectID]; !exists {
		return ErrNotFound
	}
	if group.ParentID != nil {
		parent, exists := t.groups[*group.ParentID]
		if !exists || parent.ProjectID != group.ProjectID {
			return ErrInvalidDestination
		}
	}
	if t.siblingNameExistsLocked(group.ProjectID, group.ParentID, group.Name, 0) {
		return ErrDuplicateName
	}
	group.ParentID = cloneID(group.ParentID)
	group.Version = 1
	t.groups[group.ID] = group
	return nil
}

func (t *Tree) PlaceRepository(repositoryID, projectID int64, groupID *int64) (RepositoryPlacement, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if repositoryID <= 0 || !t.destinationExistsLocked(projectID, groupID) {
		return RepositoryPlacement{}, ErrInvalidDestination
	}
	if _, exists := t.placements[repositoryID]; exists {
		return RepositoryPlacement{}, ErrConflict
	}
	placement := RepositoryPlacement{
		RepositoryID: repositoryID,
		ProjectID:    projectID,
		GroupID:      cloneID(groupID),
		Version:      1,
	}
	t.placements[repositoryID] = placement
	return placement, nil
}

func (t *Tree) MoveRepository(
	repositoryID, destinationProjectID int64,
	destinationGroupID *int64,
	expectedVersion int64,
) (RepositoryPlacement, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	placement, exists := t.placements[repositoryID]
	if !exists {
		return RepositoryPlacement{}, ErrNotFound
	}
	if expectedVersion <= 0 || placement.Version != expectedVersion {
		return RepositoryPlacement{}, ErrConflict
	}
	if !t.destinationExistsLocked(destinationProjectID, destinationGroupID) {
		return RepositoryPlacement{}, ErrInvalidDestination
	}
	placement.ProjectID = destinationProjectID
	placement.GroupID = cloneID(destinationGroupID)
	placement.Version++
	t.placements[repositoryID] = placement
	return placement, nil
}

func (t *Tree) MoveGroup(
	groupID, destinationProjectID int64,
	destinationParentID *int64,
	expectedVersion int64,
) (RepositoryGroup, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	group, exists := t.groups[groupID]
	if !exists {
		return RepositoryGroup{}, ErrNotFound
	}
	if expectedVersion <= 0 || group.Version != expectedVersion {
		return RepositoryGroup{}, ErrConflict
	}
	if _, exists := t.projects[destinationProjectID]; !exists {
		return RepositoryGroup{}, ErrInvalidDestination
	}
	if destinationParentID != nil {
		parent, exists := t.groups[*destinationParentID]
		if !exists || parent.ProjectID != destinationProjectID {
			return RepositoryGroup{}, ErrInvalidDestination
		}
		if *destinationParentID == groupID || t.isDescendantLocked(*destinationParentID, groupID) {
			return RepositoryGroup{}, ErrCycle
		}
	}
	if t.siblingNameExistsLocked(destinationProjectID, destinationParentID, group.Name, groupID) {
		return RepositoryGroup{}, ErrDuplicateName
	}

	subtree := t.subtreeLocked(groupID)
	group.ProjectID = destinationProjectID
	group.ParentID = cloneID(destinationParentID)
	group.Version++
	t.groups[groupID] = group
	for _, descendantID := range subtree {
		if descendantID == groupID {
			continue
		}
		descendant := t.groups[descendantID]
		descendant.ProjectID = destinationProjectID
		descendant.Version++
		t.groups[descendantID] = descendant
	}
	for repositoryID, placement := range t.placements {
		if placement.GroupID != nil && contains(subtree, *placement.GroupID) {
			placement.ProjectID = destinationProjectID
			placement.Version++
			t.placements[repositoryID] = placement
		}
	}
	return group, nil
}

func (t *Tree) GroupPath(groupID int64) ([]RepositoryGroup, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	group, exists := t.groups[groupID]
	if !exists {
		return nil, ErrNotFound
	}
	path := []RepositoryGroup{copyGroup(group)}
	for group.ParentID != nil {
		parent, exists := t.groups[*group.ParentID]
		if !exists {
			return nil, fmt.Errorf("%w: broken parent reference", ErrInvalidDestination)
		}
		path = append(path, copyGroup(parent))
		group = parent
	}
	for left, right := 0, len(path)-1; left < right; left, right = left+1, right-1 {
		path[left], path[right] = path[right], path[left]
	}
	return path, nil
}

func (t *Tree) Snapshot() Snapshot {
	t.mu.RLock()
	defer t.mu.RUnlock()
	result := Snapshot{}
	for _, project := range t.projects {
		result.Projects = append(result.Projects, project)
	}
	for _, group := range t.groups {
		result.Groups = append(result.Groups, copyGroup(group))
	}
	for _, placement := range t.placements {
		result.Placements = append(result.Placements, copyPlacement(placement))
	}
	sort.Slice(result.Projects, func(i, j int) bool { return result.Projects[i].ID < result.Projects[j].ID })
	sort.Slice(result.Groups, func(i, j int) bool { return result.Groups[i].ID < result.Groups[j].ID })
	sort.Slice(result.Placements, func(i, j int) bool {
		return result.Placements[i].RepositoryID < result.Placements[j].RepositoryID
	})
	return result
}

func (t *Tree) destinationExistsLocked(projectID int64, groupID *int64) bool {
	if _, exists := t.projects[projectID]; !exists {
		return false
	}
	if groupID == nil {
		return true
	}
	group, exists := t.groups[*groupID]
	return exists && group.ProjectID == projectID
}

func (t *Tree) siblingNameExistsLocked(projectID int64, parentID *int64, name string, excludedID int64) bool {
	for _, sibling := range t.groups {
		if sibling.ID != excludedID && sibling.ProjectID == projectID && sameID(sibling.ParentID, parentID) &&
			strings.EqualFold(strings.TrimSpace(sibling.Name), strings.TrimSpace(name)) {
			return true
		}
	}
	return false
}

func (t *Tree) isDescendantLocked(candidateID, ancestorID int64) bool {
	current, exists := t.groups[candidateID]
	for exists && current.ParentID != nil {
		if *current.ParentID == ancestorID {
			return true
		}
		current, exists = t.groups[*current.ParentID]
	}
	return false
}

func (t *Tree) subtreeLocked(rootID int64) []int64 {
	result := []int64{rootID}
	for index := 0; index < len(result); index++ {
		parentID := result[index]
		for _, group := range t.groups {
			if group.ParentID != nil && *group.ParentID == parentID {
				result = append(result, group.ID)
			}
		}
	}
	return result
}

func validName(value string) bool {
	trimmed := strings.TrimSpace(value)
	return len(trimmed) > 0 && len(trimmed) <= 128 && !strings.ContainsAny(trimmed, "\x00\r\n")
}

func validSlug(value string) bool {
	if len(value) < 1 || len(value) > 64 {
		return false
	}
	for index, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || (index > 0 && char == '-') {
			continue
		}
		return false
	}
	return !strings.HasSuffix(value, "-")
}

func cloneID(value *int64) *int64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func sameID(first, second *int64) bool {
	return (first == nil && second == nil) || (first != nil && second != nil && *first == *second)
}

func contains(values []int64, expected int64) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func copyGroup(group RepositoryGroup) RepositoryGroup {
	group.ParentID = cloneID(group.ParentID)
	return group
}

func copyPlacement(placement RepositoryPlacement) RepositoryPlacement {
	placement.GroupID = cloneID(placement.GroupID)
	return placement
}
