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

// Errno is an error from a file operation. Its text is the C library's
// strerror text, so commands can print it the way coreutils does
// ("cat: foo: No such file or directory").
type Errno string

func (e Errno) Error() string { return string(e) }

const (
	ErrNotExist   Errno = "No such file or directory"
	ErrNotDir     Errno = "Not a directory"
	ErrIsDir      Errno = "Is a directory"
	ErrPermission Errno = "Permission denied"
	ErrExist      Errno = "File exists"
)

// Navigate walks the filesystem tree following the given path components,
// ignoring permissions. It is the world's view (quests, world setup);
// commands acting for a user go through Lookup.
func (f *F) Navigate(path []string) (*F, error) {
	return f.walk(path, "")
}

// Lookup walks path the way the kernel does for user: every folder it
// passes through must grant the user Read, which in this model is both
// listing and search (real Unix's r and x bits on a directory). The
// final component itself is not checked; whether the user may read or
// write it is up to the caller.
func (f *F) Lookup(path []string, user string) (*F, error) {
	return f.walk(path, user)
}

// walk checks search permission when user is set.
func (f *F) walk(path []string, user string) (*F, error) {
	current := f
	for _, p := range path {
		if current.Type != Folder {
			return nil, ErrNotDir
		}
		if user != "" && !current.CanRead(user) {
			return nil, ErrPermission
		}
		child, ok := current.Files[p]
		if !ok {
			return nil, ErrNotExist
		}
		current = child
	}
	return current, nil
}

// LookupDir is Lookup for a folder whose contents the user wants to
// list or change: the folder itself must be readable too.
func (f *F) LookupDir(path []string, user string) (*F, error) {
	dir, err := f.Lookup(path, user)
	if err != nil {
		return nil, err
	}
	if dir.Type != Folder {
		return nil, ErrNotDir
	}
	if !dir.CanRead(user) {
		return nil, ErrPermission
	}
	return dir, nil
}

// Home is a user's home folder, as /etc/passwd would list it: /root for
// root, /home/<user> for everyone else.
func Home(user string) []string {
	if user == OwnerRoot {
		return []string{"root"}
	}
	return []string{"home", user}
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

// CreateFile creates a new file at path/name. The user must be able to
// reach the parent folder at path and write to it.
func (f *F) CreateFile(path []string, name string, newFile *F, user string) error {
	parent, err := f.LookupDir(path, user)
	if err != nil {
		return err
	}
	if !parent.CanWrite(user) {
		return ErrPermission
	}
	if _, exists := parent.Files[name]; exists {
		return ErrExist
	}
	if parent.Files == nil {
		parent.Files = map[string]*F{}
	}
	parent.Files[name] = newFile
	return nil
}

// DeleteFile removes the file at path/name. The user must be able to
// reach the parent folder and write to it.
func (f *F) DeleteFile(path []string, name string, user string) error {
	parent, err := f.LookupDir(path, user)
	if err != nil {
		return err
	}
	if _, exists := parent.Files[name]; !exists {
		return ErrNotExist
	}
	if !parent.CanWrite(user) {
		return ErrPermission
	}
	delete(parent.Files, name)
	return nil
}

// SetData updates the data content of the file at path/name. The user must
// be able to reach it and have write permission on the file itself.
func (f *F) SetData(path []string, name string, data string, user string) error {
	target, err := f.Lookup(append(append([]string{}, path...), name), user)
	if err != nil {
		return err
	}
	if !target.CanWrite(user) {
		return ErrPermission
	}
	target.Data = data
	return nil
}

// ReadFile returns the contents of the file at path, as user: the path
// must be reachable, a file rather than a folder, and readable.
func (f *F) ReadFile(path []string, user string) (string, error) {
	target, err := f.Lookup(path, user)
	if err != nil {
		return "", err
	}
	if target.Type == Folder {
		return "", ErrIsDir
	}
	if !target.CanRead(user) {
		return "", ErrPermission
	}
	return target.Data, nil
}
