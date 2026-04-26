package game

import (
	"testing"

	"github.com/josephburnett/nixy-go/pkg/process"
)

func newPlannerQuest(target string, done bool) *mockQuest {
	q := &mockQuest{id: "plan-test", machine: "m"}
	q.complete = done
	q.planTarget = target
	return q
}

func TestPlanHintNextChar(t *testing.T) {
	q := newPlannerQuest("cd /home", false)
	hint := PlanHint(q, nil, nil, "", nil, "cd")
	if c, ok := hint.(process.Chars); !ok || string(c) != " " {
		t.Fatalf("expected space, got %v", hint)
	}
}

func TestPlanHintFirstChar(t *testing.T) {
	q := newPlannerQuest("ssh nixy", false)
	hint := PlanHint(q, nil, nil, "", nil, "")
	if c, ok := hint.(process.Chars); !ok || string(c) != "s" {
		t.Fatalf("expected 's', got %v", hint)
	}
}

func TestPlanHintEnter(t *testing.T) {
	q := newPlannerQuest("pwd", false)
	hint := PlanHint(q, nil, nil, "", nil, "pwd")
	if tc, ok := hint.(process.TermCode); !ok || tc != process.TermEnter {
		t.Fatalf("expected TermEnter, got %v", hint)
	}
}

// TestPlanHintNoNagWhenOffPlan: when the player runs a different valid
// command (e.g. `ls` while plan is `cd /home`), the planner should NOT
// suggest Backspace. Returning nil keeps the keyboard quiet and lets
// the player finish their command; the plan re-engages on the next
// empty prompt.
func TestPlanHintNoNagWhenOffPlan(t *testing.T) {
	q := newPlannerQuest("cd /home", false)
	hint := PlanHint(q, nil, nil, "", nil, "ls")
	if hint != nil {
		t.Fatalf("expected nil hint when off-plan, got %v", hint)
	}
}

func TestPlanHintQuestComplete(t *testing.T) {
	q := newPlannerQuest("", true)
	hint := PlanHint(q, nil, nil, "", nil, "")
	if hint != nil {
		t.Fatalf("expected nil for complete quest, got %v", hint)
	}
}

func TestPlanHintNilQuest(t *testing.T) {
	hint := PlanHint(nil, nil, nil, "", nil, "")
	if hint != nil {
		t.Fatalf("expected nil for nil quest, got %v", hint)
	}
}

func TestPlanHintPartialMatch(t *testing.T) {
	q := newPlannerQuest("apt install grep", false)
	hint := PlanHint(q, nil, nil, "", nil, "apt ")
	if c, ok := hint.(process.Chars); !ok || string(c) != "i" {
		t.Fatalf("expected 'i', got %v", hint)
	}
}

// TestPlanHintNoNagOnDivergence: same principle as the no-nag test, but
// for a partial divergence (typed prefix of plan then went off — e.g.
// `cd /etc` instead of `cd /home`). Still no Backspace nag.
func TestPlanHintNoNagOnDivergence(t *testing.T) {
	q := newPlannerQuest("cd /home", false)
	hint := PlanHint(q, nil, nil, "", nil, "cd /etc")
	if hint != nil {
		t.Fatalf("expected nil hint on divergence, got %v", hint)
	}
}
