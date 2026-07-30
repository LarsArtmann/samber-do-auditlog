// Package testhelpers provides shared utilities for auditing HTML-embedded
// JavaScript in tests. It is intentionally internal to avoid expanding the public
// API surface.
package testhelpers

import (
	"strings"
	"testing"
)

// ExtractExecutableJS pulls the content of <script> tags that are NOT
// type="application/json" from an HTML string. This is the actual JavaScript
// that the browser will execute.
func ExtractExecutableJS(t *testing.T, html string) string {
	t.Helper()

	var out strings.Builder

	idx := 0

	for {
		start := strings.Index(html[idx:], "<script")
		if start == -1 {
			break
		}

		start += idx

		end := strings.Index(html[start:], "</script>")
		if end == -1 {
			break
		}

		end += start

		tagClose := strings.Index(html[start:end], ">")
		tag := html[start : tagClose+start+1]

		if strings.Contains(tag, `type="application/json"`) {
			idx = end + len("</script>")

			continue
		}

		contentStart := tagClose + start + 1
		out.WriteString(html[contentStart:end])
		out.WriteByte('\n')

		idx = end + len("</script>")
	}

	return out.String()
}

// jsStripper is a stateful scanner that removes string literals, comments,
// and regex literals from JavaScript source, leaving only structural delimiters.
type jsStripper struct {
	src    string
	pos    int
	out    strings.Builder
	lastCh byte
}

// stripJSNoise removes string literals, comments, and regex literals so that
// delimiter balancing only counts structural braces/parens/brackets.
func stripJSNoise(js string) string {
	scanner := &jsStripper{
		src:    js,
		pos:    0,
		out:    strings.Builder{},
		lastCh: 0,
	}

	for scanner.pos < len(scanner.src) {
		if scanner.skipLineComment() {
			continue
		}

		if scanner.skipBlockComment() {
			continue
		}

		if scanner.skipRegex() {
			continue
		}

		if scanner.skipQuoted('\'') {
			continue
		}

		if scanner.skipQuoted('"') {
			continue
		}

		if scanner.skipQuoted('`') {
			continue
		}

		scanner.copyByte()
	}

	return scanner.out.String()
}

func (scanner *jsStripper) remaining() bool { return scanner.pos < len(scanner.src) }

func (scanner *jsStripper) at(p int) byte {
	if p < len(scanner.src) {
		return scanner.src[p]
	}

	return 0
}

func (scanner *jsStripper) skipLineComment() bool {
	if scanner.at(scanner.pos) != '/' || scanner.at(scanner.pos+1) != '/' {
		return false
	}

	for scanner.remaining() && scanner.src[scanner.pos] != '\n' {
		scanner.pos++
	}

	return true
}

func (scanner *jsStripper) skipBlockComment() bool {
	if scanner.at(scanner.pos) != '/' || scanner.at(scanner.pos+1) != '*' {
		return false
	}

	scanner.pos += 2

	for scanner.pos+1 < len(scanner.src) && (scanner.src[scanner.pos] != '*' || scanner.src[scanner.pos+1] != '/') {
		scanner.pos++
	}

	scanner.pos += 2

	return true
}

func isRegexContext(last byte) bool {
	switch last {
	case 0, '(', '[', '{', ',', ';', '=', '!', '&', '|', '?', ':',
		'+', '*', '%', '~', '^', '<', '>', '\n', '\t', ' ':
		return true
	default:
		return false
	}
}

func (scanner *jsStripper) skipRegex() bool {
	if scanner.at(scanner.pos) != '/' || !isRegexContext(scanner.lastCh) {
		return false
	}

	scanner.pos++ // opening /

	for scanner.remaining() && scanner.src[scanner.pos] != '/' && scanner.src[scanner.pos] != '\n' {
		if scanner.src[scanner.pos] == '\\' {
			scanner.pos += 2

			continue
		}

		if scanner.src[scanner.pos] == '[' {
			scanner.skipCharClass()
		}

		scanner.pos++
	}

	scanner.pos++ // closing /

	// Skip regex flags.
	for scanner.remaining() && isASCIILetter(scanner.src[scanner.pos]) {
		scanner.pos++
	}

	scanner.out.WriteByte(' ')
	scanner.lastCh = ' '

	return true
}

func (scanner *jsStripper) skipCharClass() {
	scanner.pos++ // [

	for scanner.remaining() && scanner.src[scanner.pos] != ']' {
		if scanner.src[scanner.pos] == '\\' {
			scanner.pos++
		}

		scanner.pos++
	}
}

func (scanner *jsStripper) skipQuoted(quote byte) bool {
	if scanner.at(scanner.pos) != quote {
		return false
	}

	scanner.pos++

	for scanner.remaining() && scanner.src[scanner.pos] != quote {
		if scanner.src[scanner.pos] == '\\' {
			scanner.pos++
		}

		scanner.pos++
	}

	scanner.pos++

	scanner.out.WriteByte(' ')
	scanner.lastCh = ' '

	return true
}

func (scanner *jsStripper) copyByte() {
	byteVal := scanner.src[scanner.pos]
	scanner.out.WriteByte(byteVal)

	if byteVal != ' ' && byteVal != '\t' && byteVal != '\n' && byteVal != '\r' {
		scanner.lastCh = byteVal
	}

	scanner.pos++
}

func isASCIILetter(byteVal byte) bool {
	return (byteVal >= 'a' && byteVal <= 'z') || (byteVal >= 'A' && byteVal <= 'Z')
}

// AssertJSBalanced checks that braces, parens, and brackets are balanced
// in the given JavaScript (after stripping strings, comments, and regexes).
func AssertJSBalanced(t *testing.T, js string) {
	t.Helper()

	cleaned := stripJSNoise(js)

	pairs := map[rune]rune{'}': '{', ')': '(', ']': '['}

	var stack []rune

	for _, char := range cleaned {
		switch char {
		case '{', '(', '[':
			stack = append(stack, char)
		case '}', ')', ']':
			if len(stack) == 0 {
				t.Errorf("unbalanced %q in JS — closing with empty stack (stray delimiter)", char)

				return
			}

			top := stack[len(stack)-1]
			if top != pairs[char] {
				t.Errorf("unbalanced %q in JS — expected closing for %q, got %q", char, top, char)

				return
			}

			stack = stack[:len(stack)-1]
		}
	}

	if len(stack) > 0 {
		t.Errorf("unbalanced JS — %d unclosed delimiter(s): %q", len(stack), string(stack))
	}
}
