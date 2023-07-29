package middleware

import (
	. "github.com/amirdlt/flex"
)

type PanicHandlerFunc[I Injector] func(i I, catch any) Result

func PanicHandler[I Injector](panicHandler PanicHandlerFunc[I]) Wrapper[I] {
	return func(h Handler[I]) Handler[I] {
		return func(i I) (r Result) {
			defer func() {
				if catch := recover(); catch != nil {
					if result, ok := catch.(Result); ok {
						r = result
						return
					}

					r = panicHandler(i, catch)
				}
			}()

			r = h(i)
			return
		}
	}
}
