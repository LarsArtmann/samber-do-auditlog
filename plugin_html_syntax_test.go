package auditlog_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/samber/do/v2"
)

// extractExecutableJS pulls the content of <script> tags that are NOT
// type="application/json" from an HTML string. This is the actual JavaScript
// that the browser will execute.
func extractExecutableJS(t *testing.T, html string) string {
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

		tag := html[start : strings.Index(html[start:end], ">")+start+1]

		// Skip JSON data scripts.
		if strings.Contains(tag, `type="application/json"`) {
			idx = end + len("</script>")
			continue
		}

		contentStart := strings.Index(html[start:end], ">") + start + 1
		sb.WriteString(html[contentStart:end])
		sb.WriteByte('\n')

		idx = end + len("</script>")
	}

	return sb.String()
}

// isRegexContext returns true if a / at this position would start a regex
// literal rather than a division operator. The decision is based on the last
// significant character: after operators, openers, and statement boundaries,
// / introduces a regex.
func isRegexContext(last byte) bool {
	switch last {
	case 0, '(', '[', '{', ',', ';', '=', '!', '&', '|', '?', ':',
		'+', '*', '%', '~', '^', '<', '>', '\n', '\t', ' ':
		return true
	default:
		return false
	}
}

// stripJSNoise removes string literals, comments, and regex literals from
// JavaScript so that delimiter balancing only counts structural
// braces/parens/brackets.
func stripJSNoise(js string) string {
	var sb strings.Builder

	i := 0
	n := len(js)
	lastChar := byte(0) // last non-whitespace byte written to output

	for i < n {
		// Line comment.
		if i+1 < n && js[i] == '/' && js[i+1] == '/' {
			for i < n && js[i] != '\n' {
				i++
			}

			continue
		}

		// Block comment.
		if i+1 < n && js[i] == '/' && js[i+1] == '*' {
			i += 2
			for i+1 < n && !(js[i] == '*' && js[i+1] == '/') {
				i++
			}

			i += 2
			continue
		}

		// Regex literal — / is a regex when preceded by an operator or opener.
		if js[i] == '/' && isRegexContext(lastChar) {
			i++ // skip opening /

			for i < n && js[i] != '/' && js[i] != '\n' {
				if js[i] == '\\' && i+1 < n {
					i += 2
					continue
				}

				// Character class [a-z] — skip to closing ].
				if js[i] == '[' {
					i++
					for i < n && js[i] != ']' {
						if js[i] == '\\' && i+1 < n {
							i++
						}
						i++
					}
				}

				i++
			}

			i++ // skip closing /
			// Skip regex flags.
			for i < n && isASCIILetter(js[i]) {
				i++
			}

			sb.WriteByte(' ')
			lastChar = ' '
			continue
		}

		// Single-quoted string.
		if js[i] == '\'' {
			i++
			for i < n && js[i] != '\'' {
				if js[i] == '\\' && i+1 < n {
					i++
				}
				i++
			}

			i++
			sb.WriteByte(' ')
			lastChar = ' '
			continue
		}

		// Double-quoted string.
		if js[i] == '"' {
			i++
			for i < n && js[i] != '"' {
				if js[i] == '\\' && i+1 < n {
					i++
				}
				i++
			}

			i++
			sb.WriteByte(' ')
			lastChar = ' '
			continue
		}

		// Template literal.
		if js[i] == '`' {
			i++
			for i < n && js[i] != '`' {
				if js[i] == '\\' && i+1 < n {
					i++
				}
				i++
			}

			i++
			sb.WriteByte(' ')
			lastChar = ' '
			continue
		}

		sb.WriteByte(js[i])
		if js[i] != ' ' && js[i] != '\t' && js[i] != '\n' && js[i] != '\r' {
			lastChar = js[i]
		}

		i++
	}

	return sb.String()
}

func isASCIILetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// assertJSBalanced checks that braces, parens, and brackets are balanced
// in the given JavaScript (after stripping strings, comments, and regexes).
func assertJSBalanced(t *testing.T, js string) {
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

// TestHTMLJavaScriptSyntax validates that the JavaScript embedded in the HTML
// report has balanced delimiters (braces, parens, brackets) after stripping
// string literals, comments, and regex literals. This catches syntax errors
// like a stray '}' that the golden byte-for-byte test would miss (since it
// compares against a golden file that may also contain the error).
func TestHTMLJavaScriptSyntax(t *testing.T) {
	t.Parallel()

	html := writeHTMLToString(t)
	js := extractExecutableJS(t, html)

	if len(js) < 100 {
		t.Fatalf("extracted JS too short (%d bytes) — extraction may have failed", len(js))
	}

	assertJSBalanced(t, js)
}

// TestHTMLJavaScriptSyntax_MultiService runs the same check on a report with
// multiple scopes and services to ensure template expansion doesn't break JS.
func TestHTMLJavaScriptSyntax_MultiService(t *testing.T) {
	t.Parallel()

	p, injector := newPluginAndInjector()
	child := injector.Scope("child-scope")

	provideDB(injector, "db", "postgres://localhost")
	provideCache(injector, "cache")
	provideUserServiceWithDB(injector, "user-service", "db")
	provideDB(child, "child-db", "postgres://child")

	_ = do.MustInvokeNamed[*Database](injector, "db")
	_ = do.MustInvokeNamed[*Cache](injector, "cache")
	_ = do.MustInvokeNamed[*UserService](injector, "user-service")
	_ = do.MustInvokeNamed[*Database](child, "child-db")

	var buf bytes.Buffer

	if err := p.WriteHTML(&buf); err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	js := extractExecutableJS(t, buf.String())

	if len(js) < 100 {
		t.Fatalf("extracted JS too short (%d bytes)", len(js))
	}

	assertJSBalanced(t, js)
}
