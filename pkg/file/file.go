package file

import "fmt"

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

// Navigate walks the filesystem tree following the given path components.
// Returns the file at the end of the path.
func (f *F) Navigate(path []string) (*F, error) {
	current := f
	for _, p := range path {
		if current.Type != Folder {
			return nil, fmt.Errorf("%v is not a folder", p)
		}
		child, ok := current.Files[p]
		if !ok {
			return nil, fmt.Errorf("file %v not found", p)
		}
		current = child
	}
	return current, nil
}

// CanRead returns true if the given user can read this file.
func (f *F) CanRead(user string) bool {
	if user == OwnerRoot {
		return true
	}
	if user == f.Owner {
		return f.OwnerPermission == Read || f.OwnerPermission == Write
	}
	return f.CommonPermission == Read || f.CommonPermission == Write
}

// CanWrite returns true if the given user can write to this file.
func (f *F) CanWrite(user string) bool {
	if user == OwnerRoot {
		return true
	}
	if user == f.Owner {
		return f.OwnerPermission == Write
	}
	return f.CommonPermission == Write
}

// CreateFile creates a new file at path/name. The parent folder at path must
// exist and the user must have write permission on it.
func (f *F) CreateFile(path []string, name string, newFile *F, user string) error {
	parent, err := f.Navigate(path)
	if err != nil {
		return err
	}
	if parent.Type != Folder {
		return fmt.Errorf("%v is not a folder", path)
	}
	if !parent.CanWrite(user) {
		return fmt.Errorf("permission denied")
	}
	if _, exists := parent.Files[name]; exists {
		return fmt.Errorf("file %v already exists", name)
	}
	if parent.Files == nil {
		parent.Files = map[string]*F{}
	}
	parent.Files[name] = newFile
	return nil
}

// DeleteFile removes the file at path/name. The user must have write permission
// on the parent folder.
func (f *F) DeleteFile(path []string, name string, user string) error {
	parent, err := f.Navigate(path)
	if err != nil {
		return err
	}
	if parent.Type != Folder {
		return fmt.Errorf("%v is not a folder", path)
	}
	if !parent.CanWrite(user) {
		return fmt.Errorf("permission denied")
	}
	if _, exists := parent.Files[name]; !exists {
		return fmt.Errorf("file %v not found", name)
	}
	delete(parent.Files, name)
	return nil
}

// SetData updates the data content of the file at path/name. The user must
// have write permission on the file itself.
func (f *F) SetData(path []string, name string, data string, user string) error {
	parent, err := f.Navigate(path)
	if err != nil {
		return err
	}
	if parent.Type != Folder {
		return fmt.Errorf("%v is not a folder", path)
	}
	target, ok := parent.Files[name]
	if !ok {
		return fmt.Errorf("file %v not found", name)
	}
	if !target.CanWrite(user) {
		return fmt.Errorf("permission denied")
	}
	target.Data = data
	return nil
}
