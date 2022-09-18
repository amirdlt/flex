package core

import (
	"fmt"
	. "github.com/amirdlt/flex/common"
	"github.com/julienschmidt/httprouter"
	"io"
	"net/http"
	"reflect"
	"sync"
	"time"
)

type (
	Handler[SI ServerInjector, B any]               func(injector *HandlerInjector[SI, B]) HandlerResult
	UnhandledPanicHandler[SI ServerInjector, B any] func(*HandlerInjector[SI, B], any) HandlerResult
	WrapHandler[SI ServerInjector, B any]           func(Handler[SI, B]) Handler[SI, B]
	Middleware[SI ServerInjector, B any]            M

	LimitKeyGenerator[SI ServerInjector, B any] func(i *HandlerInjector[SI, B]) string

	limiter[SI ServerInjector, B any] struct {
		maxCount     int
		index        int
		interval     time.Duration
		requestTimes map[string][]time.Duration
		keyGenerator LimitKeyGenerator[SI, B]
		*sync.RWMutex
	}
)

func (l *limiter[_, _]) isAllowed(id string) bool {
	l.RLock()
	defer l.RUnlock()

	if l.index <= l.maxCount {
		return true
	}

	requestTimeList := l.requestTimes[id]

	var latest time.Duration
	index := l.index % (l.maxCount + 1)
	if index == 0 {
		latest = requestTimeList[l.maxCount]
	} else {
		latest = requestTimeList[index-1]
	}

	return latest-requestTimeList[index] > l.interval
}

func (l *limiter[_, _]) requestReceived(received time.Duration, id string) {
	l.Lock()

	requestTimeList, exist := l.requestTimes[id]
	if !exist {
		requestTimeList = make([]time.Duration, l.maxCount+1)
		l.requestTimes[id] = requestTimeList
	}

	_len := len(requestTimeList)

	if _len < l.maxCount {
		requestTimeList[_len] = received
	} else {
		requestTimeList[l.index%(l.maxCount+1)] = received
	}

	l.index++

	l.Unlock()
}

func NewMiddleware[SI ServerInjector, B any](server *Server[SI], handler Handler[SI, B]) Middleware[SI, B] {
	return M{
		"handler": handler,
		"server":  server,
	}
}

func (m Middleware[SI, B]) UseCustomLimiter(limitKeyGenerator LimitKeyGenerator[SI, B], maxCount int, interval time.Duration) Middleware[SI, B] {
	m["limiter::custom"] = limiter[SI, B]{
		maxCount:     maxCount,
		interval:     interval,
		index:        0,
		requestTimes: map[string][]time.Duration{},
		keyGenerator: limitKeyGenerator,
		RWMutex:      &sync.RWMutex{},
	}

	return m
}

func (m Middleware[SI, B]) UseDosLimiter(maxCount int, interval time.Duration) Middleware[SI, B] {
	m["limiter.dos"] = limiter[SI, B]{
		maxCount:     maxCount,
		interval:     interval,
		index:        0,
		requestTimes: map[string][]time.Duration{},
		keyGenerator: nil,
		RWMutex:      &sync.RWMutex{},
	}

	return m
}

func (m Middleware[SI, B]) WrapHandler(wrapper WrapHandler[SI, B]) Middleware[SI, B] {
	if _, exists := m["wrappers"]; !exists {
		m["wrappers"] = []any{}
	}

	m["wrappers"] = append(m["wrappers"].([]any), wrapper)

	return m
}

func (m Middleware[SI, B]) Register(method, path string) {
	server := m["server"].(*Server[SI])

	if serverWrappers, ok := server.middleware["wrappers"].([]any); ok {
		if middlewareWrappers, exist := m["wrappers"].([]any); exist {
			m["wrappers"] = append(middlewareWrappers, serverWrappers...)
		} else {
			m["wrappers"] = serverWrappers
		}
	}

	for k, v := range server.middleware {
		if _, exist := m[k]; !exist {
			m[k] = v
		}
	}

	handler := m["handler"].(Handler[SI, B])

	if wrappers, exist := m["wrappers"].([]any); exist {
		for _, w := range wrappers {
			if wrapper, ok := w.(WrapHandler[SI, B]); ok {
				handler = wrapper(handler)
			} else {
				handler = w.(func(any) any)(handler).(Handler[SI, B])
			}
		}
	}

	send := func(i ServerBaseInjector[SI], result HandlerResult) {
		if result.statusCode == 0 {
			result.statusCode = http.StatusOK
		}

		switch result.responseBody.(type) {
		case []byte, string, error: // ready already
		default:
			if marshalled, err := server.jsonHandler.Marshal(result.responseBody); err != nil {
				result = i.WrapInternalErr("error in json marshalling, err=" + err.Error())
				i.SetContentType("application/json")
			} else {
				result.responseBody = marshalled
			}
		}

		if _, exist := i.Headers()["Content-Type"]; !exist {
			i.SetContentType("application/json")
		}

		i.response().WriteHeader(result.statusCode)
		if _, err := fmt.Fprintf(i.response(), "%s", result.responseBody); err != nil {
			server.logger.Println("err while writing response, err" + err.Error())
		}
	}

	server.router.Handle(method, server.rootPath+path, func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		serverBaseI := ServerBaseInjector[SI]{
			owner:          server,
			pathParameters: params,
			r:              r,
			w:              w,
		}

		defer func() {
			if catch := recover(); catch != nil {
				send(serverBaseI, serverBaseI.WrapInternalErr("unhandled error occurred, err="+fmt.Sprint(catch)))
			}
		}()

		handlerI := HandlerInjector[SI, B]{
			SI:                 server.injector(serverBaseI),
			ServerBaseInjector: serverBaseI,
		}

		val := reflect.ValueOf(handlerI.requestBody)
		switch val.Kind() {
		case reflect.Array, reflect.Slice:
			if val.Type().Elem().Kind() == reflect.Uint8 {
				if arr, err := io.ReadAll(r.Body); err != nil {
					send(serverBaseI, serverBaseI.WrapBadRequestErr("could not read body"))
					return
				} else {
					val.SetBytes(arr)
				}
			}
		default:
			if !val.IsValid() || reflect.TypeOf(noBody) != val.Type() {
				if err := server.jsonHandler.NewDecoder(r.Body).Decode(&(handlerI.requestBody)); err != nil {
					send(serverBaseI, serverBaseI.WrapBadRequestErr("could not read body as desired schema, err="+err.Error()))
					return
				}
			}
		}

		send(serverBaseI, handler(&handlerI))
	})
}

func Register[SI ServerInjector, B any](server *Server[SI], method, path string, handler Handler[SI, B]) {
	server.middleware["handler"] = handler
	Middleware[SI, B](server.middleware).Register(method, path)
}
