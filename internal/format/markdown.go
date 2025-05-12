package format

import (
	"fmt"
	"strings"
)

var (
	// is thread safe/goroutine safe
	markdownReplacer = strings.NewReplacer(
		"\\", "\\\\",
		"`", "\\`",
		"*", "\\*",
		"_", "\\_",
		"{", "\\{",
		"}", "\\}",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		".", "\\.",
		"!", "\\!",
	)
)

// Escape user input outside of inline code blocks
func MarkdownEscape(userInput string) string {
	return markdownReplacer.Replace(userInput)
}

func MarkdownCodeHighlight(language string, code string) string {
	hasLeadingNewline := strings.HasPrefix(code, "\n")
	hasTrailingNewline := strings.HasSuffix(code, "\n")

	if hasLeadingNewline && hasTrailingNewline {
		return MarkdownMultilineCodeBlock(fmt.Sprintf("%s%s", language, code))
	} else if hasLeadingNewline {
		return MarkdownMultilineCodeBlock(fmt.Sprintf("%s%s\n", language, code))
	} else if hasTrailingNewline {
		return MarkdownMultilineCodeBlock(fmt.Sprintf("%s\n%s", language, code))
	}
	return MarkdownMultilineCodeBlock(fmt.Sprintf("%s\n%s\n", language, code))
}

func MarkdownMultilineCodeBlock(text string) string {
	return wrap(text, "```")
}

// WrapInInlineCodeBlock puts the user input into a inline codeblock that is properly escaped.
func MarkdownInlineCodeBlock(text string) string {
	return wrap(text, "`")
}

func MarkdownFat(text string) string {
	return wrap(text, "**")
}

func wrap(text, wrap string) (result string) {
	if text == "" {
		return ""
	}

	numWraps := strings.Count(text, wrap) + 1
	result = text
	for idx := 0; idx < numWraps; idx++ {
		result = fmt.Sprintf("%s%s%s", wrap, result, wrap)
	}
	return result
}
