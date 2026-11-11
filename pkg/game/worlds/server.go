package worlds

import "github.com/josephburnett/nixy-go/pkg/file"

func Server() *file.F {
	return &file.F{
		Type: file.Folder, Owner: file.OwnerRoot,
		OwnerPermission: file.Write, CommonPermission: file.Read,
		Files: map[string]*file.F{
			"bin": {Type: file.Folder, Owner: file.OwnerRoot,
				OwnerPermission: file.Write, CommonPermission: file.Read,
				Files: map[string]*file.F{
					"ls": {Type: file.Binary, Owner: file.OwnerRoot,
						OwnerPermission: file.Write, CommonPermission: file.Read, Data: "ls"},
					"cat": {Type: file.Binary, Owner: file.OwnerRoot,
						OwnerPermission: file.Write, CommonPermission: file.Read, Data: "cat"},
					"pwd": {Type: file.Binary, Owner: file.OwnerRoot,
						OwnerPermission: file.Write, CommonPermission: file.Read, Data: "pwd"},
					"sudo": {Type: file.Binary, Owner: file.OwnerRoot,
						OwnerPermission: file.Write, CommonPermission: file.Read, Data: "sudo"},
					"touch": {Type: file.Binary, Owner: file.OwnerRoot,
						OwnerPermission: file.Write, CommonPermission: file.Read, Data: "touch"},
					"apt": {Type: file.Binary, Owner: file.OwnerRoot,
						OwnerPermission: file.Write, CommonPermission: file.Read, Data: "apt"},
				}},
			// /etc is world-readable, as in real Unix, but only root
			// can write to it: that's what the Permissions quest's sudo
			// is for.
			"etc": {Type: file.Folder, Owner: file.OwnerRoot,
				OwnerPermission: file.Write, CommonPermission: file.Read,
				Files: map[string]*file.F{
					"hosts": {Type: file.Text, Owner: file.OwnerRoot,
						OwnerPermission: file.Write, CommonPermission: file.Read,
						Data: "server\nnixy\nlaptop"},
				}},
			// /home starts empty; the player's /home/<username> is created at
			// game bootstrap (see MachineRegistry.provisionUserHome).
			// root's home: only root may look inside.
			"root": {Type: file.Folder, Owner: file.OwnerRoot,
				OwnerPermission: file.Write, CommonPermission: file.None,
				Files: map[string]*file.F{}},
			"home": {Type: file.Folder, Owner: file.OwnerRoot,
				OwnerPermission: file.Write, CommonPermission: file.Read,
				Files: map[string]*file.F{}},
		},
	}
}
