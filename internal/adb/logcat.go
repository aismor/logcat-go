package adb

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aismor/logcat-go/internal/model"
)

const maxLogBuffer = 1000

type LogcatSession struct {
	device string
	filter model.FilterConfig

	mu         sync.RWMutex
	pidToUID   map[int]int
	cancel     context.CancelFunc
	cmd        *exec.Cmd
	entries    chan model.LogEntry
	done       chan struct{}
	uidRefresh time.Duration
}

func NewLogcatSession(device string, filter model.FilterConfig) *LogcatSession {
	return &LogcatSession{
		device:     device,
		filter:     filter,
		pidToUID:   make(map[int]int),
		entries:    make(chan model.LogEntry, 512),
		done:       make(chan struct{}),
		uidRefresh: 3 * time.Second,
	}
}

func (s *LogcatSession) Entries() <-chan model.LogEntry {
	return s.entries
}

func (s *LogcatSession) Done() <-chan struct{} {
	return s.done
}

func (s *LogcatSession) Start(parent context.Context, clearBuffer bool) error {
	ctx, cancel := context.WithCancel(parent)
	s.cancel = cancel

	if err := s.refreshPIDUIDMap(ctx); err != nil {
		// non-fatal: package filtering falls back to tag/message match
	}

	if clearBuffer {
		if err := ClearLogBuffer(ctx, s.device); err != nil {
			cancel()
			return fmt.Errorf("limpar buffer logcat: %w", err)
		}
	}

	go s.uidRefreshLoop(ctx)

	cmd := exec.CommandContext(ctx, "adb", "-s", s.device, "logcat", "-v", "threadtime")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("start logcat: %w", err)
	}

	s.cmd = cmd
	go s.readLoop(ctx, stdout)
	go s.waitLoop()

	return nil
}

func (s *LogcatSession) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *LogcatSession) waitLoop() {
	if s.cmd == nil {
		close(s.done)
		return
	}

	_ = s.cmd.Wait()
	close(s.done)
}

func (s *LogcatSession) uidRefreshLoop(ctx context.Context) {
	ticker := time.NewTicker(s.uidRefresh)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.refreshPIDUIDMap(ctx)
		}
	}
}

func (s *LogcatSession) refreshPIDUIDMap(ctx context.Context) error {
	out, err := run(ctx, s.device, "shell", "ps", "-A", "-o", "PID,UID,NAME")
	if err != nil {
		return err
	}

	next := make(map[int]int)
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if strings.EqualFold(fields[0], "PID") {
			continue
		}

		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		uid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		next[pid] = uid
	}

	s.mu.Lock()
	s.pidToUID = next
	s.mu.Unlock()

	return nil
}

func (s *LogcatSession) uidForPID(pid int) (int, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	uid, ok := s.pidToUID[pid]
	return uid, ok
}

func (s *LogcatSession) readLoop(ctx context.Context, r io.Reader) {
	defer close(s.entries)

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	assembler := NewLogAssembler()

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		entry, ok, isUpdate := assembler.Push(scanner.Text())
		if !ok || !IsDisplayable(entry) {
			continue
		}

		if !s.filter.Matches(entry, s.uidForPID) {
			continue
		}

		entry.IsUpdate = isUpdate

		select {
		case s.entries <- entry:
		case <-ctx.Done():
			return
		}
	}

	if err := scanner.Err(); err != nil && ctx.Err() == nil {
		select {
		case <-ctx.Done():
		default:
		}
	}
}

type LogStore struct {
	mu      sync.RWMutex
	entries []model.LogEntry
	limit   int
}

func NewLogStore(limit int) *LogStore {
	if limit <= 0 {
		limit = maxLogBuffer
	}
	return &LogStore{limit: limit}
}

func (s *LogStore) Append(entry model.LogEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries = append(s.entries, entry)
	if len(s.entries) > s.limit {
		s.entries = s.entries[len(s.entries)-s.limit:]
	}
}

func (s *LogStore) UpdateLast(entry model.LogEntry) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.entries) == 0 {
		return false
	}
	s.entries[len(s.entries)-1] = entry
	return true
}

func (s *LogStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = s.entries[:0]
}

func (s *LogStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries)
}

func (s *LogStore) At(index int) (model.LogEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if index < 0 || index >= len(s.entries) {
		return model.LogEntry{}, false
	}
	return s.entries[index], true
}

func (s *LogStore) All() []model.LogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.LogEntry, len(s.entries))
	copy(out, s.entries)
	return out
}
