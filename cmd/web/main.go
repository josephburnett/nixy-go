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

	"github.com/josephburnett/nixy-go/pkg/debug"
	"github.com/josephburnett/nixy-go/pkg/process"
	"github.com/josephburnett/nixy-go/pkg/session"
	"github.com/josephburnett/nixy-go/pkg/terminal"
)

const debugRing = 10

var (
	sess     *session.Session
	term     *terminal.T
	recorder *debug.Recorder
)

func main() {
	var err error
	sess, err = session.New()
	if err != nil {
		js.Global().Get("console").Call("error", "nixy init error: "+err.Error())
		return
	}

	term = terminal.New(terminal.NewHTML())
	sess.InitTerminal(term)

	recorder = debug.NewRecorder(debugRing)
	recorder.Push(sess, term, nil)

	js.Global().Set("nixyInit", js.FuncOf(handleInit))
	js.Global().Set("nixyKeystroke", js.FuncOf(handleKeystroke))
	js.Global().Set("nixyResize", js.FuncOf(handleResize))
	js.Global().Set("nixyDumpState", js.FuncOf(handleDump))

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
		runes := []rune(key)
		if len(runes) == 1 {
			datum = process.Chars(key)
		}
	}

	if datum == nil {
		return term.Render()
	}

	sess.HandleKeystroke(datum, term)
	recorder.Push(sess, term, datum)
	return term.Render()
}

func handleDump(_ js.Value, _ []js.Value) any {
	return recorder.Dump()
}

func handleResize(_ js.Value, args []js.Value) any {
	if len(args) < 2 {
		return nil
	}
	term.Resize(args[0].Int(), args[1].Int())
	return term.Render()
}
