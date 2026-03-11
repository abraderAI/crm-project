package slug

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerate(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple", "Hello World", "hello-world"},
		{"already lowercase", "hello world", "hello-world"},
		{"with numbers", "Project 42", "project-42"},
		{"special chars", "Hello! @World#", "hello-world"},
		{"multiple spaces", "Hello   World", "hello-world"},
		{"leading trailing spaces", "  Hello World  ", "hello-world"},
		{"hyphens preserved", "my-slug-name", "my-slug-name"},
		{"underscores to hyphens", "my_slug_name", "my-slug-name"},
		{"mixed case", "CamelCaseSlug", "camelcaseslug"},
		{"unicode", "Café Résumé", "caf-r-sum"},
		{"empty", "", ""},
		{"only special", "!@#$%^&*()", ""},
		{"numbers only", "12345", "12345"},
		{"leading hyphens", "---hello", "hello"},
		{"trailing hyphens", "hello---", "hello"},
		{"consecutive hyphens", "hello---world", "hello-world"},
		{"dots", "hello.world", "hello-world"},
		{"slash", "path/to/thing", "path-to-thing"},
		{"ampersand", "sales & marketing", "sales-marketing"},
		{"plus", "a+b", "a-b"},
		{"parentheses", "foo (bar)", "foo-bar"},
		{"brackets", "foo [bar]", "foo-bar"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Generate(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGenerate_Deterministic(t *testing.T) {
	input := "My Test Slug"
	a := Generate(input)
	b := Generate(input)
	assert.Equal(t, a, b)
}

func TestGenerate_NoLeadingTrailingHyphens(t *testing.T) {
	inputs := []string{
		"---test---",
		"   test   ",
		"!!!test!!!",
		"-test-",
	}
	for _, input := range inputs {
		got := Generate(input)
		if got != "" {
			assert.NotEqual(t, '-', rune(got[0]), "no leading hyphen for input: %q", input)
			assert.NotEqual(t, '-', rune(got[len(got)-1]), "no trailing hyphen for input: %q", input)
		}
	}
}

func FuzzGenerate(f *testing.F) {
	// Seed corpus with diverse inputs.
	seeds := []string{
		"Hello World",
		"my-slug-name",
		"CamelCaseSlug",
		"Café Résumé",
		"",
		"!@#$%^&*()",
		"12345",
		"---hello---",
		"hello.world.test",
		"path/to/thing",
		"   spaces   everywhere   ",
		"UPPERCASE STRING",
		"a",
		"a-b-c-d-e-f",
		"tab\there",
		"newline\nhere",
		"日本語テスト",
		"Ñoño",
		"hello___world",
		"foo--bar--baz",
		"test!@#test",
		"multiple   spaces   here",
		"trailing-",
		"-leading",
		"mix3d-numb3rs-and-l3tt3rs",
		"THIS IS VERY LONG INPUT STRING THAT SHOULD STILL PRODUCE A VALID SLUG",
		"under_score_test",
		"dot.separated.values",
		"comma,separated,values",
		"semicolon;separated;values",
		"colon:separated:values",
		"pipe|separated|values",
		"backslash\\test",
		"quote\"test",
		"single'quote'test",
		"angle<bracket>test",
		"curly{brace}test",
		"tilde~test",
		"backtick`test",
		"at@sign",
		"hash#tag",
		"dollar$sign",
		"percent%sign",
		"caret^sign",
		"ampersand&sign",
		"asterisk*sign",
		"plus+sign",
		"equals=sign",
		"question?mark",
		"exclamation!mark",
		"  leading-spaces",
		"trailing-spaces  ",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, input string) {
		result := Generate(input)

		// Slug must be lowercase.
		assert.Equal(t, result, Generate(input), "should be deterministic")

		if result == "" {
			return
		}

		// No leading or trailing hyphens.
		assert.NotEqual(t, byte('-'), result[0], "no leading hyphen")
		assert.NotEqual(t, byte('-'), result[len(result)-1], "no trailing hyphen")

		// Only valid characters.
		for _, c := range result {
			valid := (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-'
			assert.True(t, valid, "invalid char %q in slug %q", c, result)
		}

		// No consecutive hyphens.
		for i := 1; i < len(result); i++ {
			if result[i] == '-' && result[i-1] == '-' {
				t.Errorf("consecutive hyphens in slug %q", result)
			}
		}
	})
}
