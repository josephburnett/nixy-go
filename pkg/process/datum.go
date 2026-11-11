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

// Signals a process can be sent (see P.Signal). SigInt is also a key:
// Ctrl+C, which the shell turns into SIGINT for its foreground job.
var (
	SigHup  Signal = "SigHup"
	SigInt  Signal = "SigInt"
	SigKill Signal = "SigKill"
	SigPipe Signal = "SigPipe"
	SigTerm Signal = "SigTerm"

	TermBackspace TermCode = "TermBackspace"
	TermEnter     TermCode = "TermEnter"
	TermClear     TermCode = "TermClear"
)

// Number is the signal's number on Linux, as kill -l lists it. A
// process killed by a signal exits with status 128 + Number.
func (s Signal) Number() int {
	switch s {
	case SigHup:
		return 1
	case SigInt:
		return 2
	case SigKill:
		return 9
	case SigPipe:
		return 13
	case SigTerm:
		return 15
	}
	return 0
}

// Status is the exit status of a process killed by s.
func (s Signal) Status() int { return 128 + s.Number() }

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
