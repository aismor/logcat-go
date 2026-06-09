package ui

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/aismor/logcat-go/internal/model"
)

func looksLikeJSON(text string) bool {
	_, ok := extractJSON(text)
	return ok
}

func extractJSON(text string) (string, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", false
	}

	if json.Valid([]byte(text)) {
		return text, true
	}

	if normalized, ok := normalizeJSONSelection(text); ok {
		return normalized, true
	}

	for i, r := range text {
		switch r {
		case '{':
			if candidate, ok := extractBalancedJSON(text[i:], '{', '}'); ok {
				return candidate, true
			}
		case '[':
			if candidate, ok := extractBalancedJSON(text[i:], '[', ']'); ok {
				return candidate, true
			}
		}
	}
	return "", false
}

func extractBalancedJSON(s string, open, close rune) (string, bool) {
	if s == "" {
		return "", false
	}

	depth := 0
	inString := false
	escape := false

	for i, r := range s {
		if escape {
			escape = false
			continue
		}
		if r == '\\' && inString {
			escape = true
			continue
		}
		if r == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if r == open {
			depth++
			continue
		}
		if r == close {
			depth--
			if depth == 0 {
				candidate := s[:i+1]
				if json.Valid([]byte(candidate)) {
					return candidate, true
				}
				if normalized, ok := normalizeJSONSelection(candidate); ok {
					return normalized, true
				}
				return "", false
			}
		}
	}
	return "", false
}

func normalizeJSONSelection(text string) (string, bool) {
	lines := strings.Split(text, "\n")
	parts := make([]string, 0, len(lines))
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if i == 0 {
			if idx := strings.IndexAny(line, "{["); idx > 0 {
				line = line[idx:]
			}
		}
		parts = append(parts, line)
	}
	if len(parts) == 0 {
		return "", false
	}

	for _, joined := range []string{strings.Join(parts, ""), strings.Join(parts, " ")} {
		if json.Valid([]byte(joined)) {
			return joined, true
		}
	}
	return "", false
}

func resolveJSONFromEntries(entries []model.LogEntry, index int) (string, bool) {
	if index < 0 || index >= len(entries) {
		return "", false
	}

	sources := []string{
		mergeContinuationMessages(entries, index),
		messageText(entries[index]),
		formatPlainLine(entries[index]),
		entries[index].Raw,
	}
	for _, text := range sources {
		if jsonText, ok := extractJSON(text); ok {
			return jsonText, true
		}
	}
	return "", false
}

func mergeContinuationMessages(entries []model.LogEntry, start int) string {
	if start < 0 || start >= len(entries) {
		return ""
	}

	var b strings.Builder
	b.WriteString(messageText(entries[start]))
	if _, ok := extractJSON(b.String()); ok {
		return b.String()
	}

	for i := start + 1; i < len(entries) && i-start <= 32; i++ {
		if !sameLogContext(entries[start], entries[i]) {
			break
		}
		b.WriteString(messageText(entries[i]))
		if _, ok := extractJSON(b.String()); ok {
			return b.String()
		}
	}
	return b.String()
}

func sameLogContext(a, b model.LogEntry) bool {
	return a.PID == b.PID &&
		a.TID == b.TID &&
		a.Tag == b.Tag &&
		a.Date == b.Date &&
		a.Time == b.Time
}

func formatJSON(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	var value any
	dec := json.NewDecoder(bytes.NewReader([]byte(raw)))
	dec.UseNumber()
	if err := dec.Decode(&value); err != nil {
		return "", err
	}

	out, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func findMatchOffsets(text, query string) []int {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}

	lowerText := strings.ToLower(text)
	lowerQuery := strings.ToLower(query)
	size := len(lowerQuery)
	if size == 0 {
		return nil
	}

	offsets := make([]int, 0, 8)
	for i := 0; i <= len(lowerText)-size; {
		pos := strings.Index(lowerText[i:], lowerQuery)
		if pos < 0 {
			break
		}
		pos += i
		offsets = append(offsets, pos)
		i = pos + size
	}
	return offsets
}

func rowColFromOffset(text string, offset int) (row, col int) {
	if offset < 0 {
		return 0, 0
	}
	if offset > len(text) {
		offset = len(text)
	}
	prefix := text[:offset]
	row = strings.Count(prefix, "\n")
	lastNL := strings.LastIndex(prefix, "\n")
	if lastNL < 0 {
		return row, offset
	}
	return row, offset - lastNL - 1
}
