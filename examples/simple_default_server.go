package main

import (
	. "github.com/amirdlt/flex"
	. "github.com/amirdlt/flex/util"
)

func simpleDefaultServer() {
	server := Default().WrapHandler(0, func(h Handler[*BasicInjector]) Handler[*BasicInjector] {
		return func(i *BasicInjector) Result {
			i.Logger().Println("Pinged: ", i.RemoteAddr())
			return h(i)
		}
	})

	server.GET("/", func(i *BasicInjector) Result {
		return i.WrapOk(M{
			"state": "pong",
		})
	}, NoBody{})

	g := server.Group("/info")

	g.POST("/", func(i *BasicInjector) Result {
		return i.WrapOk(M{
			"received": i.RequestBody().(M),
		})
	}, M{})

	_ = server.Run()
}
