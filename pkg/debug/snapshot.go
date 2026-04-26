// Package debug captures session snapshots for post-mortem inspection.
// A Recorder maintains a small ring of snapshots; on a debug keystroke
// (Ctrl+\ in the CLI) the caller dumps the ring to a file so the bug can
// be reproduced or shared.
package debug

import (
	"fmt"
	"strings"
	"time"

	"github.com/josephburnett/nixy-go/pkg/game"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/session"
	"github.com/josephburnett/nixy-go/pkg/terminal"
)

// Snapshot is a flat, serializable view of session + terminal state at one
// instant. It deliberately copies primitives and small slices only — the
// filesystem and process tree are skipped because they're large and rarely
// the proximate cause of a hang.
type Snapshot struct {
	When     time.Time
	Datum    string
	Mode     string
	Username string
	Hostname string
	Cwd      string
	CurCmd   string
	Line     string
	Prompt   string
	Notice   string
	Thought  string
	Active   string
	Quests   []QuestState
	Tracker  []TrackerRow
	History  []string
}

type QuestState struct {
	ID    string
	State string
}

type TrackerRow struct {
	Host string
	Cwd  string
	Cmd  string
}

// Capture takes a snapshot of the current session and terminal state. The
// datum is the keystroke that produced this state (or nil for the initial
// snapshot).
func Capture(sess *session.Session, term *terminal.T, datum process.Datum) Snapshot {
	s := Snapshot{
		When:     time.Now(),
		Datum:    datumString(datum),
		Mode:     modeString(sess.Mode),
		Username: sess.Username,
		Line:     term.State.Line,
		Prompt:   promptText(term.State.Prompt),
		Notice:   term.State.Notice,
		Thought:  term.State.Thought,
	}
	if sess.Shell != nil {
		s.Hostname = sess.Shell.Hostname()
		s.Cwd = "/" + strings.Join(sess.Shell.CurrentDirectory(), "/")
		s.CurCmd = sess.Shell.CurrentCommand()
	}
	if sess.Game != nil {
		mgr := sess.Game.Manager
		if active := mgr.ActiveQuest(); active != nil {
			s.Active = active.ID()
		}
		for _, q := range mgr.Quests() {
			s.Quests = append(s.Quests, QuestState{
				ID:    q.ID(),
				State: questStateString(mgr.GetQuestState(q.ID())),
			})
		}
		for _, r := range lastN(mgr.Tracker.Records(), 20) {
			s.Tracker = append(s.Tracker, TrackerRow{
				Host: r.Hostname,
				Cwd:  "/" + strings.Join(r.Cwd, "/"),
				Cmd:  r.Command,
			})
		}
	}
	for _, h := range lastNHistory(term.State.Lines, 8) {
		if h.Prompt.IsZero() {
			s.History = append(s.History, "  "+h.Input)
		} else {
			s.History = append(s.History, promptText(h.Prompt)+h.Input)
		}
	}
	return s
}

// Recorder is a fixed-size ring of snapshots.
type Recorder struct {
	cap     int
	entries []Snapshot
}

func NewRecorder(cap int) *Recorder {
	if cap < 1 {
		cap = 1
	}
	return &Recorder{cap: cap}
}

// Push appends a fresh snapshot; if the ring is full, the oldest is dropped.
func (r *Recorder) Push(sess *session.Session, term *terminal.T, datum process.Datum) {
	r.entries = append(r.entries, Capture(sess, term, datum))
	if len(r.entries) > r.cap {
		r.entries = r.entries[len(r.entries)-r.cap:]
	}
}

// Dump returns a human-readable text rendering of the entire ring.
func (r *Recorder) Dump() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "=== nixy snapshot ring (%d entries) ===\n\n", len(r.entries))
	for i, s := range r.entries {
		fmt.Fprintf(&sb, "--- snapshot %d/%d @ %s ---\n",
			i+1, len(r.entries), s.When.Format("15:04:05.000"))
		writeSnapshot(&sb, s)
		sb.WriteString("\n")
	}
	return sb.String()
}

func writeSnapshot(sb *strings.Builder, s Snapshot) {
	fmt.Fprintf(sb, "keystroke: %s\n", s.Datum)
	fmt.Fprintf(sb, "mode:      %s\n", s.Mode)
	fmt.Fprintf(sb, "user:      %q\n", s.Username)
	fmt.Fprintf(sb, "shell:     %s @ %s   curCmd=%q\n", s.Hostname, s.Cwd, s.CurCmd)
	fmt.Fprintf(sb, "prompt:    %s\n", s.Prompt)
	fmt.Fprintf(sb, "line:      %q\n", s.Line)
	fmt.Fprintf(sb, "notice:    %q\n", s.Notice)
	fmt.Fprintf(sb, "thought:   %q\n", s.Thought)
	fmt.Fprintf(sb, "active:    %s\n", s.Active)
	if len(s.Quests) > 0 {
		sb.WriteString("quests:\n")
		for _, q := range s.Quests {
			fmt.Fprintf(sb, "  %-14s %s\n", q.ID, q.State)
		}
	}
	if len(s.Tracker) > 0 {
		fmt.Fprintf(sb, "tracker (%d, last %d shown):\n", len(s.Tracker), len(s.Tracker))
		for _, r := range s.Tracker {
			fmt.Fprintf(sb, "  %-8s %-20s %s\n", r.Host, r.Cwd, r.Cmd)
		}
	}
	if len(s.History) > 0 {
		sb.WriteString("history:\n")
		for _, h := range s.History {
			fmt.Fprintf(sb, "  %s\n", h)
		}
	}
}

func datumString(d process.Datum) string {
	if d == nil {
		return "(initial)"
	}
	switch v := d.(type) {
	case process.Chars:
		return fmt.Sprintf("Chars(%q)", string(v))
	case process.TermCode:
		return string(v)
	case process.Signal:
		return string(v)
	default:
		return fmt.Sprintf("%T(%v)", d, d)
	}
}

func modeString(m session.Mode) string {
	switch m {
	case session.ModeLogin:
		return "Login"
	case session.ModePlaying:
		return "Playing"
	default:
		return fmt.Sprintf("Mode(%d)", int(m))
	}
}

func questStateString(s game.QuestState) string {
	switch s {
	case game.QuestInactive:
		return "Inactive"
	case game.QuestActive:
		return "Active"
	case game.QuestComplete:
		return "Complete"
	default:
		return fmt.Sprintf("QuestState(%d)", int(s))
	}
}

func promptText(p terminal.PromptInfo) string {
	if p.Raw != "" {
		return p.Raw
	}
	if p.IsZero() {
		return "> "
	}
	return p.User + "@" + p.Host + ":" + p.Path + "> "
}

func lastN(rs []game.CommandRecord, n int) []game.CommandRecord {
	if len(rs) <= n {
		return rs
	}
	return rs[len(rs)-n:]
}

func lastNHistory(rs []terminal.HistoryLine, n int) []terminal.HistoryLine {
	if len(rs) <= n {
		return rs
	}
	return rs[len(rs)-n:]
}
