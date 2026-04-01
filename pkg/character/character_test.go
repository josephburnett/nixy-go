package character

import "testing"

func TestDialogQueueEnqueueAndDrain(t *testing.T) {
	q := NewDialogQueue()
	q.Enqueue([]string{"Hello", "World"})
	lines := q.Drain()
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0] != "Hello" || lines[1] != "World" {
		t.Fatalf("expected [Hello World], got %v", lines)
	}
}

func TestDialogQueueDrainClears(t *testing.T) {
	q := NewDialogQueue()
	q.Enqueue([]string{"line"})
	q.Drain()
	lines := q.Drain()
	if len(lines) != 0 {
		t.Fatalf("expected empty after drain, got %v", lines)
	}
}

func TestDialogQueueHasPending(t *testing.T) {
	q := NewDialogQueue()
	if q.HasPending() {
		t.Fatal("expected no pending initially")
	}
	q.Enqueue([]string{"line"})
	if !q.HasPending() {
		t.Fatal("expected pending after enqueue")
	}
	q.Drain()
	if q.HasPending() {
		t.Fatal("expected no pending after drain")
	}
}

func TestDialogQueueMultipleEnqueues(t *testing.T) {
	q := NewDialogQueue()
	q.Enqueue([]string{"a"})
	q.Enqueue([]string{"b", "c"})
	lines := q.Drain()
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[0] != "a" || lines[1] != "b" || lines[2] != "c" {
		t.Fatalf("expected [a b c], got %v", lines)
	}
}

func TestFindDialog(t *testing.T) {
	entries := AllDialog()

	lines := FindDialog(entries, OnQuestActivate, "connect")
	if lines == nil {
		t.Fatal("expected dialog for connect activation")
	}
	if len(lines) == 0 {
		t.Fatal("expected non-empty dialog")
	}

	lines = FindDialog(entries, OnQuestComplete, "connect")
	if lines == nil {
		t.Fatal("expected dialog for connect completion")
	}
}

func TestFindDialogNotFound(t *testing.T) {
	entries := AllDialog()
	lines := FindDialog(entries, OnQuestActivate, "nonexistent")
	if lines != nil {
		t.Fatalf("expected nil for unknown quest, got %v", lines)
	}
}

func TestAllDialogCoversAllQuests(t *testing.T) {
	entries := AllDialog()
	questIDs := []string{"connect", "orientation", "inspection", "modification", "composition", "permissions"}

	for _, id := range questIDs {
		activate := FindDialog(entries, OnQuestActivate, id)
		if activate == nil {
			t.Fatalf("missing activation dialog for quest %q", id)
		}
		complete := FindDialog(entries, OnQuestComplete, id)
		if complete == nil {
			t.Fatalf("missing completion dialog for quest %q", id)
		}
	}
}
