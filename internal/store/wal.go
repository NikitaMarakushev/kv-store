package store

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
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
	file        *os.File
	writer      *bufio.Writer
	mu          sync.Mutex
	syncCounter int
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

	w.syncCounter++
	if w.syncCounter >= 1000 {
		w.syncCounter = 0
		if err := w.file.Sync(); err != nil {
			return err
		}
	}

	return nil
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
			if err == io.EOF {
				break
			}
			log.Printf("WAL replay stopped (partial write): %v", err)
			break
		}

		if op != 1 && op != 2 {
			log.Printf("WAL replay stopped: invalid op code %d (possible old JSON format or corruption)", op)
			break
		}

		if op == 1 {
			entry.Op = "SET"
		} else {
			entry.Op = "DEL"
		}

		var keyLen uint16
		if err := binary.Read(reader, binary.LittleEndian, &keyLen); err != nil {
			log.Printf("WAL replay stopped at key len: %v", err)
			break
		}

		if keyLen > 10000 {
			log.Printf("WAL replay stopped: suspiciously large keyLen %d", keyLen)
			break
		}

		keyBytes := make([]byte, keyLen)
		if _, err := io.ReadFull(reader, keyBytes); err != nil {
			log.Printf("WAL replay stopped at key: %v", err)
			break
		}
		entry.Key = string(keyBytes)

		var valLen uint16
		if err := binary.Read(reader, binary.LittleEndian, &valLen); err != nil {
			log.Printf("WAL replay stopped at val len: %v", err)
			break
		}

		if valLen > 65000 {
			log.Printf("WAL replay stopped: suspiciously large valLen %d", valLen)
			break
		}

		if valLen > 0 {
			valBytes := make([]byte, valLen)
			if _, err := io.ReadFull(reader, valBytes); err != nil {
				log.Printf("WAL replay stopped at value: %v", err)
				break
			}
			entry.Value = string(valBytes)
		}

		var expiresAt int64
		if err := binary.Read(reader, binary.LittleEndian, &expiresAt); err != nil {
			log.Printf("WAL replay stopped at expiresAt: %v", err)
			break
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
	_ = w.file.Sync()
	return w.file.Close()
}
