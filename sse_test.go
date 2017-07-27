package frodo

import "testing"

func TestGetConnectionCount(t *testing.T) {
	es := NewEventSource()

	if es.ConnectionCount() != 0 {
		t.Fatal("There's clients connected. How?")
	}
}
