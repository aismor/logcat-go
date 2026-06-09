package sysmon

import "time"

// Snapshot holds a point-in-time reading of this process.
type Snapshot struct {
	RAMMB  float64
	CPUPct float64
}

// Monitor samples RAM and CPU usage of the current process.
type Monitor struct {
	lastCPUTicks uint64
	lastSample   time.Time
	initialized  bool
}

// New creates a process monitor.
func New() *Monitor {
	return &Monitor{}
}

// Sample returns current RAM (MB) and CPU (% of one core, 0–100+).
func (m *Monitor) Sample() Snapshot {
	ram := readRAMMB()
	cpu := m.sampleCPU()
	return Snapshot{RAMMB: ram, CPUPct: cpu}
}
