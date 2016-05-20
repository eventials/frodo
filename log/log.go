package log

import (
    "fmt"
    "os"
)

type LogLevel string

const (
    LPanic   LogLevel = "PANIC"
    LFatal   LogLevel = "FATAL"
    LError   LogLevel = "ERROR"
    LWarning LogLevel = "WARNING"
    LInfo    LogLevel = "INFO"
)

type Logger interface {
    Panic(format string, args ...interface{})
    Fatal(format string, args ...interface{})
    Error(format string, args ...interface{})
    Warning(format string, args ...interface{})
    Info(format string, args ...interface{})
}

var (
    handlers = make(map[string]Logger)
)

func AddHandler(loggerName string, args ...interface{}) error {
    var logger Logger

    switch loggerName {
        case "console":
            logger = newConsoleLogger()
        case "sentry":
            if len(args) == 0 {
                return fmt.Errorf("Missing paramenter: DSN")
            }

            dsnUrl, ok := args[0].(string)

            if !ok {
                return fmt.Errorf("DSN paramenter must be string.")
            }

            logger = newSentryLogger(dsnUrl)
    }

    handlers[loggerName] = logger
    return nil
}

func Panic(format string, args ...interface{}) {
    for _, logger := range handlers {
        logger.Panic(format, args...)
    }

    msg := fmt.Sprintf(format, args...)
    panic(msg)
}

func Fatal(format string, args ...interface{}) {
    for _, logger := range handlers {
        logger.Fatal(format, args...)
    }

    os.Exit(1)
}

func Error(format string, args ...interface{}) {
    for _, logger := range handlers {
        logger.Error(format, args...)
    }
}

func Warning(format string, args ...interface{}) {
    for _, logger := range handlers {
        logger.Warning(format, args...)
    }
}

func Info(format string, args ...interface{}) {
    for _, logger := range handlers {
        logger.Info(format, args...)
    }
}
