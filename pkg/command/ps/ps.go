package ps

import (
	"fmt"
	"strings"

	"github.com/josephburnett/nixy-go/pkg/command"
	"github.com/josephburnett/nixy-go/pkg/computer"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

func init() {
	simulation.Register("ps", &simulation.Binary{
		Launch: launch,
		// `ps` lists your processes, `ps -e` everyone's, with the
		// columns of `ps -o pid,user,cmd`.
		ValidArgs: command.Positional{Flags: "e", Writes: true}.Validate,
	})
}

func launch(sim *simulation.S, owner, hostname string, _, args []string) (process.P, error) {
	c, err := sim.GetComputer(hostname)
	if err != nil {
		return nil, err
	}
	flags, _ := command.Options(args)
	return &ps{owner: owner, c: c, all: strings.ContainsRune(flags, 'e')}, nil
}

// ps reads the process table when first read, not at launch: by then
// it is in the table itself, as real ps lists itself (and the rest of
// its pipeline).
type ps struct {
	process.Written
	owner string
	c     *computer.C
	all   bool
}

func (p *ps) Read() []process.Output {
	if _, exited := p.Exit(); !exited {
		var sb strings.Builder
		fmt.Fprintf(&sb, "%5s %-8s %s\n", "PID", "USER", "CMD")
		for _, info := range p.c.Processes() {
			if p.all || info.Owner == p.owner {
				fmt.Fprintf(&sb, "%5d %-8s %s\n", info.PID, info.Owner, info.Command)
			}
		}
		p.Write(process.Stdout, process.CharsData(sb.String()))
		p.Finish(0)
	}
	return p.Written.Read()
}

func (p *ps) Stdin(process.Data) error { return process.ErrNoInput }
func (p *ps) Next() []process.Datum    { return nil }
func (p *ps) Owner() string            { return p.owner }
