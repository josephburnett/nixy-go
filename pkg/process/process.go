package process

type Process interface {
	Read() (string, error)
	Write(string) error
	Kill() error
}

type ProcessSpace struct {
	next      int
	Processes map[int]*Process
}

func (ps *ProcessSpace) Add(p *Process) error {
	ps.Processes[ps.next] = p
	ps.next ++
}
