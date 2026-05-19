package tui

import (
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
)

type markdownRenderer struct {
	enabled bool
	style   string
	width   int
	term    *glamour.TermRenderer
}

func (r *markdownRenderer) Render(s string, width int) string {
	if !r.enabled {
		return wrapText(s, width)
	}
	width = max(10, width)
	if r.term == nil || r.width != width {
		term, err := newGlamourRenderer(width, r.style)
		if err != nil {
			return wrapText(s, width)
		}
		r.term = term
		r.width = width
	}
	out, err := r.term.Render(s)
	if err != nil {
		return wrapText(s, width)
	}
	return strings.TrimRight(out, "\n")
}

func (r *markdownRenderer) Resize(width int) {
	if r.width != width {
		r.term = nil
		r.width = 0
	}
}

func newGlamourRenderer(width int, style string) (*glamour.TermRenderer, error) {
	options := []glamour.TermRendererOption{glamour.WithWordWrap(width)}
	switch strings.ToLower(strings.TrimSpace(style)) {
	case "", "auto", "dark", "weazlchat":
		options = append(options, glamour.WithStyles(weazlMarkdownStyle()))
	default:
		options = append(options, glamour.WithStandardStyle(style))
	}
	return glamour.NewTermRenderer(options...)
}

func weazlMarkdownStyle() ansi.StyleConfig {
	return ansi.StyleConfig{
		Document: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: strPtr("#FAFAFA"),
			},
		},
		Heading: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: strPtr("#F25D94"),
				Bold:  boolPtr(true),
			},
		},
		H1: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: strPtr("#F25D94"),
				Bold:  boolPtr(true),
			},
		},
		H2: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: strPtr("#F7D774"),
				Bold:  boolPtr(true),
			},
		},
		H3: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: strPtr("#04B575"),
				Bold:  boolPtr(true),
			},
		},
		Strong: ansi.StylePrimitive{
			Color: strPtr("#F7D774"),
			Bold:  boolPtr(true),
		},
		Emph: ansi.StylePrimitive{
			Color:  strPtr("#04B575"),
			Italic: boolPtr(true),
		},
		Link: ansi.StylePrimitive{
			Color:     strPtr("#7D56F4"),
			Underline: boolPtr(true),
		},
		LinkText: ansi.StylePrimitive{
			Color: strPtr("#F25D94"),
			Bold:  boolPtr(true),
		},
		BlockQuote: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: strPtr("#8E8E93"),
			},
			IndentToken: strPtr("│ "),
		},
		List: ansi.StyleList{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{
					Color: strPtr("#FAFAFA"),
				},
			},
			LevelIndent: 2,
		},
		Enumeration: ansi.StylePrimitive{
			Color: strPtr("#7D56F4"),
			Bold:  boolPtr(true),
		},
		Item: ansi.StylePrimitive{
			Color: strPtr("#F7D774"),
		},
		Code: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:           strPtr("#F7D774"),
				BackgroundColor: strPtr("#181820"),
			},
		},
		CodeBlock: ansi.StyleCodeBlock{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{
					Color:           strPtr("#FAFAFA"),
					BackgroundColor: strPtr("#181820"),
				},
			},
			Chroma: &ansi.Chroma{
				Text:        ansi.StylePrimitive{Color: strPtr("#FAFAFA")},
				Keyword:     ansi.StylePrimitive{Color: strPtr("#F25D94"), Bold: boolPtr(true)},
				Name:        ansi.StylePrimitive{Color: strPtr("#04B575")},
				Literal:     ansi.StylePrimitive{Color: strPtr("#F7D774")},
				Comment:     ansi.StylePrimitive{Color: strPtr("#8E8E93"), Italic: boolPtr(true)},
				Operator:    ansi.StylePrimitive{Color: strPtr("#7D56F4")},
				Punctuation: ansi.StylePrimitive{Color: strPtr("#8E8E93")},
				Background:  ansi.StylePrimitive{BackgroundColor: strPtr("#181820")},
			},
		},
		Table: ansi.StyleTable{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{
					Color: strPtr("#FAFAFA"),
				},
			},
			CenterSeparator: strPtr("│"),
			ColumnSeparator: strPtr("│"),
			RowSeparator:    strPtr("─"),
		},
	}
}

func strPtr(s string) *string {
	return &s
}

func boolPtr(v bool) *bool {
	return &v
}
