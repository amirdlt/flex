package flex

import (
	"fmt"
	"io"
	"log"
)

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
