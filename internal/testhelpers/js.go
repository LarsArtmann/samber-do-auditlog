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

	var sb strings.Builder

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
		sb.WriteString(html[contentStart:end])
		sb.WriteByte('\n')

		idx = end + len("</script>")
	}

	return sb.String()
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
	s := &jsStripper{src: js}

	for s.pos < len(s.src) {
		if s.skipLineComment() {
			continue
		}

		if s.skipBlockComment() {
			continue
		}

		if s.skipRegex() {
			continue
		}

		if s.skipQuoted('\'') {
			continue
		}

		if s.skipQuoted('"') {
			continue
		}

		if s.skipQuoted('`') {
			continue
		}

		s.copyByte()
	}

	return s.out.String()
}

func (s *jsStripper) remaining() bool { return s.pos < len(s.src) }

func (s *jsStripper) at(p int) byte {
	if p < len(s.src) {
		return s.src[p]
	}

	return 0
}

func (s *jsStripper) skipLineComment() bool {
	if s.at(s.pos) != '/' || s.at(s.pos+1) != '/' {
		return false
	}

	for s.remaining() && s.src[s.pos] != '\n' {
		s.pos++
	}

	return true
}

func (s *jsStripper) skipBlockComment() bool {
	if s.at(s.pos) != '/' || s.at(s.pos+1) != '*' {
		return false
	}

	s.pos += 2

	for s.pos+1 < len(s.src) && !(s.src[s.pos] == '*' && s.src[s.pos+1] == '/') {
		s.pos++
	}

	s.pos += 2

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

func (s *jsStripper) skipRegex() bool {
	if s.at(s.pos) != '/' || !isRegexContext(s.lastCh) {
		return false
	}

	s.pos++ // opening /

	for s.remaining() && s.src[s.pos] != '/' && s.src[s.pos] != '\n' {
		if s.src[s.pos] == '\\' {
			s.pos += 2

			continue
		}

		if s.src[s.pos] == '[' {
			s.skipCharClass()
		}

		s.pos++
	}

	s.pos++ // closing /

	// Skip regex flags.
	for s.remaining() && isASCIILetter(s.src[s.pos]) {
		s.pos++
	}

	s.out.WriteByte(' ')
	s.lastCh = ' '

	return true
}

func (s *jsStripper) skipCharClass() {
	s.pos++ // [

	for s.remaining() && s.src[s.pos] != ']' {
		if s.src[s.pos] == '\\' {
			s.pos++
		}

		s.pos++
	}
}

func (s *jsStripper) skipQuoted(quote byte) bool {
	if s.at(s.pos) != quote {
		return false
	}

	s.pos++

	for s.remaining() && s.src[s.pos] != quote {
		if s.src[s.pos] == '\\' {
			s.pos++
		}

		s.pos++
	}

	s.pos++

	s.out.WriteByte(' ')
	s.lastCh = ' '

	return true
}

func (s *jsStripper) copyByte() {
	b := s.src[s.pos]
	s.out.WriteByte(b)

	if b != ' ' && b != '\t' && b != '\n' && b != '\r' {
		s.lastCh = b
	}

	s.pos++
}

func isASCIILetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// AssertJSBalanced checks that braces, parens, and brackets are balanced
// in the given JavaScript (after stripping strings, comments, and regexes).
func AssertJSBalanced(t *testing.T, js string) {
	t.Helper()

	cleaned := stripJSNoise(js)

	pairs := map[rune]rune{'}': '{', ')': '(', ']': '['}

	var stack []rune

	for _, ch := range cleaned {
		switch ch {
		case '{', '(', '[':
			stack = append(stack, ch)
		case '}', ')', ']':
			if len(stack) == 0 {
				t.Errorf("unbalanced %q in JS — closing with empty stack (stray delimiter)", ch)

				return
			}

			top := stack[len(stack)-1]
			if top != pairs[ch] {
				t.Errorf("unbalanced %q in JS — expected closing for %q, got %q", ch, top, ch)

				return
			}

			stack = stack[:len(stack)-1]
		}
	}

	if len(stack) > 0 {
		t.Errorf("unbalanced JS — %d unclosed delimiter(s): %q", len(stack), string(stack))
	}
}
