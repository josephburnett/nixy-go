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

func TestPlanHintBackspace(t *testing.T) {
	q := newPlannerQuest("cd /home", false)
	hint := PlanHint(q, nil, nil, "", nil, "ls")
	if tc, ok := hint.(process.TermCode); !ok || tc != process.TermBackspace {
		t.Fatalf("expected TermBackspace, got %v", hint)
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

func TestPlanHintBackspaceOnDivergence(t *testing.T) {
	q := newPlannerQuest("cd /home", false)
	hint := PlanHint(q, nil, nil, "", nil, "cd /etc")
	if tc, ok := hint.(process.TermCode); !ok || tc != process.TermBackspace {
		t.Fatalf("expected TermBackspace, got %v", hint)
	}
}
