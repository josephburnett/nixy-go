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

func (g *G) Read() (out process.Data, eof bool, err error) {
	return g.process.Read()
}

func (g *G) Write(in process.Data) (eof bool, err error) {
	errs := g.process.Test([]process.Data{in})
	if len(errs) != 1 {
		return false, fmt.Errorf("guide: tested 1 data but got %v errs", len(errs))
	}
	err = errs[0]
	if err != nil {
		// Filter out invalid input
		return false, err
	}
	return g.process.Write(in)
}
