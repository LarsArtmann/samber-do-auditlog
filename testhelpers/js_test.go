package testhelpers

import (
	"strings"
	"testing"
)

func TestExtractExecutableJS_SkipsJSONScripts(t *testing.T) {
	t.Parallel()

	html := `<html>
<head><script type="application/json" id="data">{"key": "}"}</script></head>
<body>
<script>console.log("hello");</script>
<script>var x = function() { return 42; };</script>
</body>
</html>`

	result := ExtractExecutableJS(t, html)

	if strings.Contains(result, `"key"`) {
		t.Error("should exclude JSON script blocks")
	}

	if !strings.Contains(result, `console.log`) {
		t.Error("should include executable scripts")
	}

	if !strings.Contains(result, `function()`) {
		t.Error("should include all executable script blocks")
	}
}

func TestExtractExecutableJS_NoScripts(t *testing.T) {
	t.Parallel()

	result := ExtractExecutableJS(t, `<html><body><p>none</p></body></html>`)

	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestAssertJSBalanced_ValidCode(t *testing.T) {
	t.Parallel()

	cases := []string{
		`function foo() { var arr = [1, 2, 3]; return arr.map(x => x * 2); }`,
		`var s = "function() { broken }"; var arr = [1, 2];`,
		`// comment with } bracket
/* block comment with ) paren */
var x = { a: 1 };`,
		`var re = /pattern/g; var obj = { a: 1 };`,
		"var tmpl = `hello`; var obj = { a: 1 };",
	}

	for i, js := range cases {
		AssertJSBalanced(t, js)

		if t.Failed() {
			t.Errorf("case %d incorrectly flagged as unbalanced: %s", i, js[:min(40, len(js))])
		}
	}
}

func TestStripJSNoise(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "line comment stripped",
			input: `var x = 1; // comment { with } brackets`,
			want:  `var x = 1; `,
		},
		{
			name:  "block comment stripped",
			input: `var x = /* comment { } */ 1;`,
			want:  `var x =  1;`,
		},
		{
			name:  "single-quoted string stripped",
			input: `var x = 'string { with } brackets';`,
			want:  `var x =  ;`,
		},
		{
			name:  "double-quoted string stripped",
			input: `var x = "string ( with ) parens";`,
			want:  `var x =  ;`,
		},
		{
			name:  "template literal stripped",
			input: "var x = `template`;",
			want:  "var x =  ;",
		},
		{
			name:  "regex stripped",
			input: `var re = /pattern/g; var x = 1;`,
			want:  `var re =  ; var x = 1;`,
		},
		{
			name:  "char class in regex",
			input: `var re = /[{}]/g; var x = 1;`,
			want:  `var re =  ; var x = 1;`,
		},
		{
			name:  "escaped quote in string",
			input: `var s = "it\'s ok"; var x = { a: 1 };`,
			want:  `var s =  ; var x = { a: 1 };`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := stripJSNoise(tt.input)
			if got != tt.want {
				t.Errorf("stripJSNoise(%q)\n  got:  %q\n  want: %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsASCIILetter(t *testing.T) {
	t.Parallel()

	for _, ch := range []byte{'a', 'z', 'A', 'Z'} {
		if !isASCIILetter(ch) {
			t.Errorf("isASCIILetter(%q) = false, want true", ch)
		}
	}

	for _, ch := range []byte{'0', '9', '_', '-', ' ', 0} {
		if isASCIILetter(ch) {
			t.Errorf("isASCIILetter(%q) = true, want false", ch)
		}
	}
}

func TestIsRegexContext(t *testing.T) {
	t.Parallel()

	regexChars := []byte{
		0, '(', '[', '{', ',', ';', '=', '!', '&', '|', '?', ':',
		'+', '*', '%', '~', '^', '<', '>', '\n', '\t', ' ',
	}

	for _, ch := range regexChars {
		if !isRegexContext(ch) {
			t.Errorf("isRegexContext(%q) = false, want true", ch)
		}
	}

	nonRegexChars := []byte{'a', 'x', '0', '_', ')', ']'}

	for _, ch := range nonRegexChars {
		if isRegexContext(ch) {
			t.Errorf("isRegexContext(%q) = true, want false", ch)
		}
	}
}
