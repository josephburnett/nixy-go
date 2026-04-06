package terminal

import "unicode/utf8"

// WrapLine breaks a single line into multiple display lines at the given width.
// If width <= 0, it returns the line as-is.
func WrapLine(line string, width int) []string {
	if width <= 0 {
		return []string{line}
	}
	if utf8.RuneCountInString(line) <= width {
		return []string{line}
	}
	var result []string
	runes := []rune(line)
	for len(runes) > 0 {
		end := width
		if end > len(runes) {
			end = len(runes)
		}
		result = append(result, string(runes[:end]))
		runes = runes[end:]
	}
	return result
}

// ReflowLines wraps all original lines to the given width, then returns
// the last height display lines (for scrolling).
func ReflowLines(lines []string, width, height int) []string {
	var display []string
	for _, line := range lines {
		wrapped := WrapLine(line, width)
		display = append(display, wrapped...)
	}
	if len(display) > height {
		display = display[len(display)-height:]
	}
	return display
}
