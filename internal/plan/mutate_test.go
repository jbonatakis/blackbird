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
