package middleware

import (
	"fmt"
	"github.com/amirdlt/flex/common"
	. "github.com/amirdlt/flex/flx"
	"io"
	"net"
	"strings"
	"time"
)

func Monitor[I Injector](_ I, w io.Writer) Wrapper[I] {
	return func(h Handler[I]) Handler[I] {
		return func(i I) Result {
			t := time.Now()

			defer func() {
				_, _ = fmt.Fprintln(w, common.Map[string, any]{
					"duration":    time.Since(t),
					"client":      net.ParseIP(strings.Trim(i.RemoteAddr()[:strings.LastIndex(i.RemoteAddr(), ":")], "[]")),
					"client_str":  i.RemoteAddr(),
					"forwarded":   i.GetRequestHeader("X-Forwarded-For"),
					"url":         i.URL(),
					"host":        i.Host(),
					"method":      i.Method(),
					"content-len": i.ContentLength(),
				})
			}()

			return h(i)
		}
	}
}
