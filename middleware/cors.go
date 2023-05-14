package middleware

import (
	. "github.com/amirdlt/flex"
	"net/http"
)

func CORS[I Injector]() Wrapper[I] {
	return func(inner Handler[I]) Handler[I] {
		return func(i I) Result {
			result := inner(i)

			i.ResponseHeaders().Set("Access-Control-Allow-Origin", "*")
			i.ResponseHeaders().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
			i.ResponseHeaders().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
			i.ResponseHeaders().Set("Access-Control-Allow-Credentials", "true")

			if i.Method() == http.MethodOptions {
				result = i.WrapNoContent()
			}

			return result
		}
	}
}
