package core

import (
	"fmt"
	"github.com/amirdlt/flex/common"
	. "github.com/amirdlt/flex/common"
	"github.com/amirdlt/flex/db/mongo"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"os"
)

type Server[SI ServerInjector] struct {
	defaultErrorCodes map[int]string
	config            common.M
	router            *httprouter.Router
	logger            *log.Logger
	rootPath          string
	parent            *Server[SI]
	injector          func(injector ServerBaseInjector[SI]) *SI
	mongoClients      mongo.Clients
	groups            map[string]*Server[SI]
	middleware        Middleware[SI, any]
	jsonHandler       JsonHandler
}

func NewServer[SI ServerInjector](config common.M, injector func(baseInjector ServerBaseInjector[SI]) *SI) *Server[SI] {
	middleware := M{}
	s := &Server[SI]{
		logger:            log.Default(),
		config:            config,
		parent:            nil,
		rootPath:          "",
		router:            httprouter.New(),
		defaultErrorCodes: getDefaultErrorCodes(),
		injector:          injector,
		mongoClients:      mongo.Clients{},
		groups:            map[string]*Server[SI]{},
		middleware:        middleware,
		jsonHandler:       DefaultJsonHandler{},
	}

	middleware["server"] = s
	return s
}

func (s *Server[_]) GetMongoClients() mongo.Clients {
	return s.mongoClients
}

func (s *Server[_]) SetJsonHandler(jsonHandler JsonHandler) {
	s.jsonHandler = jsonHandler
}

func (s *Server[_]) GetRootPath() string {
	return s.rootPath
}

func (s *Server[_]) GetLogger() *log.Logger {
	return s.logger
}

func (s *Server[SI]) GetMiddleware() Middleware[SI, any] {
	return s.middleware
}

func (s *Server[SI]) SetDefaultMongoClient(connectionUrl string) {
	if err := s.mongoClients.AddClient("", connectionUrl); err != nil {
		panic("could not connect to mongo client: " + err.Error())
	}
}

func (s *Server[SI]) AddMongoClient(name, connectionUrl string) {
	if name == "" {
		panic("mongo client name cannot be empty")
	}

	if err := s.mongoClients.AddClient(name, connectionUrl); err != nil {
		panic("could not connect to mongo client: " + err.Error())
	}
}

func (s *Server[SI]) Group(path string) *Server[SI] {
	if path == "" {
		return s
	}

	if g, exists := s.groups[path]; exists {
		return g
	} else {
		middleware := CopyMap(s.middleware)
		s.groups[path] = &Server[SI]{
			rootPath:          s.rootPath + path,
			logger:            log.New(os.Stderr, s.rootPath+path, log.LstdFlags),
			parent:            s,
			router:            s.router,
			defaultErrorCodes: getDefaultErrorCodes(),
			injector:          s.injector,
			groups:            map[string]*Server[SI]{},
			mongoClients:      s.mongoClients,
			middleware:        middleware,
			jsonHandler:       s.jsonHandler,
		}

		middleware["server"] = s.groups[path]

		return s.groups[path]
	}

}

func (s *Server[_]) Run(port int) error {
	return http.ListenAndServe(":"+fmt.Sprint(port), s.router)
}

func (s *Server[_]) GetMongoClient(name string) mongo.Client {
	if c, exist := s.mongoClients[name]; exist {
		return c
	}

	panic("no client found with this name: " + name)
}

func (s *Server[_]) GetDefaultMongoClient() mongo.Client {
	return s.GetMongoClient("")
}

func (s *Server[_]) Cleanup() {
	s.mongoClients.ClearAllClients()
}

func (s *Server[_]) GetRouter() *httprouter.Router {
	return s.router
}

func (s *Server[SI]) WrapHandler(wrapper func(handler any) any) *Server[SI] {
	m := s.middleware

	if _, exists := m["wrappers"]; !exists {
		m["wrappers"] = []any{}
	}

	m["wrappers"] = append(m["wrappers"].([]any), wrapper)
	return s
}

func getDefaultErrorCodes() map[int]string {
	return map[int]string{
		http.StatusBadRequest:          "ERR_BAD_REQUEST",
		http.StatusInternalServerError: "ERR_INTERNAL_SERVER",
		http.StatusTooManyRequests:     "ERR_TOO_MANY_REQUESTS",
		http.StatusNotFound:            "ERR_NOT_FOUND",
		http.StatusFound:               "ERR_ALREADY_EXIST",
		http.StatusConflict:            "ERR_CONFLICT",
		http.StatusForbidden:           "ERR_FORBIDDEN",
		http.StatusNotImplemented:      "ERR_NOT_IMPLEMENTED",
	}
}
