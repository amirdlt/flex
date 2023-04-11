package flex

import (
	"fmt"
	. "github.com/amirdlt/flex/util"
	"io"
	"log"
	"sync"
)

type loggerLevel struct {
	levels M
	*sync.RWMutex
}

const (
	LogPrintLevel = "print"
	LogErrorLevel = "error"
	LogWarnLevel  = "warn"
	LogTraceLevel = "trace"
	LogDebugLevel = "debug"
	LogInfoLevel  = "info"
	LogFatalLevel = "fatal"
)

func (l loggerLevel) isEnabledLogLevel(level string) bool {
	l.RLock()
	defer l.RUnlock()

	_, has := l.levels[level]
	return has
}

func (l loggerLevel) enableLogLevel(level string) {
	l.Lock()
	defer l.Unlock()

	l.levels[level] = nil
}

func (l loggerLevel) disableLogLevel(level string) {
	l.Lock()
	defer l.Unlock()

	delete(l.levels, level)
}

var defaultLoggerLevels = M{
	"print": nil,
	"trace": nil,
	"debug": nil,
	"info":  nil,
	"warn":  nil,
	"error": nil,
	"fatal": nil,
}

type logger struct {
	*log.Logger
	out io.Writer
}

func (l *logger) print(v ...any) {
	_ = l.Output(3, fmt.Sprint(v...))
}

func (l *logger) printf(format string, v ...any) {
	_ = l.Output(3, fmt.Sprintf(format, v...))
}

func (l *logger) println(v ...any) {
	_ = l.Output(3, fmt.Sprintln(v...))
}

func (l *logger) SetOutput(w io.Writer) {
	l.out = w
	l.Logger.SetOutput(w)
}
