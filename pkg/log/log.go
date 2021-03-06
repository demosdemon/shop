package log

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
	LevelTrace
)

var levelStrings = map[Level]string{
	LevelError: "ERROR",
	LevelWarn:  "WARNING",
	LevelInfo:  "INFO",
	LevelDebug: "DEBUG",
	LevelTrace: "TRACE",
}

func (l Level) String() string {
	s := levelStrings[l]
	if s == "" {
		s = fmt.Sprintf("Level(%d)", l)
	}
	return s
}

type Logger interface {
	Logf(level Level, format string, v ...interface{})
	Errorf(format string, v ...interface{})
	Warnf(format string, v ...interface{})
	Infof(format string, v ...interface{})
	Debugf(format string, v ...interface{})
	Tracef(format string, v ...interface{})
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

func (l LeveledLogger) Errorf(format string, v ...interface{}) { l.Logf(LevelError, format, v...) }
func (l LeveledLogger) Warnf(format string, v ...interface{})  { l.Logf(LevelWarn, format, v...) }
func (l LeveledLogger) Infof(format string, v ...interface{})  { l.Logf(LevelInfo, format, v...) }
func (l LeveledLogger) Debugf(format string, v ...interface{}) { l.Logf(LevelDebug, format, v...) }
func (l LeveledLogger) Tracef(format string, v ...interface{}) { l.Logf(LevelTrace, format, v...) }
