package main

import (
	"fmt"
	"io"
	"os"
	"unicode/utf8"

	_ "github.com/josephburnett/nixy-go/pkg/command/shell"
	"github.com/josephburnett/nixy-go/pkg/guide"
	"github.com/josephburnett/nixy-go/pkg/hosts/nixy"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/simulation"
	"github.com/josephburnett/nixy-go/pkg/terminal"
)

func main() {
	t, shell, err := launch()
	if err != nil {
		panic(err)
	}
	in := make([]byte, 1)
	for {
		// Read shell
		out, eof, err := shell.Read()
		if err != nil {
			panic(err)
		}
		if eof {
			return
		}
		// Write term
		err = t.Write(out)
		if err != nil {
			panic(err)
		}
		// Render term
		view := t.Render()
		fmt.Print(view)
		// Read keyboard
		count, err := os.Stdin.Read(in)
		if err == io.EOF {
			return
		}
		if err != nil {
			panic(err)
		}
		// Write shell
		if count > 0 {
			r, _ := utf8.DecodeRune(in)
			s := string(r)
			data := process.CharsData(s)
			if s == ">" {
				data[0] = process.TermEnter
			}
			if s == "<" {
				data[0] = process.TermBackspace
			}
			if s == "^" {
				data[0] = process.TermClear
			}
			if s == "\n" {
				continue
			}
			_, err := shell.Write(data)
			t.Hint(err)
		}
	}
}

func launch() (*terminal.T, *guide.G, error) {
	sim := simulation.New()
	err := sim.Boot("repl", nixy.Filesystem)
	if err != nil {
		return nil, nil, err
	}
	proc, err := sim.Launch("repl", "root", "shell", "", nil)
	if err != nil {
		return nil, nil, err
	}
	g := guide.New(proc)
	t := terminal.New()
	return t, g, nil
}
