package command

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		want Pipeline
	}{
		{
			name: "empty",
			in:   "",
			want: Pipeline{},
		},
		{
			name: "whitespace only",
			in:   "   ",
			want: Pipeline{},
		},
		{
			name: "single command, no args",
			in:   "ls",
			want: Pipeline{Segments: []Segment{{Name: "ls"}}},
		},
		{
			name: "partial command name",
			in:   "l",
			want: Pipeline{Segments: []Segment{{Name: "l"}}},
		},
		{
			name: "command with trailing space (in args mode, no args)",
			in:   "ls ",
			want: Pipeline{Segments: []Segment{{Name: "ls", HasArgs: true, PartialArgs: ""}}},
		},
		{
			name: "command with one arg",
			in:   "cat readme.txt",
			want: Pipeline{Segments: []Segment{{
				Name:        "cat",
				Args:        []string{"readme.txt"},
				PartialArgs: "readme.txt",
				HasArgs:     true,
			}}},
		},
		{
			name: "command with two args",
			in:   "mv a b",
			want: Pipeline{Segments: []Segment{{
				Name:        "mv",
				Args:        []string{"a", "b"},
				PartialArgs: "a b",
				HasArgs:     true,
			}}},
		},
		{
			name: "partial arg",
			in:   "cat re",
			want: Pipeline{Segments: []Segment{{
				Name:        "cat",
				Args:        []string{"re"},
				PartialArgs: "re",
				HasArgs:     true,
			}}},
		},
		{
			name: "two-segment pipeline",
			in:   "ls | grep target",
			want: Pipeline{Segments: []Segment{
				{Name: "ls"},
				{Name: "grep", Args: []string{"target"}, PartialArgs: "target", HasArgs: true},
			}},
		},
		{
			name: "trailing pipe (empty fresh segment)",
			in:   "ls |",
			want: Pipeline{Segments: []Segment{
				{Name: "ls"},
				{},
			}},
		},
		{
			name: "trailing pipe with space",
			in:   "ls | ",
			want: Pipeline{Segments: []Segment{
				{Name: "ls"},
				{},
			}},
		},
		{
			name: "three-segment pipeline",
			in:   "cat a | grep b | ls",
			want: Pipeline{Segments: []Segment{
				{Name: "cat", Args: []string{"a"}, PartialArgs: "a", HasArgs: true},
				{Name: "grep", Args: []string{"b"}, PartialArgs: "b", HasArgs: true},
				{Name: "ls"},
			}},
		},
		{
			name: "pipe directly after command (no space)",
			in:   "ls|grep target",
			want: Pipeline{Segments: []Segment{
				{Name: "ls"},
				{Name: "grep", Args: []string{"target"}, PartialArgs: "target", HasArgs: true},
			}},
		},
		{
			name: "partial command in second segment",
			in:   "ls | gr",
			want: Pipeline{Segments: []Segment{
				{Name: "ls"},
				{Name: "gr"},
			}},
		},
		{
			name: "second segment in args mode, empty args",
			in:   "ls | grep ",
			want: Pipeline{Segments: []Segment{
				{Name: "ls"},
				{Name: "grep", HasArgs: true},
			}},
		},
		{
			name: "leading whitespace on first segment is trimmed",
			in:   "  ls",
			want: Pipeline{Segments: []Segment{{Name: "ls"}}},
		},
		{
			name: "double space between args is preserved in PartialArgs",
			in:   "ls  /tmp",
			want: Pipeline{Segments: []Segment{{
				Name:        "ls",
				Args:        []string{"/tmp"},
				PartialArgs: " /tmp",
				HasArgs:     true,
			}}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := Parse(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Parse(%q):\n got:  %#v\n want: %#v", tc.in, got, tc.want)
			}
		})
	}
}

func TestPipelineHelpers(t *testing.T) {
	t.Run("empty has no segments", func(t *testing.T) {
		p := Parse("")
		if p.IsPipeline() {
			t.Errorf("empty pipeline should not be IsPipeline()")
		}
		if p.InPipe() {
			t.Errorf("empty pipeline should not be InPipe()")
		}
		if last := p.Last(); last.Name != "" || last.HasArgs || last.PartialArgs != "" || len(last.Args) != 0 {
			t.Errorf("Last() of empty should be zero Segment, got %#v", last)
		}
	})

	t.Run("single segment", func(t *testing.T) {
		p := Parse("ls /tmp")
		if p.IsPipeline() {
			t.Errorf("single-segment line is not a pipeline")
		}
		if p.InPipe() {
			t.Errorf("single-segment line: last segment is not in pipe-receiver position")
		}
		if p.Last().Name != "ls" {
			t.Errorf("Last().Name = %q, want %q", p.Last().Name, "ls")
		}
	})

	t.Run("multi-segment", func(t *testing.T) {
		p := Parse("ls | grep target")
		if !p.IsPipeline() {
			t.Errorf("two-segment line should be a pipeline")
		}
		if !p.InPipe() {
			t.Errorf("two-segment line: last segment is in pipe-receiver position")
		}
		if p.Last().Name != "grep" {
			t.Errorf("Last().Name = %q, want %q", p.Last().Name, "grep")
		}
	})
}
