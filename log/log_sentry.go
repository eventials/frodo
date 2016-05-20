package log

import (
    "fmt"

    "github.com/getsentry/raven-go"
)

type SentryLogger struct {
}

func newSentryLogger(dsnUrl string) *SentryLogger {
    raven.SetDSN(dsnUrl)
    return &SentryLogger{}
}

func (logger *SentryLogger) Panic(format string, args ...interface{}) {
    logger.captureError(format, args...)
}

func (logger *SentryLogger) Fatal(format string, args ...interface{}) {
    logger.captureError(format, args...)
}

func (logger *SentryLogger) Error(format string, args ...interface{}) {
    logger.captureError(format, args...)
}

func (logger *SentryLogger) Warning(format string, args ...interface{}) { }

func (logger *SentryLogger) Info(format string, args ...interface{}) { }

func (logger *SentryLogger) captureError(format string, args ...interface{}) {
    err := fmt.Errorf(format, args...)
    raven.CaptureErrorAndWait(err, nil)
}