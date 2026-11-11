package worlds

import "github.com/josephburnett/nixy-go/pkg/file"

func Nixy() *file.F {
	return &file.F{
		Type: file.Folder, Owner: file.OwnerRoot,
		OwnerPermission: file.Write, CommonPermission: file.Read,
		Files: map[string]*file.F{
			"bin": {Type: file.Folder, Owner: file.OwnerRoot,
				OwnerPermission: file.Write, CommonPermission: file.Read,
				// grep, rm and touch ship so the early quests need no apt:
				// installing software takes root, which the Permissions
				// quest teaches. sudo is for Lockdown, once Nixy locks
				// her home. wc and yes come later, from apt.
				Files: programs("ls", "cat", "pwd", "apt", "grep", "rm", "touch", "sudo",
					"whoami", "echo", "mkdir", "man", "ps", "kill")},
			"etc": {Type: file.Folder, Owner: file.OwnerRoot,
				OwnerPermission: file.Write, CommonPermission: file.Read,
				Files: map[string]*file.F{
					"hosts": {Type: file.Text, Owner: file.OwnerRoot,
						OwnerPermission: file.Write, CommonPermission: file.Read,
						Data: "nixy\nlaptop\n"},
				}},
			// root's home: only root may look inside.
			"root": {Type: file.Folder, Owner: file.OwnerRoot,
				OwnerPermission: file.Write, CommonPermission: file.None,
				Files: map[string]*file.F{}},
			"home": {Type: file.Folder, Owner: file.OwnerRoot,
				OwnerPermission: file.Write, CommonPermission: file.Read,
				Files: map[string]*file.F{
					// Nixy's home. CommonPermission is Write to begin with: Nixy is
					// loose with permissions, so the early quests need no sudo.
					// The Lockdown quest tightens it to Read, and sudo becomes the
					// way back in.
					"nixy": {Type: file.Folder, Owner: "nixy",
						OwnerPermission: file.Write, CommonPermission: file.Write,
						Files: map[string]*file.F{
							"readme.txt": {Type: file.Text, Owner: "nixy",
								OwnerPermission: file.Write, CommonPermission: file.Read,
								Data: "Welcome to Nixy! I'm glad you're here.\nPlease look around and help me out!\n"},
						}},
				}},
			"var": {Type: file.Folder, Owner: file.OwnerRoot,
				OwnerPermission: file.Write, CommonPermission: file.Read,
				Files: map[string]*file.F{
					"log": {Type: file.Folder, Owner: file.OwnerRoot,
						OwnerPermission: file.Write, CommonPermission: file.Read,
						Files: map[string]*file.F{
							"system.log": {Type: file.Text, Owner: file.OwnerRoot,
								OwnerPermission: file.Write, CommonPermission: file.Read,
								Data: "info: system started\ninfo: services running\nerror: disk space low\ninfo: backup complete\nerror: network timeout\n"},
						}},
				}},
		},
	}
}
