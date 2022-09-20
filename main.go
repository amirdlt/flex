package main

import (
	"fmt"
	. "github.com/amirdlt/flex/common"
	. "github.com/amirdlt/flex/core"
	"log"
	"net/http"
	"time"
)

type userInfo struct {
	Firstname string `json:"name"`
	Age       int    `json:"age"`
}

type _ServerInjector struct {
	Username string
	ServerBaseInjector[_ServerInjector]
}

func main() {
	server := NewServer[_ServerInjector](M{}, func(b ServerBaseInjector[_ServerInjector]) *_ServerInjector {
		return &_ServerInjector{
			Username:           "AmirDLT2000",
			ServerBaseInjector: b,
		}
	})

	server.SetDefaultMongoClient("mongodb://localhost:27017")

	fmt.Println("Halle")

	g1 := server.Group("/v1").Group("/v2")

	Register(g1, http.MethodGet, "/ping", func(i *HandlerInjector[_ServerInjector, NoBody]) HandlerResult {
		return i.WrapTextPlain("pong", http.StatusOK)
	})

	g1.WrapHandler(func(handler any) any {
		return func() any {
			fmt.Println("Hi im from server middleware")

			return handler
		}
	})

	NewMiddleware(g1, func(i *HandlerInjector[_ServerInjector, userInfo]) HandlerResult {
		fmt.Println("dev req body=", i.RequestBody())
		fmt.Println("dev: ", i.SI.Username)
		fmt.Println("dev: ", i.Method())
		if _, err := i.Server().GetDefaultMongoClient().GetDatabase("my-app").GetCollection("my-col").InsertOne(i.Context(), M{
			"time": time.Now(),
		}); err != nil {
			return i.WrapInternalErr("could not insert time to collection")
		}

		return i.WrapOk("ok i get it")
	}).WrapHandler(func(h Handler[_ServerInjector, userInfo]) Handler[_ServerInjector, userInfo] {
		return func(i *HandlerInjector[_ServerInjector, userInfo]) HandlerResult {
			t := time.Now()
			defer func() {
				fmt.Println("wrapper: ", time.Since(t).Milliseconds())
			}()

			return h(i)
		}
	}).Register(http.MethodPost, "/")

	log.Fatal(server.Run(5000))
}
