package flex

import (
	"fmt"
	. "github.com/amirdlt/flex/util"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"reflect"
)

type (
	Handler[I Injector]    func(I) Result
	Wrapper[I Injector]    func(Handler[I]) Handler[I]
	Middleware[I Injector] struct {
		server   *Server[I]
		handler  Handler[I]
		wrappers Map[int, []Wrapper[I]]
	}

	BasicHandler    = Handler[*BasicInjector]
	BasicWrapper    = Wrapper[*BasicInjector]
	BasicMiddleware = Middleware[*BasicInjector]
)

func NewMiddleware[I Injector](handler Handler[I]) *Middleware[I] {
	m := newMiddleware((*Server[I])(nil))
	m.handler = handler
	return m
}

func newMiddleware[I Injector](server *Server[I]) *Middleware[I] {
	return &Middleware[I]{
		server:   server,
		wrappers: Map[int, []Wrapper[I]]{},
	}
}

func (m *Middleware[I]) WrapHandler(priority int, wrapper Wrapper[I]) *Middleware[I] {
	m.wrappers[priority] = append(m.wrappers[priority], wrapper)
	return m
}

func (m *Middleware[I]) serverMiddlewareClone(group ...*Server[I]) *Middleware[I] {
	var server *Server[I]
	switch len(group) {
	case 0:
		server = m.server
	case 1:
		server = group[0]
	default:
		panic("group must be one arg at max")
	}

	clone := newMiddleware(server)
	clone.wrappers = CopyMap(m.wrappers)
	return clone
}

func (m *Middleware[I]) mergeMiddleware(middleware *Middleware[I]) {

	// merge wrappers at upper level
	for k, v := range middleware.wrappers {
		m.wrappers[k] = append(m.wrappers[k], v...)
	}

	if middleware.handler != nil {
		m.handler = middleware.handler
	}

	if m.server == nil {
		m.server = middleware.server
	}
}

func (m *Middleware[I]) register(method, path string, bodyType reflect.Type, specialFixedPath ...bool) {
	if len(specialFixedPath) == 0 {
		specialFixedPath = []bool{false}
	}

	switch bodyType.Kind() {
	case reflect.Pointer, reflect.UnsafePointer, reflect.Chan, reflect.Func, reflect.Interface, reflect.Uintptr, reflect.Invalid:
		panic("inappropriate body type's kind: " + bodyType.String())
	}

	server := m.server
	handler := m.handler

	m.wrappers.Items().Sort(func(i, j Item[int, []Wrapper[I]]) bool {
		return i.Key() < j.Key()
	}).ForEach(func(item Item[int, []Wrapper[I]]) {
		for _, w := range item.Value() {
			handler = w(handler)
		}
	})

	if specialFixedPath[0] {
		server.router.HandleSpecialFixedPath(method, server.rootPath+path, func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
			httpRouterHandler(server, params, path, r, w, bodyType, handler)
		})
	} else {
		server.router.Handle(method, server.rootPath+path, func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
			httpRouterHandler(server, params, path, r, w, bodyType, handler)
		})
	}
}

func sendResponse(i *BasicInjector, result Result) {
	if result.terminate {
		return
	}

	if result.statusCode == 0 {
		result.statusCode = http.StatusOK
	}

	if result.responseBody != nil {
		switch result.responseBody.(type) {
		case []byte, string, error: // ready already
		default:
			if marshalled, err := i.jsonHandler.Marshal(result.responseBody); err != nil {
				result = i.WrapInternalErr("error in json marshalling, err=" + err.Error())
				i.SetContentType("application/json")
			} else {
				result.responseBody = marshalled
			}
		}
	}

	if _, exist := i.ResponseHeaders()["Content-Type"]; !exist {
		i.SetContentType("application/json")
	}

	i.w.WriteHeader(result.statusCode)
	if result.responseBody != nil {
		if _, err := fmt.Fprintf(i.w, "%s", result.responseBody); err != nil {
			i.LogPrintln("err while writing response, err=", err.Error())
		}
	}

	result.terminate = true
}

func httpRouterHandler[I Injector](
	server *Server[I],
	params httprouter.Params,
	path string,
	r *http.Request,
	w http.ResponseWriter,
	bodyType reflect.Type,
	handler Handler[I]) {
	baseI := server.CreateBasicInjector(path, params, r, w)
	baseI.bodyType = bodyType
	defer func() {
		if catch := recover(); catch != nil {
			if result, ok := catch.(Result); ok {
				sendResponse(baseI, result)
			} else {
				panic(catch)
			}
		}
	}()

	result := handler(server.injector(baseI))
	sendResponse(baseI, result)
}
