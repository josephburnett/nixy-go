package head

import (
	"strings"

	"github.com/josephburnett/nixy-go/pkg/command"
	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

func init() {
	simulation.Register("head", &simulation.Binary{
		Launch: launch,
		// `head FILE`, or bare `head` reading a pipe: the first lines.
		ValidArgs: command.Positional{
			Args:       []command.Arg{command.FilePath},
			Required:   1,
			Writes:     true,
			ReadsStdin: true,
		}.Validate,
	})
}

// lines is how many lines head prints, as real head's default.
const lines = 10

func launch(sim *simulation.S, owner, hostname string, cwd, args []string) (process.P, error) {
	if len(args) == 0 {
		return &stdinHead{owner: owner}, nil
	}
	c, err := sim.GetComputer(hostname)
	if err != nil {
		return nil, err
	}
	data, err := c.Filesystem.ReadFile(file.Resolve(cwd, args[0]), owner)
	if err != nil {
		return command.Fail(owner, "head", "cannot open '"+args[0]+"' for reading", err), nil
	}
	if data != "" && !strings.HasSuffix(data, "\n") {
		data += "\n" // as cat does, so the prompt starts a new line
	}
	h := &stdinHead{owner: owner}
	h.take(data)
	h.CloseStdin()
	return h, nil
}

// stdinHead copies lines until it has printed enough, then exits; a
// writer still sending gets SIGPIPE (`yes | head`).
type stdinHead struct {
	process.Written
	owner   string
	printed int
	partial string
}

func (h *stdinHead) Stdin(in process.Data) error {
	h.take(in.String())
	return nil
}

func (h *stdinHead) take(data string) {
	data = h.partial + data
	h.partial = ""
	for data != "" && h.printed < lines {
		line, rest, ended := strings.Cut(data, "\n")
		if !ended {
			h.partial = line
			return
		}
		h.Write(process.Stdout, process.CharsData(line+"\n"))
		h.printed++
		data = rest
	}
	if h.printed >= lines {
		h.Finish(0)
	}
}

// CloseStdin prints a last line that had no newline, as head does.
func (h *stdinHead) CloseStdin() {
	if h.partial != "" && h.printed < lines {
		h.Write(process.Stdout, process.CharsData(h.partial))
	}
	h.Finish(0)
}

func (h *stdinHead) Next() []process.Datum { return nil }
func (h *stdinHead) Owner() string         { return h.owner }
