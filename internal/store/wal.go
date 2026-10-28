package store

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type WALEntry struct {
	Op        string    `json:"op"`
	Key       string    `json:"key"`
	Value     string    `json:"value,omitempty"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

type WAL struct {
	file   *os.File
	writer *bufio.Writer
	mu     sync.Mutex
}

func NewWAL(filepath string) (*WAL, error) {
	f, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open WAL file: %w", err)
	}

	return &WAL{
		file:   f,
		writer: bufio.NewWriterSize(f, 32*1024),
	}, nil
}

func (w *WAL) Append(entry WALEntry) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	if _, err := w.writer.Write(data); err != nil {
		return err
	}
	if _, err := w.writer.WriteString("\n"); err != nil {
		return err
	}

	if err := w.writer.Flush(); err != nil {
		return err
	}

	return w.file.Sync()
}

func (w *WAL) Replay(applyFn func(WALEntry) error) error {
	if _, err := w.file.Seek(0, 0); err != nil {
		return err
	}

	scanner := bufio.NewScanner(w.file)
	for scanner.Scan() {
		var entry WALEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return fmt.Errorf("failed to unmarshal WAL entry: %w", err)
		}
		if err := applyFn(entry); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	_ = w.writer.Flush()
	return w.file.Close()
}
