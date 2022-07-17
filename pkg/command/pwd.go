package command

import (
	"strings"

	"github.com/josephburnett/nixy-go/pkg/binary"
	"github.com/josephburnett/nixy-go/pkg/process"
)

func init() {
	binary.Register("pwd", pwd)
}

func pwd(args binary.Args) (process.Process, error) {
	return &singleValueProcess{
		parent: args.Parent,
		value:  strings.Join(args.Directory, "/"),
	}, nil
}
