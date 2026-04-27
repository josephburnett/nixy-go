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
				}},
			"etc": {Type: file.Folder, Owner: file.OwnerRoot,
				OwnerPermission: file.Write, CommonPermission: file.Read,
				Files: map[string]*file.F{
					"hosts": {Type: file.Text, Owner: file.OwnerRoot,
						OwnerPermission: file.Write, CommonPermission: file.Read,
						Data: "nixy\nlaptop"},
				}},
			"home": {Type: file.Folder, Owner: file.OwnerRoot,
				OwnerPermission: file.Write, CommonPermission: file.Read,
				Files: map[string]*file.F{
					// Nixy's home. CommonPermission is Write for now — Nixy is loose
					// with permissions so the player can clean up junk.txt without
					// sudo. The future Permissions quest will tighten this back to
					// Read and teach sudo as the way in.
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
