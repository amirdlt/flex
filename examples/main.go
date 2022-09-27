package main

import (
	"fmt"
	. "github.com/amirdlt/flex"
	"github.com/amirdlt/flex/middleware"
	. "github.com/amirdlt/flex/util"
	"log"
	"net/http"
	"os"
	"time"
)

type userInfo struct {
	Firstname string `json:"name"`
	Age       int    `json:"age"`
}

type injector struct {
	Username string
	*BasicInjector
}

func main() {
	type me struct {
		Name string
	}

	m := Map[string, me]{
		"adlt": me{"amirdlt"},
	}

	fmt.Println(m)

	server := NewServer[*injector](M{}, func(i *BasicInjector) *injector {
		return &injector{
			BasicInjector: i,
		}
	}).WrapHandler(201, middleware.Monitor(&injector{}, os.Stdout))

	//server.SetDefaultMongoClient("mongodb+srv://amirdlt:amirdlt2000@amirdltapp.7srwd.mongodb.net/?retryWrites=true&w=majority")
	server.SetDefaultMongoClient("mongodb://localhost:27017")

	fmt.Println("Halle")

	server.WrapHandler(200, middleware.DosLimiter(320, 10*time.Second, func(i *injector) Result {
		return i.WrapTooManyRequestsErr("dos limiter is here")
	}))

	g1 := server.Group("/v1").Group("/v2")

	g1.WrapHandler(100, func(handler Handler[*injector]) Handler[*injector] {
		return func(i *injector) Result {
			fmt.Println("Hi im from server middleware")

			return handler(i)
		}
	}).WrapHandler(-1, middleware.PanicHandler(func(i *injector, catch any) Result {
		return i.WrapInternalErr(fmt.Sprint(catch))
	}))

	g1.Handle(http.MethodGet, "/ping", NewMiddleware(func(i *injector) Result {
		fmt.Println(i.Username)
		//panic("not implemented")
		sum := 0
		for i := 0; i < 1_000_000_000; i++ {
			sum += i
		}

		i.SetValue("sum", sum)

		return i.WrapOk(i.GetDataMap())
	}).WrapHandler(0, func(handler Handler[*injector]) Handler[*injector] {
		return func(i *injector) Result {
			fmt.Println("Hi im from outer middleware")
			i.Username = "Amir?"
			return handler(i)
		}
	}).WrapHandler(0, func(handler Handler[*injector]) Handler[*injector] {
		return func(i *injector) Result {
			fmt.Println("Hi im from second outer middleware")
			i.Username = "Amir?"
			return handler(i)
		}
	}), NoBody{})

	g1.Handle(http.MethodGet, "/ping2", func(i *injector) Result {
		fmt.Println(i.RequestBody())
		return i.WrapOk("pong2")
	}, 0)

	log.Fatal(server.Run())
	//	"mongodb+srv://amirdlt:amirdlt2000@amirdltapp.7srwd.mongodb.net/?retryWrites=true&w=majority"
}
