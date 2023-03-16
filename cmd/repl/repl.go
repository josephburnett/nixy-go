package main

import (
	"fmt"
	"io"
	"os"
	"unicode/utf8"

	_ "github.com/josephburnett/nixy-go/pkg/command/shell"
	"github.com/josephburnett/nixy-go/pkg/computer"
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
			shell.Write(data)
		}
	}
}

func launch() (*terminal.T, process.P, error) {
	sim := simulation.New()
	comp := computer.New(nil)
	err := comp.Boot()
	if err != nil {
		return nil, nil, err
	}
	err = sim.AddComputer("repl", comp)
	if err != nil {
		return nil, nil, err
	}
	ctx := simulation.Context{
		Simulation: sim,
	}
	proc, err := sim.Launch("repl", "shell", ctx, "", nil)
	if err != nil {
		return nil, nil, err
	}
	t := terminal.New()
	return t, proc, nil
}
