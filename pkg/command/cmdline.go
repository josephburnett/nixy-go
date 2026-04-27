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
}

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
func parseSegment(raw string, isLast bool) Segment {
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
