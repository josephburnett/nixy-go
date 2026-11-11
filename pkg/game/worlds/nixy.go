package worlds

import "github.com/josephburnett/nixy-go/pkg/file"

func Nixy() *file.F {
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
					"apt": {Type: file.Binary, Owner: file.OwnerRoot,
						OwnerPermission: file.Write, CommonPermission: file.Read, Data: "apt"},
					// Shipped, so the early quests need no apt: installing
					// software takes root, which the Permissions quest
					// teaches.
					"grep": {Type: file.Binary, Owner: file.OwnerRoot,
						OwnerPermission: file.Write, CommonPermission: file.Read, Data: "grep"},
					"rm": {Type: file.Binary, Owner: file.OwnerRoot,
						OwnerPermission: file.Write, CommonPermission: file.Read, Data: "rm"},
					"touch": {Type: file.Binary, Owner: file.OwnerRoot,
						OwnerPermission: file.Write, CommonPermission: file.Read, Data: "touch"},
					// For the Lockdown quest, once Nixy locks her home.
					"sudo": {Type: file.Binary, Owner: file.OwnerRoot,
						OwnerPermission: file.Write, CommonPermission: file.Read, Data: "sudo"},
				}},
			"etc": {Type: file.Folder, Owner: file.OwnerRoot,
				OwnerPermission: file.Write, CommonPermission: file.Read,
				Files: map[string]*file.F{
					"hosts": {Type: file.Text, Owner: file.OwnerRoot,
						OwnerPermission: file.Write, CommonPermission: file.Read,
						Data: "nixy\nlaptop"},
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
								Data: "Welcome to Nixy! I'm glad you're here.\nPlease look around and help me out!"},
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
								Data: "info: system started\ninfo: services running\nerror: disk space low\ninfo: backup complete\nerror: network timeout"},
						}},
				}},
		},
	}
}
