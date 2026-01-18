package plan

import (
	"errors"
	"testing"
	"time"
)

func TestMoveItem_RejectsHierarchyCycle(t *testing.T) {
	now := time.Now()

	a := wi("A", now)
	b := wi("B", now)
	c := wi("C", now)

	a.ChildIDs = []string{"B"}
	b.ParentID = ptr("A")
	b.ChildIDs = []string{"C"}
	c.ParentID = ptr("B")

	g := WorkGraph{
		SchemaVersion: SchemaVersion,
		Items: map[string]WorkItem{
			"A": a,
			"B": b,
			"C": c,
		},
	}

	p := "C"
	err := MoveItem(&g, "A", &p, nil, now)
	if err == nil {
		t.Fatalf("expected error")
	}
	var ce HierarchyCycleError
	if !errors.As(err, &ce) {
		t.Fatalf("expected HierarchyCycleError, got %T: %v", err, err)
	}
	if len(ce.Cycle) < 2 || ce.Cycle[0] != "A" || ce.Cycle[len(ce.Cycle)-1] != "A" {
		t.Fatalf("unexpected cycle: %#v", ce.Cycle)
	}
}

func TestAddDep_RejectsDepCycle(t *testing.T) {
	now := time.Now()

	g := WorkGraph{
		SchemaVersion: SchemaVersion,
		Items: map[string]WorkItem{
			"A": wi("A", now),
			"B": func() WorkItem {
				it := wi("B", now)
				it.Deps = []string{"A"}
				return it
			}(),
		},
	}

	if err := AddDep(&g, "A", "B", now); err == nil {
		t.Fatalf("expected error")
	} else {
		var ce DepCycleError
		if !errors.As(err, &ce) {
			t.Fatalf("expected DepCycleError, got %T: %v", err, err)
		}
	}
}

func TestAddDep_CycleKeepsUpdatedAt(t *testing.T) {
	now := time.Now()
	after := now.Add(5 * time.Minute)

	g := WorkGraph{
		SchemaVersion: SchemaVersion,
		Items: map[string]WorkItem{
			"A": wi("A", now),
			"B": func() WorkItem {
				it := wi("B", now)
				it.Deps = []string{"A"}
				return it
			}(),
		},
	}
	beforeUpdatedAt := g.Items["A"].UpdatedAt

	if err := AddDep(&g, "A", "B", after); err == nil {
		t.Fatalf("expected error")
	}
	if got := g.Items["A"].UpdatedAt; !got.Equal(beforeUpdatedAt) {
		t.Fatalf("UpdatedAt changed on failed add: got %s want %s", got, beforeUpdatedAt)
	}
}

func TestSetDeps_CycleKeepsUpdatedAt(t *testing.T) {
	now := time.Now()
	after := now.Add(10 * time.Minute)

	g := WorkGraph{
		SchemaVersion: SchemaVersion,
		Items: map[string]WorkItem{
			"A": wi("A", now),
			"B": func() WorkItem {
				it := wi("B", now)
				it.Deps = []string{"A"}
				return it
			}(),
		},
	}
	beforeUpdatedAt := g.Items["A"].UpdatedAt

	if err := SetDeps(&g, "A", []string{"B"}, after); err == nil {
		t.Fatalf("expected error")
	}
	if got := g.Items["A"].UpdatedAt; !got.Equal(beforeUpdatedAt) {
		t.Fatalf("UpdatedAt changed on failed set: got %s want %s", got, beforeUpdatedAt)
	}
}

func TestDeleteItem_RefusesChildrenByDefault(t *testing.T) {
	now := time.Now()

	p := wi("P", now)
	c := wi("C", now)
	p.ChildIDs = []string{"C"}
	c.ParentID = ptr("P")

	g := WorkGraph{
		SchemaVersion: SchemaVersion,
		Items: map[string]WorkItem{
			"P": p,
			"C": c,
		},
	}

	if _, err := DeleteItem(&g, "P", false, false, now); err == nil {
		t.Fatalf("expected error")
	}
}

func TestDeleteItem_RefusesDependentsUnlessForce(t *testing.T) {
	now := time.Now()

	g := WorkGraph{
		SchemaVersion: SchemaVersion,
		Items: map[string]WorkItem{
			"A": wi("A", now),
			"B": func() WorkItem {
				it := wi("B", now)
				it.Deps = []string{"A"}
				return it
			}(),
		},
	}

	if _, err := DeleteItem(&g, "A", false, false, now); err == nil {
		t.Fatalf("expected error")
	}

	res, err := DeleteItem(&g, "A", false, true, now)
	if err != nil {
		t.Fatalf("DeleteItem force: %v", err)
	}
	if len(res.DeletedIDs) != 1 || res.DeletedIDs[0] != "A" {
		t.Fatalf("deleted = %#v, want [\"A\"]", res.DeletedIDs)
	}
	if _, ok := g.Items["A"]; ok {
		t.Fatalf("A should be deleted")
	}
	if got := g.Items["B"].Deps; len(got) != 0 {
		t.Fatalf("B.Deps = %#v, want []", got)
	}
	if errs := Validate(g); len(errs) != 0 {
		t.Fatalf("Validate after delete force: %v", errs)
	}
}

func TestDeleteItem_ForceDedupesDetachedIDs(t *testing.T) {
	now := time.Now()

	a := wi("A", now)
	b := wi("B", now)
	b.ParentID = ptr("A")
	a.ChildIDs = []string{"B"}

	c := wi("C", now)
	c.Deps = []string{"A", "B"}

	g := WorkGraph{
		SchemaVersion: SchemaVersion,
		Items: map[string]WorkItem{
			"A": a,
			"B": b,
			"C": c,
		},
	}

	res, err := DeleteItem(&g, "A", true, true, now)
	if err != nil {
		t.Fatalf("DeleteItem force: %v", err)
	}
	if len(res.DetachedIDs) != 1 || res.DetachedIDs[0] != "C" {
		t.Fatalf("DetachedIDs = %#v, want [\"C\"]", res.DetachedIDs)
	}
	if got := g.Items["C"].Deps; len(got) != 0 {
		t.Fatalf("C.Deps = %#v, want []", got)
	}
}

func TestParentCycleIfMove_GuardsInvalidPlan(t *testing.T) {
	now := time.Now()

	a := wi("A", now)
	b := wi("B", now)
	a.ParentID = ptr("B")
	b.ParentID = ptr("A")

	g := WorkGraph{
		SchemaVersion: SchemaVersion,
		Items: map[string]WorkItem{
			"A": a,
			"B": b,
			"C": wi("C", now),
		},
	}

	if cycle := parentCycleIfMove(&g, "C", "A"); cycle != nil {
		t.Fatalf("expected nil cycle for invalid parent loop, got %#v", cycle)
	}
}

func wi(id string, now time.Time) WorkItem {
	return WorkItem{
		ID:                 id,
		Title:              id,
		Description:        "",
		AcceptanceCriteria: []string{},
		Prompt:             "",
		ParentID:           nil,
		ChildIDs:           []string{},
		Deps:               []string{},
		Status:             StatusTodo,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}
