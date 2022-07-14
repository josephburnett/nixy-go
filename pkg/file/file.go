package file

type File struct {
	Type             FileType
	Owner            string
	OwnerPermission  FilePermission
	CommonPermission FilePermission
	Data             string
}

type FileType string

const (
	Text   FileType = "Text"
	Binary FileType = "Binary"
)

type FilePermission string

const (
	Read    FilePermission = "Read"
	Write   FilePermission = "Write"
	Execute FilePermission = "Execute"
)
