package session

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/process"
)

// TestPlaythrough plays the whole game pressing only the hint keys, as
// a player who always takes the green key would, and keeps a
// transcript of what they'd see: Nixy's dialog, the thought line at
// each prompt, and the terminal. Set NIXY_TRANSCRIPT=FILE to write it
// out and read the game as a player meets it.
func TestPlaythrough(t *testing.T) {
	sess, term := newLoggedIn(t, "player")
	var out strings.Builder
	dialogSeen := 0
	thought := ""
	flush := func() {
		for _, l := range term.State.Lines {
			switch {
			case l.Prompt.IsZero():
				fmt.Fprintf(&out, "%s\n", l.Input)
			case l.Prompt.Raw != "":
				fmt.Fprintf(&out, "%s%s\n", l.Prompt.Raw, l.Input)
			default:
				fmt.Fprintf(&out, "%s@%s:%s> %s\n", l.Prompt.User, l.Prompt.Host, l.Prompt.Path, l.Input)
			}
		}
		term.State.Lines = nil
		for _, d := range term.State.Dialog[dialogSeen:] {
			if d.Text != "" {
				fmt.Fprintf(&out, "    NIXY: %s\n", d.Text)
			}
		}
		dialogSeen = len(term.State.Dialog)
		if term.State.Thought != thought && term.State.Thought != "" {
			fmt.Fprintf(&out, "    (%s...)\n", term.State.Thought)
		}
		thought = term.State.Thought
	}
	flush()
	keys := 0
	for ; sess.Game.Manager.ActiveQuest() != nil; keys++ {
		if keys > hintFinishKeys {
			t.Fatalf("hints didn't finish: quest %q", activeQuestID(sess))
		}
		hint := term.State.HintKey
		if hint == nil {
			t.Fatalf("no hint at %q in quest %q", term.State.Line, activeQuestID(sess))
		}
		pressAccepted(t, sess, term, hint)
		if hint == process.Datum(process.SigInt) || hint == process.Datum(process.TermEnter) {
			flush()
		}
	}
	flush()
	typeCommand(t, sess, term, "nx log")
	flush()
	t.Logf("finished in %d keys", keys)
	// What the player made and read along the way looks like real
	// text files: `wc -l` agrees with what cat shows.
	for _, h := range []string{"laptop", "nixy", "server"} {
		c, _ := sess.Game.Sim.GetComputer(h)
		walkText(c.Filesystem, "/", func(path, data string) {
			if data != "" && !strings.HasSuffix(data, "\n") {
				t.Errorf("%s:%s doesn't end in a newline", h, path)
			}
		})
	}
	if path := os.Getenv("NIXY_TRANSCRIPT"); path != "" {
		if err := os.WriteFile(path, []byte(out.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func walkText(f *file.F, path string, fn func(path, data string)) {
	if f.Type == file.Text {
		fn(path, f.Data)
	}
	for n, c := range f.Files {
		walkText(c, strings.TrimSuffix(path, "/")+"/"+n, fn)
	}
}
