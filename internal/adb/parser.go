package adb

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/aismor/logcat-go/internal/model"
)

var (
	threadtimeLine = regexp.MustCompile(`^(\d{2}-\d{2})\s+(\d{2}:\d{2}:\d{2}\.\d{3})\s+(\d+)\s+(\d+)\s+([VDIWEF])\s+([^:]+):\s?(.*)$`)
	longLine       = regexp.MustCompile(`^\[\s*(\d+)\s+(\d+)\s+(\d+)\s+([VDIWEF])\s+([^:]+)\s*\]\s*(.*)$`)
)

func ParseLogLine(line string) (model.LogEntry, bool) {
	line = strings.TrimRight(line, "\r\n")
	if line == "" {
		return model.LogEntry{}, false
	}

	if matches := threadtimeLine.FindStringSubmatch(line); len(matches) == 8 {
		pid, _ := strconv.Atoi(matches[3])
		tid, _ := strconv.Atoi(matches[4])
		return model.LogEntry{
			Raw:     line,
			Date:    matches[1],
			Time:    matches[2],
			PID:     pid,
			TID:     tid,
			Level:   matches[5],
			Tag:     matches[6],
			Message: matches[7],
		}, true
	}

	if matches := longLine.FindStringSubmatch(line); len(matches) == 7 {
		uid, _ := strconv.Atoi(matches[1])
		pid, _ := strconv.Atoi(matches[2])
		tid, _ := strconv.Atoi(matches[3])
		return model.LogEntry{
			Raw:     line,
			UID:     uid,
			PID:     pid,
			TID:     tid,
			Level:   matches[4],
			Tag:     strings.TrimSpace(matches[5]),
			Message: matches[6],
		}, true
	}

	return model.LogEntry{Raw: line}, true
}
