package file

import (
	"reflect"
	"testing"
)

func TestResolve(t *testing.T) {
	for _, tc := range []struct {
		name string
		cwd  []string
		expr string
		want []string
	}{
		{"empty expr from root", []string{}, "", []string{}},
		{"empty expr from cwd", []string{"home", "joe"}, "", []string{"home", "joe"}},
		{"single relative", []string{"home"}, "joe", []string{"home", "joe"}},
		{"absolute discards cwd", []string{"home", "joe"}, "/etc", []string{"etc"}},
		{"absolute with subdir", []string{"a"}, "/etc/hosts", []string{"etc", "hosts"}},
		{"absolute root", []string{"home"}, "/", []string{}},
		{"relative dotdot", []string{"home", "joe"}, "..", []string{"home"}},
		{"relative dot", []string{"home"}, ".", []string{"home"}},
		{"relative mixed", []string{"home", "joe"}, "../etc", []string{"home", "etc"}},
		{"absolute with dotdot — should be handled", []string{}, "/home/joe/../etc", []string{"home", "etc"}},
		{"absolute with dot — should be handled", []string{}, "/home/./joe", []string{"home", "joe"}},
		{"dotdot past root clamps", []string{}, "..", []string{}},
		{"absolute dotdot past root clamps", []string{"x"}, "/..", []string{}},
		{"trailing slash", []string{"home"}, "joe/", []string{"home", "joe"}},
		{"double slash", []string{}, "//etc//hosts", []string{"etc", "hosts"}},
		{"leading dot-slash", []string{"home"}, "./joe", []string{"home", "joe"}},
		{"chained dotdot", []string{"a", "b", "c"}, "../../d", []string{"a", "d"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := Resolve(tc.cwd, tc.expr)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Resolve(%v, %q) = %v, want %v", tc.cwd, tc.expr, got, tc.want)
			}
		})
	}
}

func TestSplit(t *testing.T) {
	for _, tc := range []struct {
		name     string
		cwd      []string
		expr     string
		wantDir  []string
		wantLast string
	}{
		{"empty expr → cwd, empty", []string{"home"}, "", []string{"home"}, ""},
		{"no slash → cwd, expr", []string{"home"}, "joe", []string{"home"}, "joe"},
		{"single slash relative", []string{"home"}, "joe/", []string{"home", "joe"}, ""},
		{"two-component relative", []string{"home"}, "joe/foo", []string{"home", "joe"}, "foo"},
		{"absolute root, last only", []string{"x"}, "/foo", []string{}, "foo"},
		{"absolute, two components", []string{}, "/etc/hosts", []string{"etc"}, "hosts"},
		{"absolute, trailing slash", []string{}, "/etc/", []string{"etc"}, ""},
		{"relative dotdot dir", []string{"home", "joe"}, "../foo", []string{"home"}, "foo"},
		{"absolute dotdot in dir", []string{}, "/home/joe/../etc/x", []string{"home", "etc"}, "x"},
		{"just slash", []string{"x"}, "/", []string{}, ""},
		{"trailing slash on absolute", []string{}, "/", []string{}, ""},
		{"dot in dir", []string{"home"}, "./foo", []string{"home"}, "foo"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gotDir, gotLast := Split(tc.cwd, tc.expr)
			if !reflect.DeepEqual(gotDir, tc.wantDir) {
				t.Errorf("Split(%v, %q) dir = %v, want %v", tc.cwd, tc.expr, gotDir, tc.wantDir)
			}
			if gotLast != tc.wantLast {
				t.Errorf("Split(%v, %q) last = %q, want %q", tc.cwd, tc.expr, gotLast, tc.wantLast)
			}
		})
	}
}
