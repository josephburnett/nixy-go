package process

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
