package store

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"sync"
	"time"
)

type WALEntry struct {
	Op        string
	Key       string
	Value     string
	ExpiresAt time.Time
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

	var buf bytes.Buffer

	var op uint8
	if entry.Op == "SET" {
		op = 1
	} else {
		op = 2
	}
	binary.Write(&buf, binary.LittleEndian, op)

	keyBytes := []byte(entry.Key)
	binary.Write(&buf, binary.LittleEndian, uint16(len(keyBytes)))
	buf.Write(keyBytes)

	valBytes := []byte(entry.Value)
	binary.Write(&buf, binary.LittleEndian, uint16(len(valBytes)))
	buf.Write(valBytes)

	var expiresAt int64
	if !entry.ExpiresAt.IsZero() {
		expiresAt = entry.ExpiresAt.Unix()
	}
	binary.Write(&buf, binary.LittleEndian, expiresAt)

	if _, err := w.writer.Write(buf.Bytes()); err != nil {
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

	reader := bufio.NewReader(w.file)

	for {
		var entry WALEntry

		var op uint8
		if err := binary.Read(reader, binary.LittleEndian, &op); err != nil {
			if err.Error() == "EOF" {
				break
			}
			return fmt.Errorf("failed to read op: %w", err)
		}

		if op == 1 {
			entry.Op = "SET"
		} else {
			entry.Op = "DEL"
		}

		var keyLen uint16
		if err := binary.Read(reader, binary.LittleEndian, &keyLen); err != nil {
			return fmt.Errorf("failed to read key len: %w", err)
		}
		keyBytes := make([]byte, keyLen)
		if _, err := reader.Read(keyBytes); err != nil {
			return fmt.Errorf("failed to read key: %w", err)
		}
		entry.Key = string(keyBytes)

		var valLen uint16
		if err := binary.Read(reader, binary.LittleEndian, &valLen); err != nil {
			return fmt.Errorf("failed to read val len: %w", err)
		}
		if valLen > 0 {
			valBytes := make([]byte, valLen)
			if _, err := reader.Read(valBytes); err != nil {
				return fmt.Errorf("failed to read value: %w", err)
			}
			entry.Value = string(valBytes)
		}

		var expiresAt int64
		if err := binary.Read(reader, binary.LittleEndian, &expiresAt); err != nil {
			return fmt.Errorf("failed to read expiresAt: %w", err)
		}
		if expiresAt > 0 {
			entry.ExpiresAt = time.Unix(expiresAt, 0)
		}

		if err := applyFn(entry); err != nil {
			return err
		}
	}

	return nil
}

func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	_ = w.writer.Flush()
	return w.file.Close()
}
