package log

import "testing"

func TestConsoleLog(t *testing.T) {
    logger := newConsoleLogger()
    logger.Info("Console message")
    logger.Info("Console message with parameters: %s %d", "test", 100)
}