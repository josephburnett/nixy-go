package filesystem

import "fmt"

var _ File = &Text{}

type Text struct {
	Data string
	Own  string
	Perm Permission
}

func (t *Text) Read() (string, error) {
	return t.Data, nil
}

func (t *Text) Write(s string) error {
	t.Data = s
	return nil
}

func (t *Text) Execute(_ string) (File, error) {
	return nil, fmt.Errorf("cannot execute a text file")
}

func (t *Text) Owner() string {
	return t.Own
}

func (t *Text) Permission() Permission {
	return t.Perm
}
