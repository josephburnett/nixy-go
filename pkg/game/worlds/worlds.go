// Package worlds defines each machine's filesystem as shipped.
//
// Every machine ships the everyday basics a real Unix has everywhere
// (ls, cat, pwd, whoami, man, and on the machines the player works on,
// echo, mkdir, ps and kill). The rest are a `sudo apt install` away,
// which is where quests teach them: the server is a bare box (no touch,
// for the Permissions quest), and on nixy wc and yes come from apt.
// nx panic puts /bin back as defined here.
package worlds

import "github.com/josephburnett/nixy-go/pkg/file"

// program is /bin/<name>: root's, and anyone may run it.
func program(name string) *file.F {
	return &file.F{Type: file.Binary, Owner: file.OwnerRoot,
		OwnerPermission: file.Write, CommonPermission: file.Read, Data: name}
}

// programs is a /bin with the named programs in it.
func programs(names ...string) map[string]*file.F {
	out := map[string]*file.F{}
	for _, n := range names {
		out[n] = program(n)
	}
	return out
}
