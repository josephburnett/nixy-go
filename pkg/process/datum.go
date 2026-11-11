package process

import "strings"

type Datum interface {
	isDatum()
}
type Data []Datum

type Chars string
type Signal string
type TermCode string

func (c Chars) isDatum()    {}
func (s Signal) isDatum()   {}
func (t TermCode) isDatum() {}

var (
	SigHup  Signal = "SigHup"
	SigKill Signal = "SigKill"
	SigTerm Signal = "SigTerm"

	TermBackspace TermCode = "TermBackspace"
	TermEnter     TermCode = "TermEnter"
	TermClear     TermCode = "TermClear"
)

func CharsData(s string) Data {
	return Data{Chars(s)}
}

// String returns the characters in d, skipping key codes and signals.
func (d Data) String() string {
	var sb strings.Builder
	for _, x := range d {
		if c, ok := x.(Chars); ok {
			sb.WriteString(string(c))
		}
	}
	return sb.String()
}
