package guide

import (
	"fmt"

	"github.com/josephburnett/nixy-go/pkg/process"
)

type G struct {
	process process.P
}

func New(p process.P) *G {
	return &G{
		process: p,
	}
}

func (g *G) Stdout() (out process.Data, eof bool, err error) {
	return g.process.Stdout()
}

func (g *G) Stderr() (out process.Data, eof bool, err error) {
	return g.process.Stderr()
}

func (g *G) Stdin(in process.Data) (eof bool, err error) {
	if len(in) != 1 {
		return false, fmt.Errorf("guide: expected 1 datum, got %v", len(in))
	}
	valid := g.process.Next()
	for _, v := range valid {
		if datumEqual(in[0], v) {
			return g.process.Stdin(in)
		}
	}
	return false, fmt.Errorf("invalid input")
}

func (g *G) Next() []process.Datum {
	return g.process.Next()
}

func datumEqual(a, b process.Datum) bool {
	switch a := a.(type) {
	case process.Chars:
		if b, ok := b.(process.Chars); ok {
			return a == b
		}
	case process.TermCode:
		if b, ok := b.(process.TermCode); ok {
			return a == b
		}
	case process.Signal:
		if b, ok := b.(process.Signal); ok {
			return a == b
		}
	}
	return false
}
