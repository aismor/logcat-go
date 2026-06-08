package ui

import (
	"strconv"
	"strings"

	"github.com/aismor/logcat-go/internal/model"
)

const (
	displayWrapWidth    = 100
	maxDisplayBlocks    = 350
	maxDisplayTextRunes = 128 * 1024
)

func normalizeLevel(level string) string {
	if level == "" {
		return "I"
	}
	return level
}

func messageText(entry model.LogEntry) string {
	msg := strings.TrimSpace(entry.Message)
	if msg == "" {
		msg = strings.TrimSpace(entry.Raw)
	}
	return msg
}

func linePrefix(entry model.LogEntry) string {
	var b strings.Builder

	if entry.Date != "" && entry.Time != "" {
		b.WriteString(entry.Date)
		b.WriteByte(' ')
		b.WriteString(entry.Time)
	}

	if entry.PID > 0 {
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(strconv.Itoa(entry.PID))
		b.WriteByte('-')
		b.WriteString(strconv.Itoa(entry.TID))
	}

	tag := strings.TrimSpace(entry.Tag)
	if tag != "" {
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(tag)
	}

	lvl := normalizeLevel(entry.Level)
	if b.Len() > 0 {
		b.WriteByte(' ')
	}
	b.WriteString(lvl)

	return b.String()
}

func wrapRunes(text string, width int) []string {
	if width <= 0 || text == "" {
		return []string{text}
	}
	runes := []rune(text)
	if len(runes) <= width {
		return []string{text}
	}
	chunks := make([]string, 0, len(runes)/width+1)
	for i := 0; i < len(runes); i += width {
		end := i + width
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[i:end]))
	}
	return chunks
}

// formatDisplayBlock quebra mensagens longas (JSON/API) em várias linhas visíveis.
func formatDisplayBlock(entry model.LogEntry) string {
	prefix := strings.TrimSpace(linePrefix(entry))
	msg := messageText(entry)

	if msg == "" {
		return prefix
	}

	head := prefix + " "
	indent := strings.Repeat(" ", len([]rune(head)))
	chunks := wrapRunes(msg, displayWrapWidth)

	var b strings.Builder
	b.Grow(len(head) + len(msg) + len(chunks))
	b.WriteString(head)
	b.WriteString(chunks[0])
	for i := 1; i < len(chunks); i++ {
		b.WriteByte('\n')
		b.WriteString(indent)
		b.WriteString(chunks[i])
	}
	return b.String()
}

func formatPlainLine(entry model.LogEntry) string {
	prefix := strings.TrimSpace(linePrefix(entry))
	msg := messageText(entry)
	if msg == "" {
		return prefix
	}
	return prefix + " " + msg
}

func buildPlainText(entries []model.LogEntry) string {
	if len(entries) == 0 {
		return ""
	}

	var b strings.Builder
	for i, entry := range entries {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(formatDisplayBlock(entry))
	}
	return b.String()
}

func buildDisplayText(entries []model.LogEntry) string {
	return trimDisplayText(buildPlainText(entries))
}

func buildDisplayTextFromBlocks(blocks []string) string {
	if len(blocks) == 0 {
		return ""
	}

	start := 0
	if len(blocks) > maxDisplayBlocks {
		start = len(blocks) - maxDisplayBlocks
	}

	var b strings.Builder
	for i := start; i < len(blocks); i++ {
		if i > start {
			b.WriteByte('\n')
		}
		b.WriteString(blocks[i])
	}
	return trimDisplayText(b.String())
}

func trimDisplayText(text string) string {
	runes := []rune(text)
	if len(runes) <= maxDisplayTextRunes {
		return text
	}
	tail := string(runes[len(runes)-maxDisplayTextRunes:])
	if cut := strings.Index(tail, "\n"); cut >= 0 && cut < 256 {
		tail = tail[cut+1:]
	}
	return "…\n" + tail
}
