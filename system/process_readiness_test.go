package system

import "testing"

func TestControlReadyWriterConsumesOneSplitOwnedEvent(t *testing.T) {
	writer := newControlReadyWriter()
	first := `noise before event
{"event":"soksak.host.`
	second := `ready","protocol":1,"socket":"/tmp/control.sock","identifier":"com.soksak.test","pid":42}
noise after event
`
	if _, err := writer.Write([]byte(first)); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte(second)); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-writer.events:
		if event.Protocol != 1 || event.Socket != "/tmp/control.sock" ||
			event.Identifier != "com.soksak.test" || event.PID != 42 {
			t.Fatalf("event=%+v", event)
		}
	default:
		t.Fatal("split readiness event was not published")
	}
}
