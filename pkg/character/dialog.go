package character

// DialogTrigger identifies when dialog should appear.
type DialogTrigger int

const (
	OnQuestActivate DialogTrigger = iota
	OnQuestComplete
)

// DialogEntry is a single dialog that can be triggered.
type DialogEntry struct {
	Trigger DialogTrigger
	QuestID string
	Lines   []string
}

// DialogQueue manages pending dialog lines.
type DialogQueue struct {
	pending []string
}

func NewDialogQueue() *DialogQueue {
	return &DialogQueue{}
}

// Enqueue adds dialog lines to be displayed.
func (q *DialogQueue) Enqueue(lines []string) {
	q.pending = append(q.pending, lines...)
}

// Drain returns all pending lines and clears the queue.
func (q *DialogQueue) Drain() []string {
	lines := q.pending
	q.pending = nil
	return lines
}

// HasPending returns true if there are dialog lines waiting.
func (q *DialogQueue) HasPending() bool {
	return len(q.pending) > 0
}
