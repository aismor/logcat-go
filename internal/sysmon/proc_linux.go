//go:build linux

package sysmon

import (
	"bufio"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const procClockTicks = 100.0

func readRAMMB() float64 {
	f, err := os.Open("/proc/self/status")
	if err != nil {
		return readMemStatsMB()
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "VmRSS:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			break
		}
		kb, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			break
		}
		return kb / 1024.0
	}
	return readMemStatsMB()
}

func readMemStatsMB() float64 {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return float64(stats.Alloc) / (1024 * 1024)
}

func (m *Monitor) sampleCPU() float64 {
	ticks, ok := readProcessCPUTicks()
	now := time.Now()

	if !ok {
		return 0
	}
	if !m.initialized {
		m.lastCPUTicks = ticks
		m.lastSample = now
		m.initialized = true
		return 0
	}

	elapsed := now.Sub(m.lastSample).Seconds()
	if elapsed <= 0 {
		return 0
	}

	delta := float64(ticks - m.lastCPUTicks)
	m.lastCPUTicks = ticks
	m.lastSample = now

	seconds := delta / procClockTicks
	pct := (seconds / elapsed) * 100.0
	if cores := runtime.NumCPU(); cores > 0 {
		pct /= float64(cores)
	}
	if pct < 0 {
		return 0
	}
	return pct
}

func readProcessCPUTicks() (uint64, bool) {
	data, err := os.ReadFile("/proc/self/stat")
	if err != nil {
		return 0, false
	}

	closeIdx := strings.LastIndexByte(string(data), ')')
	if closeIdx < 0 || closeIdx+2 >= len(data) {
		return 0, false
	}

	fields := strings.Fields(string(data[closeIdx+2:]))
	if len(fields) < 12 {
		return 0, false
	}

	utime, err := strconv.ParseUint(fields[11], 10, 64)
	if err != nil {
		return 0, false
	}
	stime, err := strconv.ParseUint(fields[12], 10, 64)
	if err != nil {
		return 0, false
	}
	return utime + stime, true
}
