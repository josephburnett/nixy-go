package man

// Pages are the manual pages, by the binary each describes. They say
// what this game's command does, in the shape of a real page: real man
// pages are longer, but nothing here should be untrue of them.
var Pages = map[string]string{
	"apt": `APT(8)
NAME
    apt - install software packages
SYNOPSIS
    apt install PACKAGE
    apt list
DESCRIPTION
    apt install puts a package's command in /bin. Changing /bin
    takes root, so run it with sudo. apt list shows the packages.
`,
	"cat": `CAT(1)
NAME
    cat - print files
SYNOPSIS
    cat FILE
    COMMAND | cat
DESCRIPTION
    Print the contents of FILE. With no FILE, print what comes
    through the pipe.
`,
	"echo": `ECHO(1)
NAME
    echo - print words
SYNOPSIS
    echo [-n] [WORD...]
DESCRIPTION
    Print the WORDs, separated by spaces, and a newline.
    -n  leave off the newline
    Try: echo hello > note.txt
`,
	"grep": `GREP(1)
NAME
    grep - print lines that match a pattern
SYNOPSIS
    grep PATTERN FILE
    COMMAND | grep PATTERN
DESCRIPTION
    Print the lines of FILE (or of the pipe) that contain PATTERN.
    Exit status is 1 when no line matches.
`,
	"head": `HEAD(1)
NAME
    head - print the first lines
SYNOPSIS
    head FILE
    COMMAND | head
DESCRIPTION
    Print the first 10 lines of FILE, or of the pipe.
`,
	"kill": `KILL(1)
NAME
    kill - send a signal to a process
SYNOPSIS
    kill [-9] PID
DESCRIPTION
    Send SIGTERM to process PID, asking it to end. -9 sends SIGKILL,
    which no process can ignore. You may signal only your own
    processes, unless you are root. Find PIDs with ps.
`,
	"ls": `LS(1)
NAME
    ls - list folder contents
SYNOPSIS
    ls [-a] [PATH]
DESCRIPTION
    List the folder PATH (or the current folder). Names starting
    with . are hidden.
    -a  list all names, hidden ones and . and .. too
`,
	"man": `MAN(1)
NAME
    man - show a command's manual
SYNOPSIS
    man COMMAND
DESCRIPTION
    Show the manual page for COMMAND. For shell builtins such as
    cd, use help.
`,
	"mkdir": `MKDIR(1)
NAME
    mkdir - make a folder
SYNOPSIS
    mkdir DIR
DESCRIPTION
    Make the folder DIR. You need write permission on the folder
    it goes in. The new folder is yours.
`,
	"mv": `MV(1)
NAME
    mv - move or rename a file
SYNOPSIS
    mv SOURCE DEST
DESCRIPTION
    Rename SOURCE to DEST, or move it into DEST if DEST is a folder.
`,
	"ps": `PS(1)
NAME
    ps - list processes
SYNOPSIS
    ps [-e]
DESCRIPTION
    List your running processes: their PID, owner and command.
    -e  list everyone's
`,
	"pwd": `PWD(1)
NAME
    pwd - print the current folder
SYNOPSIS
    pwd
`,
	"rm": `RM(1)
NAME
    rm - remove a file
SYNOPSIS
    rm FILE
DESCRIPTION
    Remove FILE. There is no undo. Folders are removed with rmdir.
`,
	"rmdir": `RMDIR(1)
NAME
    rmdir - remove an empty folder
SYNOPSIS
    rmdir DIR
DESCRIPTION
    Remove the folder DIR, which must be empty.
`,
	"shell": `SHELL(1)
NAME
    shell - the command interpreter
DESCRIPTION
    Reads command lines and runs them. COMMAND | COMMAND pipes the
    first's output into the second; COMMAND > FILE writes it to FILE
    (>> adds to the end). Ctrl+C interrupts a running command.
    ~ is your home folder. Builtins: see help.
`,
	"ssh": `SSH(1)
NAME
    ssh - log in to another machine
SYNOPSIS
    ssh HOST
DESCRIPTION
    Open a shell on HOST, one of the names in /etc/hosts. exit
    comes back.
`,
	"sudo": `SUDO(8)
NAME
    sudo - run a command as root
SYNOPSIS
    sudo COMMAND [ARGS...]
DESCRIPTION
    Run COMMAND as root, who may read and change any file.
    A redirect (> FILE) is still done by your shell, as you.
`,
	"touch": `TOUCH(1)
NAME
    touch - create an empty file
SYNOPSIS
    touch FILE
DESCRIPTION
    Create FILE if it doesn't exist. The new file is yours.
`,
	"wc": `WC(1)
NAME
    wc - count lines, words and bytes
SYNOPSIS
    wc [-lwc] FILE
    COMMAND | wc [-lwc]
DESCRIPTION
    Print the number of lines, words and bytes in FILE or the pipe.
    -l lines  -w words  -c bytes
`,
	"whoami": `WHOAMI(1)
NAME
    whoami - print your user name
SYNOPSIS
    whoami
DESCRIPTION
    Print the name of the user running the command: you, or root
    under sudo.
`,
	"yes": `YES(1)
NAME
    yes - print a line forever
SYNOPSIS
    yes [WORD...]
DESCRIPTION
    Print y (or the WORDs) over and over, until interrupted with
    Ctrl+C or its reader goes away (yes | head).
`,
}
