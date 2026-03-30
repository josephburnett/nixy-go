package worlds

import "github.com/josephburnett/nixy-go/pkg/file"

func Laptop() *file.F {
	return &file.F{
		Type: file.Folder, Owner: file.OwnerRoot,
		OwnerPermission: file.Write, CommonPermission: file.Read,
		Files: map[string]*file.F{
			"bin": {Type: file.Folder, Owner: file.OwnerRoot,
				OwnerPermission: file.Write, CommonPermission: file.Read,
				Files: map[string]*file.F{
					"ssh": {Type: file.Binary, Owner: file.OwnerRoot,
						OwnerPermission: file.Write, CommonPermission: file.Read, Data: "ssh"},
					"ls": {Type: file.Binary, Owner: file.OwnerRoot,
						OwnerPermission: file.Write, CommonPermission: file.Read, Data: "ls"},
					"cat": {Type: file.Binary, Owner: file.OwnerRoot,
						OwnerPermission: file.Write, CommonPermission: file.Read, Data: "cat"},
					"pwd": {Type: file.Binary, Owner: file.OwnerRoot,
						OwnerPermission: file.Write, CommonPermission: file.Read, Data: "pwd"},
				}},
			"etc": {Type: file.Folder, Owner: file.OwnerRoot,
				OwnerPermission: file.Write, CommonPermission: file.Read,
				Files: map[string]*file.F{
					"hosts": {Type: file.Text, Owner: file.OwnerRoot,
						OwnerPermission: file.Write, CommonPermission: file.Read,
						Data: "laptop\nnixy"},
				}},
			"home": {Type: file.Folder, Owner: file.OwnerRoot,
				OwnerPermission: file.Write, CommonPermission: file.Read,
				Files: map[string]*file.F{
					"user": {Type: file.Folder, Owner: "user",
						OwnerPermission: file.Write, CommonPermission: file.Read,
						Files: map[string]*file.F{}},
				}},
		},
	}
}
