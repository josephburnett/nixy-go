package process

type Datum interface {
	isDatum()
}

type Chars string
type CharPotentialProbe string
type CharPotential map[rune]Potential
type Guide string
type Signal string
type TermCode string

type Potential struct {
	// err     error
	// command bool
	// guide   bool
}

func (c Chars) isDatum()              {}
func (c CharPotentialProbe) isDatum() {}
func (c CharPotential) isDatum()      {}
func (g Guide) isDatum()              {}
func (s Signal) isDatum()             {}
func (t TermCode) isDatum()           {}

var (
	SigHup  Signal = "SigHup"
	SigKill Signal = "SigKill"
	SigTerm Signal = "SigTerm"

	TermBackspace TermCode = "TermBackspace"
	TermEnter     TermCode = "TermEnter"
	TermClear     TermCode = "TermClear"
)
