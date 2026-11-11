package guide

import (
	"fmt"
	"slices"

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

func (g *G) Read() []process.Output { return g.process.Read() }

func (g *G) Exit() (int, bool) { return g.process.Exit() }

// Stdin passes in to the process only if it is one of the keys the
// process offers (Next). Anything else is refused with an error.
func (g *G) Stdin(in process.Data) error {
	if len(in) != 1 {
		return fmt.Errorf("guide: expected 1 datum, got %v", len(in))
	}
	if !slices.Contains(g.process.Next(), in[0]) {
		return fmt.Errorf("invalid input")
	}
	return g.process.Stdin(in)
}

func (g *G) Next() []process.Datum {
	return g.process.Next()
}
