package wc

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/josephburnett/nixy-go/pkg/command"
	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

func init() {
	simulation.Register("wc", &simulation.Binary{
		Launch: launch,
		// `wc [-lwc] FILE`, or `wc [-lwc]` reading a pipe: counts of
		// lines, words and bytes (all three without options).
		ValidArgs: command.Positional{
			Args:       []command.Arg{command.FilePath},
			Required:   1,
			Flags:      "clw",
			Writes:     true,
			ReadsStdin: true,
		}.Validate,
	})
}

func launch(sim *simulation.S, owner, hostname string, cwd, args []string) (process.P, error) {
	flags, args := command.Options(args)
	if flags == "" {
		flags = "lwc"
	}
	if len(args) == 0 {
		return &stdinWc{owner: owner, flags: flags}, nil
	}
	c, err := sim.GetComputer(hostname)
	if err != nil {
		return nil, err
	}
	data, err := c.Filesystem.ReadFile(file.Resolve(cwd, args[0]), owner)
	if err != nil {
		return command.Fail(owner, "wc", args[0], err), nil
	}
	return command.NewSingleValueProcess(owner, count(data, flags, args[0])), nil
}

// count formats the counts in wc's order (lines, words, bytes). For a
// file, columns are as wide as its byte count, and the name follows;
// for stdin they are 7 wide, as GNU wc prints them.
func count(data, flags, name string) string {
	n := map[rune]int{
		'l': strings.Count(data, "\n"),
		'w': len(strings.Fields(data)),
		'c': len(data),
	}
	width := 7
	if name != "" {
		width = len(strconv.Itoa(n['c']))
	}
	var cols []string
	for _, f := range "lwc" {
		if strings.ContainsRune(flags, f) {
			cols = append(cols, fmt.Sprintf("%*d", width, n[f]))
		}
	}
	if len(cols) == 1 {
		cols[0] = strings.TrimLeft(cols[0], " ") // a single count isn't padded
	}
	if name != "" {
		cols = append(cols, name)
	}
	return strings.Join(cols, " ") + "\n"
}

// stdinWc counts its input once it closes.
type stdinWc struct {
	process.Written
	owner string
	flags string
	buf   strings.Builder
}

func (w *stdinWc) Stdin(in process.Data) error {
	w.buf.WriteString(in.String())
	return nil
}

func (w *stdinWc) CloseStdin() {
	w.Write(process.Stdout, process.CharsData(count(w.buf.String(), w.flags, "")))
	w.Finish(0)
}

func (w *stdinWc) Next() []process.Datum { return nil }
func (w *stdinWc) Owner() string         { return w.owner }
