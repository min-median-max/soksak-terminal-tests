package system

import (
	"bytes"
	"encoding/json"
	"sync"
)

type controlReadyEvent struct {
	Event      string `json:"event"`
	Protocol   int    `json:"protocol"`
	Socket     string `json:"socket"`
	Identifier string `json:"identifier"`
	PID        int    `json:"pid"`
}

type controlReadyWriter struct {
	mu      sync.Mutex
	pending []byte
	events  chan controlReadyEvent
}

func newControlReadyWriter() *controlReadyWriter {
	return &controlReadyWriter{events: make(chan controlReadyEvent, 1)}
}

func (writer *controlReadyWriter) Write(payload []byte) (int, error) {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	writer.pending = append(writer.pending, payload...)
	for {
		newline := bytes.IndexByte(writer.pending, '\n')
		if newline < 0 {
			break
		}
		line := append([]byte(nil), writer.pending[:newline]...)
		writer.pending = writer.pending[newline+1:]
		var event controlReadyEvent
		if json.Unmarshal(line, &event) == nil && event.Event == testControlReadyEvent {
			select {
			case writer.events <- event:
			default:
			}
		}
	}
	return len(payload), nil
}
