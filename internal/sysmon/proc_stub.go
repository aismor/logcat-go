//go:build !linux

package sysmon

import (
	"runtime"
)

func readRAMMB() float64 {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return float64(stats.Alloc) / (1024 * 1024)
}

func (m *Monitor) sampleCPU() float64 {
	_ = m
	return 0
}
