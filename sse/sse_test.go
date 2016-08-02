package sse

import "testing"

func TestGetConnectionCount(t *testing.T) {
    es := NewEventSource(false, false)

    if es.ConnectionCount() != 0 {
        t.Fatal("There's clients connected. How?")
    }
}
