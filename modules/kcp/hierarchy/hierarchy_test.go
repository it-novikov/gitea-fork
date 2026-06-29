package hierarchy

import (
	"errors"
	"testing"
)

func id(value int64) *int64 { return &value }

func newTwoProjectTree(t *testing.T) *Tree {
	t.Helper()
	tree := NewTree()
	if err := tree.CreateProject(Project{ID: 1, OwnerID: 10, Name: "Alpha", Slug: "alpha"}); err != nil {
		t.Fatal(err)
	}
	if err := tree.CreateProject(Project{ID: 2, OwnerID: 10, Name: "Beta", Slug: "beta"}); err != nil {
		t.Fatal(err)
	}
	return tree
}

func TestMoveSubtreeAcrossProjectsMovesRepositoryPlacements(t *testing.T) {
	tree := newTwoProjectTree(t)
	for _, group := range []RepositoryGroup{
		{ID: 100, ProjectID: 1, Name: "Platform"},
		{ID: 101, ProjectID: 1, ParentID: id(100), Name: "Services"},
		{ID: 102, ProjectID: 1, ParentID: id(101), Name: "Identity"},
	} {
		if err := tree.CreateGroup(group); err != nil {
			t.Fatal(err)
		}
	}
	placement, err := tree.PlaceRepository(500, 1, id(102))
	if err != nil {
		t.Fatal(err)
	}
	moved, err := tree.MoveGroup(100, 2, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	if moved.ProjectID != 2 || moved.Version != 2 {
		t.Fatalf("unexpected moved root: %+v", moved)
	}
	snapshot := tree.Snapshot()
	for _, group := range snapshot.Groups {
		if group.ProjectID != 2 {
			t.Fatalf("group %d did not move with subtree", group.ID)
		}
	}
	if got := snapshot.Placements[0]; got.RepositoryID != placement.RepositoryID || got.ProjectID != 2 || got.Version != 2 {
		t.Fatalf("repository placement did not move atomically: %+v", got)
	}
}

func TestMoveRejectsCyclesAndDuplicateSiblingNames(t *testing.T) {
	tree := newTwoProjectTree(t)
	if err := tree.CreateGroup(RepositoryGroup{ID: 100, ProjectID: 1, Name: "Parent"}); err != nil {
		t.Fatal(err)
	}
	if err := tree.CreateGroup(RepositoryGroup{ID: 101, ProjectID: 1, ParentID: id(100), Name: "Child"}); err != nil {
		t.Fatal(err)
	}
	if _, err := tree.MoveGroup(100, 1, id(101), 1); !errors.Is(err, ErrCycle) {
		t.Fatalf("expected cycle error, got %v", err)
	}
	if err := tree.CreateGroup(RepositoryGroup{ID: 102, ProjectID: 1, Name: " parent "}); !errors.Is(err, ErrDuplicateName) {
		t.Fatalf("expected case-insensitive duplicate error, got %v", err)
	}
}

func TestRepositoryMoveUsesOptimisticVersionAndPreservesIdentity(t *testing.T) {
	tree := newTwoProjectTree(t)
	if err := tree.CreateGroup(RepositoryGroup{ID: 200, ProjectID: 2, Name: "Destination"}); err != nil {
		t.Fatal(err)
	}
	if _, err := tree.PlaceRepository(900, 1, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := tree.MoveRepository(900, 2, id(200), 2); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected version conflict, got %v", err)
	}
	moved, err := tree.MoveRepository(900, 2, id(200), 1)
	if err != nil {
		t.Fatal(err)
	}
	if moved.RepositoryID != 900 || moved.ProjectID != 2 || moved.GroupID == nil || *moved.GroupID != 200 {
		t.Fatalf("unexpected repository placement: %+v", moved)
	}
	if _, err := tree.MoveRepository(900, 1, nil, 1); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale version was accepted: %v", err)
	}
}

func TestGroupPathIsRootFirst(t *testing.T) {
	tree := newTwoProjectTree(t)
	if err := tree.CreateGroup(RepositoryGroup{ID: 1_000, ProjectID: 1, Name: "One"}); err != nil {
		t.Fatal(err)
	}
	if err := tree.CreateGroup(RepositoryGroup{ID: 1_001, ProjectID: 1, ParentID: id(1_000), Name: "Two"}); err != nil {
		t.Fatal(err)
	}
	path, err := tree.GroupPath(1_001)
	if err != nil {
		t.Fatal(err)
	}
	if len(path) != 2 || path[0].ID != 1_000 || path[1].ID != 1_001 {
		t.Fatalf("unexpected path: %+v", path)
	}
}
