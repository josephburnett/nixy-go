//go:build js && wasm

package main

import (
	"syscall/js"

	_ "github.com/josephburnett/nixy-go/pkg/command/apt"
	_ "github.com/josephburnett/nixy-go/pkg/command/cat"
	_ "github.com/josephburnett/nixy-go/pkg/command/grep"
	_ "github.com/josephburnett/nixy-go/pkg/command/ls"
	_ "github.com/josephburnett/nixy-go/pkg/command/mv"
	_ "github.com/josephburnett/nixy-go/pkg/command/pwd"
	_ "github.com/josephburnett/nixy-go/pkg/command/rm"
	_ "github.com/josephburnett/nixy-go/pkg/command/shell"
	_ "github.com/josephburnett/nixy-go/pkg/command/ssh"
	_ "github.com/josephburnett/nixy-go/pkg/command/sudo"
	_ "github.com/josephburnett/nixy-go/pkg/command/touch"

	"github.com/josephburnett/nixy-go/pkg/game"
	"github.com/josephburnett/nixy-go/pkg/guide"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/session"
	"github.com/josephburnett/nixy-go/pkg/terminal"
)

var (
	g    *game.Game
	gd   *guide.G
	sh   session.ShellInfo
	term *terminal.T
)

func main() {
	sess, err := session.New()
	if err != nil {
		js.Global().Get("console").Call("error", "nixy init error: "+err.Error())
		return
	}

	g = sess.Game
	gd = sess.Guide
	sh = sess.Shell
	term = terminal.New(terminal.NewHTML())

	js.Global().Set("nixyInit", js.FuncOf(handleInit))
	js.Global().Set("nixyKeystroke", js.FuncOf(handleKeystroke))
	js.Global().Set("nixyResize", js.FuncOf(handleResize))

	// Block forever — WASM must not return from main.
	select {}
}

func handleInit(_ js.Value, _ []js.Value) any {
	return term.Render()
}

func handleKeystroke(_ js.Value, args []js.Value) any {
	if len(args) == 0 {
		return nil
	}
	key := args[0].String()

	var datum process.Datum
	switch key {
	case "Enter":
		datum = process.TermEnter
	case "Backspace":
		datum = process.TermBackspace
	default:
		// Single printable character
		runes := []rune(key)
		if len(runes) == 1 {
			datum = process.Chars(key)
		}
	}

	if datum == nil {
		return term.Render()
	}

	// Write to shell through guide
	_, err := gd.Stdin(process.Data{datum})
	term.Hint(err)

	// Drain stdout
	for i := 0; i < 50; i++ {
		out, eof, _ := gd.Stdout()
		if eof {
			break
		}
		if len(out) > 0 {
			term.Write(out)
		} else {
			break
		}
	}

	// Drain stderr
	for i := 0; i < 10; i++ {
		errOut, _, _ := gd.Stderr()
		if len(errOut) > 0 {
			term.Write(errOut)
		} else {
			break
		}
	}

	// After Enter, check quest state and dialog
	if _, ok := datum.(process.TermCode); ok && datum == process.TermEnter {
		g.AfterCommand()
		dialog := g.Manager.Dialog.Drain()
		if len(dialog) > 0 {
			term.SetDialog(dialog)
		}
	}

	// Update keyboard display
	valid := gd.Next()
	hint := g.GetHint(sh.Hostname(), sh.CurrentDirectory(), sh.CurrentCommand())
	term.SetKeyboard(valid, hint)

	return term.Render()
}

func handleResize(_ js.Value, args []js.Value) any {
	if len(args) < 2 {
		return nil
	}
	term.Resize(args[0].Int(), args[1].Int())
	return term.Render()
}
