package flex

import (
	"fmt"
	"log"
)

type logger struct {
	*log.Logger
}

func (l logger) Print(v ...any) {
	_ = l.Output(3, fmt.Sprint(v...))
}

func (l logger) Printf(format string, v ...any) {
	_ = l.Output(3, fmt.Sprintf(format, v...))
}

func (l logger) Println(v ...any) {
	_ = l.Output(3, fmt.Sprintln(v...))
}
