package logger

import (
	"sync"
	"time"
)

// LogEntry represents a single log entry
type LogEntry struct {
	ID        int64             `json:"id"`
	Level     string            `json:"level"`
	Message   string            `json:"message"`
	Timestamp time.Time         `json:"timestamp"`
	Attrs     map[string]string `json:"attrs,omitempty"`
}

// Buffer is a thread-safe ring buffer for log entries
type Buffer struct {
	capacity int
	entries  []*LogEntry
	head     int
	count    int
	mu       sync.RWMutex
}

// NewBuffer creates a new ring buffer with the specified capacity
func NewBuffer(capacity int) *Buffer {
	return &Buffer{
		capacity: capacity,
		entries:  make([]*LogEntry, capacity),
		head:     0,
		count:    0,
	}
}

// Add adds a new log entry to the buffer
func (b *Buffer) Add(entry *LogEntry) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Add entry at head position
	b.entries[b.head] = entry

	// Move head forward, wrapping around if needed
	b.head = (b.head + 1) % b.capacity

	// Increment count if not yet at capacity
	if b.count < b.capacity {
		b.count++
	}
}

// GetEntries returns up to limit entries from the buffer, optionally filtered by level
func (b *Buffer) GetEntries(level string, limit int) []*LogEntry {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if limit <= 0 || limit > b.count {
		limit = b.count
	}

	entries := make([]*LogEntry, 0, limit)

	// Start from the oldest entry (head - count) and move forward
	start := (b.head - b.count + b.capacity) % b.capacity
	for i := 0; i < limit; i++ {
		idx := (start + i) % b.capacity
		entry := b.entries[idx]
		if entry == nil {
			continue
		}
		if level != "" && entry.Level != level {
			continue
		}
		entries = append(entries, entry)
	}

	return entries
}

// Clear removes all entries from the buffer
func (b *Buffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.head = 0
	b.count = 0
	b.entries = make([]*LogEntry, b.capacity)
}

// Len returns the current number of entries in the buffer
func (b *Buffer) Len() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.count
}
