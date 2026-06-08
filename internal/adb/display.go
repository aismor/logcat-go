package adb

import (
	"strings"
	"unicode"

	"github.com/aismor/logcat-go/internal/model"
)

const maxLogLineRunes = 512 * 1024

type LogAssembler struct {
	last *model.LogEntry
}

func NewLogAssembler() *LogAssembler {
	return &LogAssembler{}
}

// Push emite uma linha por registro logcat (sem mesclar JSON/respostas longas).
func (a *LogAssembler) Push(line string) (model.LogEntry, bool, bool) {
	entry, ok := ParseLogLine(line)
	if !ok {
		return model.LogEntry{}, false, false
	}

	entry.Message = sanitizeLogText(entry.Message)
	entry.Raw = sanitizeLogText(entry.Raw)

	if isContinuation(entry) {
		entry = a.inheritContinuation(entry)
	} else {
		entry.Level = normalizeLevel(entry.Level)
	}

	a.last = &entry
	if !IsDisplayable(entry) {
		return model.LogEntry{}, false, false
	}
	return entry, true, false
}

func (a *LogAssembler) inheritContinuation(entry model.LogEntry) model.LogEntry {
	text := strings.TrimSpace(entry.Raw)
	if text == "" {
		text = strings.TrimSpace(entry.Message)
	}
	entry.Message = sanitizeLogText(text)
	entry.Level = "I"

	if a.last == nil {
		return entry
	}

	entry.Date = a.last.Date
	entry.Time = a.last.Time
	entry.PID = a.last.PID
	entry.TID = a.last.TID
	entry.UID = a.last.UID
	entry.Level = a.last.Level
	entry.Tag = a.last.Tag
	return entry
}

func isContinuation(entry model.LogEntry) bool {
	return entry.Date == "" && entry.Time == "" && entry.Level == "" && entry.Tag == ""
}

func normalizeLevel(level string) string {
	if level == "" {
		return "I"
	}
	return level
}

func sanitizeLogText(s string) string {
	if s == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '\n', '\r', '\t':
			b.WriteRune(' ')
		default:
			if unicode.IsControl(r) {
				continue
			}
			b.WriteRune(r)
		}
	}

	out := strings.TrimSpace(b.String())
	runes := []rune(out)
	if len(runes) > maxLogLineRunes {
		return string(runes[:maxLogLineRunes]) + "…(truncado)"
	}
	return out
}

func IsDisplayable(entry model.LogEntry) bool {
	if strings.TrimSpace(entry.Message) != "" {
		return true
	}
	if entry.Tag == "" && entry.Date == "" {
		return strings.TrimSpace(entry.Raw) != ""
	}
	return false
}
