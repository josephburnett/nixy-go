package command

import "strings"

// Pipeline is a (possibly partial) command line. A bare command parses as a
// single-segment Pipeline; `a | b` parses as two segments.
//
// Parse handles both finalized lines (for execution) and in-progress lines
// (for keystroke validation): a trailing `|` produces an empty trailing
// segment, and a trailing space produces a Segment with HasArgs=true but
// no Args yet. The validator distinguishes "mid-command-name" from "in
// args" by looking at HasArgs on the last segment.
type Pipeline struct {
	Segments []Segment
}

// Segment is one command in a pipeline.
//
// Args is the canonically-split argument list (whitespace-tokenized) — used
// by binaries at launch time. PartialArgs is the raw text after the first
// space, including any embedded spaces — used by keystroke validators that
// need to see what the user has typed so far. HasArgs reports whether the
// user has typed at least one space after Name; this is what separates
// "mid-command-name" (no space yet, can still extend the name) from "in
// args" (space typed, even with empty args).
type Segment struct {
	Name        string
	Args        []string
	PartialArgs string
	HasArgs     bool
	// Redirect is the segment's `> FILE` or `>> FILE`, if it has one.
	// Its args end at the operator, so PartialArgs never includes it.
	Redirect *Redirect
}

// Redirect sends a command's stdout to a file: `>` replaces the file's
// contents, `>>` appends. The shell opens the file, as the user,
// before the command runs.
type Redirect struct {
	Append bool
	// Raw is everything typed after the operator, spaces included:
	// "" right after `>`, " no" while typing `> notes.txt`.
	Raw string
}

// Target is the file named, as typed.
func (r Redirect) Target() string { return strings.TrimSpace(r.Raw) }

// Parse splits a (possibly partial) command line into a Pipeline.
//
// The empty (or all-whitespace) line returns an empty Pipeline. Otherwise
// the line is split on `|` and each segment is parsed independently.
// Segments are normalized for leading whitespace; trailing whitespace is
// preserved on the last segment to support partial-input parsing (the
// trailing space is what tells the validator the user is in args mode).
func Parse(line string) Pipeline {
	if strings.TrimSpace(line) == "" {
		return Pipeline{}
	}
	parts := strings.Split(line, "|")
	var p Pipeline
	for i, raw := range parts {
		p.Segments = append(p.Segments, parseSegment(raw, i == len(parts)-1))
	}
	return p
}

// parseSegment normalizes one `|`-separated piece. Leading whitespace is
// always discarded (it has no semantic meaning anywhere). Trailing
// whitespace is discarded for non-last segments — they are committed by
// the user typing `|`, so any trailing space is incidental — but preserved
// on the last segment, where a trailing space is the signal that the user
// has entered args mode (HasArgs=true with empty PartialArgs).
//
// A `>` ends the command: what follows is the redirect, and the command
// before it is committed like a non-last segment.
func parseSegment(raw string, isLast bool) Segment {
	if cmd, target, ok := strings.Cut(raw, ">"); ok {
		r := &Redirect{Raw: target}
		if rest, ok := strings.CutPrefix(target, ">"); ok {
			r.Append, r.Raw = true, rest
		}
		seg := parseSegment(cmd, false)
		seg.Redirect = r
		return seg
	}
	raw = strings.TrimLeft(raw, " ")
	if !isLast {
		raw = strings.TrimRight(raw, " ")
	}
	if raw == "" {
		return Segment{}
	}
	name, partial, ok := strings.Cut(raw, " ")
	if !ok {
		return Segment{Name: raw}
	}
	seg := Segment{
		Name:        name,
		PartialArgs: partial,
		HasArgs:     true,
	}
	if fields := strings.Fields(partial); len(fields) > 0 {
		seg.Args = fields
	}
	return seg
}

// Text is the command as its validator sees it: the name, and the args
// typed so far after a space.
func (s Segment) Text() string {
	if s.HasArgs {
		return s.Name + " " + s.PartialArgs
	}
	return s.Name
}

// Argv is the segment as a program sees it: its name, then its args.
func (s Segment) Argv() []string {
	return append([]string{s.Name}, s.Args...)
}

// IsPipeline reports whether the line contains at least one `|` (i.e. has
// more than one segment). The empty Pipeline returns false.
func (p Pipeline) IsPipeline() bool {
	return len(p.Segments) > 1
}

// Last returns the final segment — the one the cursor is in for partial
// input. Returns the zero Segment for an empty Pipeline.
func (p Pipeline) Last() Segment {
	if len(p.Segments) == 0 {
		return Segment{}
	}
	return p.Segments[len(p.Segments)-1]
}

// InPipe reports whether the last segment is in pipe-receiver position —
// i.e. there is at least one earlier segment whose stdout will feed it.
func (p Pipeline) InPipe() bool {
	return len(p.Segments) > 1
}
