package debug

type Terminal struct {
	ps *process.ProcessSpace
	fs *filesystem.File
}

func NewTerminal() (*Terminal, error) {
	return &Terminal{
		ps: process.NewProcessSpace(),
		fs: filesystem: NewFileSystem(),
	}
}
