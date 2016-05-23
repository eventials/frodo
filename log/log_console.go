package log

import (
    "fmt"
    "log"
    "os"
)

type ConsoleLogger struct {
    std *log.Logger
}

func newConsoleLogger() *ConsoleLogger {
    return &ConsoleLogger{
        log.New(os.Stderr, "", log.LstdFlags),
    }
}

func (logger *ConsoleLogger) Panic(format string, args ...interface{}) {
    logger.all(LPanic, format, args...)
}

func (logger *ConsoleLogger) Fatal(format string, args ...interface{}) {
    logger.all(LFatal, format, args...)
}

func (logger *ConsoleLogger) Error(format string, args ...interface{}) {
    logger.all(LError, format, args...)
}

func (logger *ConsoleLogger) Warning(format string, args ...interface{}) {
    logger.all(LWarning, format, args...)
}

func (logger *ConsoleLogger) Info(format string, args ...interface{}) {
    logger.all(LInfo, format, args...)
}

func (logger *ConsoleLogger) all(logLevel LogLevel, format string, args ...interface{}) {
    level := fmt.Sprintf("[%s] ", logLevel)
    format = level + format + "\n"

    msg := fmt.Sprintf(format, args...)
    logger.std.Output(2, msg)
}
