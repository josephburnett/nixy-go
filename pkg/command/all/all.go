// Package all registers every command. Front ends and tests that run
// the whole game import it, so a new command is added in one place.
package all

import (
	_ "github.com/josephburnett/nixy-go/pkg/command/apt"
	_ "github.com/josephburnett/nixy-go/pkg/command/cat"
	_ "github.com/josephburnett/nixy-go/pkg/command/echo"
	_ "github.com/josephburnett/nixy-go/pkg/command/grep"
	_ "github.com/josephburnett/nixy-go/pkg/command/kill"
	_ "github.com/josephburnett/nixy-go/pkg/command/ls"
	_ "github.com/josephburnett/nixy-go/pkg/command/mkdir"
	_ "github.com/josephburnett/nixy-go/pkg/command/mv"
	_ "github.com/josephburnett/nixy-go/pkg/command/ps"
	_ "github.com/josephburnett/nixy-go/pkg/command/pwd"
	_ "github.com/josephburnett/nixy-go/pkg/command/rm"
	_ "github.com/josephburnett/nixy-go/pkg/command/rmdir"
	_ "github.com/josephburnett/nixy-go/pkg/command/shell"
	_ "github.com/josephburnett/nixy-go/pkg/command/ssh"
	_ "github.com/josephburnett/nixy-go/pkg/command/sudo"
	_ "github.com/josephburnett/nixy-go/pkg/command/touch"
	_ "github.com/josephburnett/nixy-go/pkg/command/whoami"
	_ "github.com/josephburnett/nixy-go/pkg/command/yes"
)
