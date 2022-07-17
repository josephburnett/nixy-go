package command

import "fmt"

var (
	ErrEndOfFile       = fmt.Errorf("end of file")
	ErrReadOnlyProcess = fmt.Errorf("read only process")
)
