package game

import (
	"testing"

	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

// mockQuest implements Quest for testing the planner.
type mockQuest struct {
	id      string
	target  string
	done    bool
	machine string
}

func (q *mockQuest) ID() string                       { return q.id }
func (q *mockQuest) Description() string               { return "test quest" }
func (q *mockQuest) Machine() string                   { return q.machine }
func (q *mockQuest) RequiredAchievements() []Achievement { return nil }
func (q *mockQuest) GrantedAchievements() []Achievement  { return nil }
func (q *mockQuest) Setup(_ *simulation.S) error       { return nil }
func (q *mockQuest) IsComplete(_ *simulation.S, _ *CommandTracker) bool { return q.done }

func (q *mockQuest) PlanNextCommand(_ *simulation.S, _ *CommandTracker, _ string, _ []string) string {
	if q.done {
		return ""
	}
	return q.target
}

func TestPlanHintNextChar(t *testing.T) {
	q := &mockQuest{target: "cd /home"}
	hint := PlanHint(q, nil, nil, "", nil,"cd")
	if c, ok := hint.(process.Chars); !ok || string(c) != " " {
		t.Fatalf("expected space, got %v", hint)
	}
}

func TestPlanHintFirstChar(t *testing.T) {
	q := &mockQuest{target: "ssh nixy"}
	hint := PlanHint(q, nil, nil, "", nil,"")
	if c, ok := hint.(process.Chars); !ok || string(c) != "s" {
		t.Fatalf("expected 's', got %v", hint)
	}
}

func TestPlanHintEnter(t *testing.T) {
	q := &mockQuest{target: "pwd"}
	hint := PlanHint(q, nil, nil, "", nil,"pwd")
	if tc, ok := hint.(process.TermCode); !ok || tc != process.TermEnter {
		t.Fatalf("expected TermEnter, got %v", hint)
	}
}

func TestPlanHintBackspace(t *testing.T) {
	q := &mockQuest{target: "cd /home"}
	hint := PlanHint(q, nil, nil, "", nil,"ls")
	if tc, ok := hint.(process.TermCode); !ok || tc != process.TermBackspace {
		t.Fatalf("expected TermBackspace, got %v", hint)
	}
}

func TestPlanHintQuestComplete(t *testing.T) {
	q := &mockQuest{target: "", done: true}
	hint := PlanHint(q, nil, nil, "", nil,"")
	if hint != nil {
		t.Fatalf("expected nil for complete quest, got %v", hint)
	}
}

func TestPlanHintNilQuest(t *testing.T) {
	hint := PlanHint(nil, nil, nil, "", nil,"")
	if hint != nil {
		t.Fatalf("expected nil for nil quest, got %v", hint)
	}
}

func TestPlanHintPartialMatch(t *testing.T) {
	q := &mockQuest{target: "apt install grep"}
	// Type "apt " (correct prefix)
	hint := PlanHint(q, nil, nil, "", nil,"apt ")
	if c, ok := hint.(process.Chars); !ok || string(c) != "i" {
		t.Fatalf("expected 'i', got %v", hint)
	}
}

func TestPlanHintBackspaceOnDivergence(t *testing.T) {
	q := &mockQuest{target: "cd /home"}
	// Type "cd /etc" — diverges at position 4
	hint := PlanHint(q, nil, nil, "", nil,"cd /etc")
	if tc, ok := hint.(process.TermCode); !ok || tc != process.TermBackspace {
		t.Fatalf("expected TermBackspace, got %v", hint)
	}
}
