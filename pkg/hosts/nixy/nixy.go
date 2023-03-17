package nixy

import "github.com/josephburnett/nixy-go/pkg/file"

var Filesystem = &file.F{
	Type:             file.Folder,
	Owner:            file.OwnerRoot,
	OwnerPermission:  file.Write,
	CommonPermission: file.Read,
	Files: map[string]*file.F{
		"bin": &file.F{
			Type:             file.Folder,
			Owner:            file.OwnerRoot,
			OwnerPermission:  file.Write,
			CommonPermission: file.Read,
			Files: map[string]*file.F{
				"pwd": &file.F{
					Type:             file.Binary,
					Owner:            file.OwnerRoot,
					OwnerPermission:  file.Write,
					CommonPermission: file.Read,
					Data:             "pwd",
				},
			},
		},
	},
}
