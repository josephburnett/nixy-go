package filesystem

type File interface {
	Read() (string, error)
	Write(string) error
	Execute(string) (File, error)
	Owner() string
	Permission() Permission
}

type Permission int

const (
	None Permission = iota
	Read
	Write
	Execute
)
