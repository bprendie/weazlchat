package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/bprendie/weazlchat/internal/storage"
)

// wrapText wraps text to fit within the specified width, preserving paragraphs
func wrapText(s string, width int) string {
	width = max(10, width)
	var out strings.Builder
	paragraphs := strings.Split(s, "\n")
	for i, paragraph := range paragraphs {
		if paragraph == "" {
			if i < len(paragraphs)-1 {
				out.WriteByte('\n')
			}
			continue
		}
		out.WriteString(wrapLine(paragraph, width))
		if i < len(paragraphs)-1 {
			out.WriteByte('\n')
		}
	}
	return out.String()
}

// wrapLine wraps a single line of text to fit within the specified width
func wrapLine(s string, width int) string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return ""
	}
	var out strings.Builder
	lineWidth := 0
	for _, word := range words {
		wordWidth := lipgloss.Width(word)
		if lineWidth > 0 && lineWidth+1+wordWidth > width {
			out.WriteByte('\n')
			lineWidth = 0
		}
		if lineWidth > 0 {
			out.WriteByte(' ')
			lineWidth++
		}
		if wordWidth <= width {
			out.WriteString(word)
			lineWidth += wordWidth
			continue
		}
		for _, r := range word {
			rw := lipgloss.Width(string(r))
			if lineWidth > 0 && lineWidth+rw > width {
				out.WriteByte('\n')
				lineWidth = 0
			}
			out.WriteRune(r)
			lineWidth += rw
		}
	}
	return out.String()
}

// estimateMessages estimates total tokens for a slice of messages
func estimateMessages(messages []storage.Message) int {
	total := 0
	for _, msg := range messages {
		total += estimateTokens(msg.Content)
		total += estimateTokens(msg.ToolCalls)
	}
	return total
}

// estimateTokens provides a rough token count estimate for text
func estimateTokens(s string) int {
	words := len(strings.Fields(s))
	if words == 0 {
		return 0
	}
	return max(1, int(float64(words)*1.33))
}

// countLines counts the number of lines in a string
func countLines(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

// limitToolOutput truncates tool output to the specified maximum character count
func limitToolOutput(s string, maxChars int) string {
	if maxChars <= 0 {
		maxChars = 12000
	}
	if len(s) <= maxChars {
		return s
	}
	return s[:maxChars] + fmt.Sprintf("\n\n[truncated: %d chars omitted]", len(s)-maxChars)
}
