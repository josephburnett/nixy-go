package main

import (
	"github.com/josephburnett/nixy-go/pkg/computer"
	"github.com/josephburnett/nixy-go/pkg/environment"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/term"
)

func main() {
	t, shell, err := launch()
	if err != nil {
		panic(err)
	}
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

		// Read keyboard

		// Write shell

	}
}

func launch() (*term.Term, process.Process, error) {
	env, err := environment.NewEnvironment(nil)
	comp := computer.NewComputer(nil)
	env.Add("repl", comp)
	if err != nil {
		return nil, nil, err
	}
	ctx := environment.Context{
		Env: env,
	}
	proc, err := env.Launch("repl", "shell", ctx, "", nil)
	if err != nil {
		return nil, nil, err
	}
	t := term.NewTerm()
	return t, proc, nil
}
