package ui

import (
	"strings"

	"github.com/aismor/logcat-go/internal/model"
)

func filterEntriesBySearch(entries []model.LogEntry, query string) []model.LogEntry {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return entries
	}

	filtered := make([]model.LogEntry, 0, len(entries)/4)
	for _, entry := range entries {
		if entryMatchesSearch(entry, query) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func entryMatchesSearch(entry model.LogEntry, query string) bool {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return true
	}

	parts := strings.Fields(query)
	haystack := strings.ToLower(formatPlainLine(entry))
	for _, part := range parts {
		if !strings.Contains(haystack, part) {
			return false
		}
	}
	return true
}
