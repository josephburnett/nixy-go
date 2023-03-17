package file

type F struct {
	Type             FileType
	Owner            string
	OwnerPermission  FilePermission
	CommonPermission FilePermission
	Data             string
	Files            map[string]*F
}

type FileType string

const (
	Folder FileType = "Folder"
	Text   FileType = "Text"
	Binary FileType = "Binary"
)

type FilePermission string

const (
	None  FilePermission = "None"
	Read  FilePermission = "Read"
	Write FilePermission = "Write"
)

const (
	OwnerRoot = "root"
)
