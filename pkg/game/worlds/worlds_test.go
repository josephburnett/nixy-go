package worlds

import (
	"strings"
	"testing"

	"github.com/josephburnett/nixy-go/pkg/file"
)

// Text files end in a newline, as real ones do: otherwise `wc -l`
// counts one line fewer than `cat` shows.
func TestTextFilesEndInNewline(t *testing.T) {
	for name, world := range map[string]func() *file.F{"laptop": Laptop, "nixy": Nixy, "server": Server} {
		walkText(world(), "/", func(path, data string) {
			if data != "" && !strings.HasSuffix(data, "\n") {
				t.Errorf("%s:%s doesn't end in a newline", name, path)
			}
		})
	}
}

func walkText(f *file.F, path string, fn func(path, data string)) {
	if f.Type == file.Text {
		fn(path, f.Data)
	}
	for n, c := range f.Files {
		walkText(c, strings.TrimSuffix(path, "/")+"/"+n, fn)
	}
}
