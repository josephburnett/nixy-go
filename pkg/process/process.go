package process

type Process interface {
	Read() (Data, bool, error)
	Write(Data) error
	Owner() string
	Parent() Process
	Kill() error
}

type ProcessSpace struct {
	next      int
	processes map[int]Process
}

func NewProcessSpace() *ProcessSpace {
	return &ProcessSpace{
		processes: map[int]Process{},
	}
}

func (ps *ProcessSpace) Add(p Process) {
	ps.processes[ps.next] = p
	ps.next++
}
