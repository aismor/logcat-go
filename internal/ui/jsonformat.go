package ui

import (
	"bytes"
	"encoding/json"
	"strings"
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

	startObj := strings.Index(text, "{")
	startArr := strings.Index(text, "[")
	start := -1
	switch {
	case startObj >= 0 && startArr >= 0:
		if startObj < startArr {
			start = startObj
		} else {
			start = startArr
		}
	case startObj >= 0:
		start = startObj
	case startArr >= 0:
		start = startArr
	}
	if start < 0 {
		return "", false
	}

	endObj := strings.LastIndex(text, "}")
	endArr := strings.LastIndex(text, "]")
	end := endObj
	if endArr > end {
		end = endArr
	}
	if end <= start {
		return "", false
	}

	candidate := text[start : end+1]
	if json.Valid([]byte(candidate)) {
		return candidate, true
	}

	if normalized, ok := normalizeJSONSelection(candidate); ok {
		return normalized, true
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
