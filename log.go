package main

import (
	"fmt"
	"io"
	"log"
)

type Level int

const (
	LevelError Level = iota + 1
	LevelWarn
	LevelInfo
	LevelDebug
)

func (l Level) String() string {
	switch l {
	case LevelError:
		return "ERROR"
	case LevelWarn:
		return "WARNING"
	case LevelInfo:
		return "INFO"
	case LevelDebug:
		return "DEBUG"
	default:
		return "NOTSET"
	}
}

type Logger interface {
	Logf(level Level, format string, v ...interface{})
	Debugf(format string, v ...interface{})
	Errorf(format string, v ...interface{})
	Infof(format string, v ...interface{})
	Warnf(format string, v ...interface{})
}

type LeveledLogger struct {
	Level
	*log.Logger
}

func NewLogger(level Level, out io.Writer, prefix string) Logger {
	return LeveledLogger{
		Level:  level,
		Logger: log.New(out, prefix, log.LstdFlags),
	}
}

func (l LeveledLogger) Logf(level Level, format string, v ...interface{}) {
	if l.Level >= level {
		msg := fmt.Sprintf(format, v...)
		l.Printf("[%-7s] %s", level, msg)
	}
}

func (l LeveledLogger) Debugf(format string, v ...interface{}) { l.Logf(LevelDebug, format, v...) }
func (l LeveledLogger) Errorf(format string, v ...interface{}) { l.Logf(LevelError, format, v...) }
func (l LeveledLogger) Infof(format string, v ...interface{})  { l.Logf(LevelInfo, format, v...) }
func (l LeveledLogger) Warnf(format string, v ...interface{})  { l.Logf(LevelWarn, format, v...) }
