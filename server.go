package flex

import (
	"context"
	"fmt"
	"github.com/amirdlt/flex/db/mongo"
	. "github.com/amirdlt/flex/util"
	"github.com/julienschmidt/httprouter"
	"io"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"
)

type Server[I Injector] struct {
	defaultErrorCodes map[int]string
	config            M
	router            Router
	logger            logger
	rootPath          string
	parent            *Server[I]
	injector          func(*BasicInjector) I
	mongoClients      mongo.Clients
	groups            map[string]*Server[I]
	jsonHandler       JsonHandler
	middleware        *Middleware[I]
	httpServer        *http.Server
	startTime         time.Time
}

type BasicServer = Server[*BasicInjector]

func New[I Injector](config M, injector func(baseInjector *BasicInjector) I) *Server[I] {
	if injector == nil {
		panic("injector can not be nil")
	}

	var i I
	if val := reflect.ValueOf(i); val.Kind() != reflect.Ptr {
		panic("expected a pointer as an handler injector, got " + val.Kind().String())
	}

	if config == nil {
		config = M{}
	}

	logger := logger{log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile), os.Stderr}
	if loggerOut, exist := config["logger_out"]; exist {
		if f, err := GetFileOutputStream(loggerOut.(string)); err != nil {
			panic(err)
		} else {
			logger.SetOutput(f)
		}
	}

	s := &Server[I]{
		logger:            logger,
		config:            config,
		parent:            nil,
		rootPath:          "",
		router:            Router{Router: httprouter.New(), apis: map[string][]string{}},
		defaultErrorCodes: getDefaultErrorCodes(),
		injector:          injector,
		mongoClients:      mongo.Clients{},
		groups:            map[string]*Server[I]{},
		jsonHandler:       DefaultJsonHandler{},
	}

	s.middleware = newMiddleware(s)

	if server, exist := s.LookupConfig("server"); exist {
		if hs, ok := server.(*http.Server); !ok {
			panic("expected an *http.Server, got " + fmt.Sprint(server))
		} else if hs.Handler != nil {
			panic("http.Server.Handler must be nil, as it set by server construction")
		} else {
			hs.Handler = s.router
			s.httpServer = hs
		}
	} else {
		s.httpServer = &http.Server{Handler: s.router}
	}

	return s
}

func Default() *Server[*BasicInjector] {
	return New(M{}, func(bi *BasicInjector) *BasicInjector {
		return bi
	})
}

func (s *Server[_]) StartTime() time.Time {
	return s.startTime
}

func (s *Server[_]) LookupConfig(key string) (any, bool) {
	v, exist := s.config[key]
	return v, exist
}

func (s *Server[_]) Config(key string) any {
	return s.config[key]
}

func (s *Server[_]) MongoClients() mongo.Clients {
	return s.mongoClients
}

func (s *Server[_]) SetJsonHandler(jsonHandler JsonHandler) {
	s.jsonHandler = jsonHandler
}

func (s *Server[_]) RootPath() string {
	return s.rootPath
}

func (s *Server[I]) SetDefaultMongoClient(connectionUrl string) {
	if err := s.mongoClients.AddClient("", connectionUrl); err != nil {
		panic("could not connect to mongo client: " + err.Error())
	}
}

func (s *Server[I]) AddMongoClient(name, connectionUrl string) {
	if name == "" {
		panic("mongo client name cannot be empty")
	}

	if err := s.mongoClients.AddClient(name, connectionUrl); err != nil {
		panic("could not connect to mongo client: " + err.Error())
	}
}

func (s *Server[I]) Group(path string) *Server[I] {
	if path == "" {
		return s
	}

	if g, exists := s.groups[path]; exists {
		return g
	} else {
		g = &Server[I]{
			rootPath:          s.rootPath + path,
			logger:            s.logger,
			parent:            s,
			router:            s.router,
			defaultErrorCodes: CopyMap(s.defaultErrorCodes),
			injector:          s.injector,
			groups:            map[string]*Server[I]{},
			mongoClients:      s.mongoClients,
			jsonHandler:       s.jsonHandler,
			startTime:         s.startTime,
		}

		g.middleware = s.middleware.serverMiddlewareClone(g)
		s.groups[path] = g
		return g
	}
}

func (s *Server[_]) Run(addr ...string) error {
	if s.parent != nil {
		panic("only root server can be run not its children")
	}

	if s.httpServer.Addr == "" {
		var address string
		switch len(addr) {
		case 0:
			if port, exist := os.LookupEnv("PORT"); exist {
				if !strings.Contains(port, ":") {
					address = ":" + port
				}
			} else {
				address = ":8091"
			}
		case 1:
			address = addr[0]
			if !strings.Contains(address, ":") {
				address = ":" + address
			}
		default:
			panic("one address must be specified at max")
		}

		s.httpServer.Addr = address
	}

	s.startTime = time.Now()

	s.logger.Println("server is listening on:", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server[_]) GetMongoClient(name string) mongo.Client {
	if c, exist := s.mongoClients[name]; exist {
		return c
	}

	panic("no client found with this name: " + name)
}

func (s *Server[_]) DefaultMongoClient() mongo.Client {
	return s.GetMongoClient("")
}

func (s *Server[_]) Cleanup() {
	s.mongoClients.ClearAllClients()
}

func (s *Server[_]) Router() Router {
	return s.router
}

func (s *Server[I]) Handle(method, path string, handler any, bodyInstance ...any) {
	if len(bodyInstance) == 0 {
		bodyInstance = []any{[]byte{}}
	}

	bodyType := reflect.TypeOf(bodyInstance[0])
	if h, ok := handler.(func(I) Result); ok {
		s.middleware.handler = h
		s.middleware.register(method, path, bodyType)
		return
	} else if mid, ok := handler.(*Middleware[I]); ok {
		m := s.middleware.serverMiddlewareClone()
		m.mergeMiddleware(mid)
		m.register(method, path, bodyType)
		return
	}

	panic("invalid type of handler: " + reflect.TypeOf(handler).String())
}

func (s *Server[_]) POST(path string, handler any, bodyInstance ...any) {
	s.Handle(http.MethodPost, path, handler, bodyInstance...)
}

func (s *Server[_]) GET(path string, handler any, bodyInstance ...any) {
	s.Handle(http.MethodGet, path, handler, bodyInstance...)
}

func (s *Server[_]) PUT(path string, handler any, bodyInstance ...any) {
	s.Handle(http.MethodPut, path, handler, bodyInstance...)
}

func (s *Server[_]) DELETE(path string, handler any, bodyInstance ...any) {
	s.Handle(http.MethodDelete, path, handler, bodyInstance...)
}

func (s *Server[_]) OPTIONS(path string, handler any, bodyInstance ...any) {
	s.Handle(http.MethodOptions, path, handler, bodyInstance...)
}

func (s *Server[_]) HEAD(path string, handler any, bodyInstance ...any) {
	s.Handle(http.MethodHead, path, handler, bodyInstance...)
}

func (s *Server[_]) PATCH(path string, handler any, bodyInstance ...any) {
	s.Handle(http.MethodPatch, path, handler, bodyInstance...)
}

func (s *Server[_]) CONNECT(path string, handler any, bodyInstance ...any) {
	s.Handle(http.MethodConnect, path, handler, bodyInstance...)
}

func (s *Server[_]) TRACE(path string, handler any, bodyInstance ...any) {
	s.Handle(http.MethodTrace, path, handler, bodyInstance...)
}

func (s *Server[I]) WrapHandler(priority int, wrapper Wrapper[I]) *Server[I] {
	s.middleware.WrapHandler(priority, wrapper)
	return s
}

func (s *Server[I]) FileServer(path, root string) {
	fs := http.FileServer(http.Dir(root))
	s.GET(path, func(i I) Result {
		fs.ServeHTTP(i.response(), i.request())
		return Result{terminate: true}
	}, NoBody{})
}

func (s *Server[I]) LogPrintln(v ...any) *Server[I] {
	s.logger.println(append([]any{"path={" + s.rootPath + "}"}, v...)...)
	return s
}

func (s *Server[I]) LogPrint(v ...any) *Server[I] {
	s.logger.print(append([]any{"path={" + s.rootPath + "} "}, v...)...)
	return s
}

func (s *Server[I]) LogPrintf(format string, v ...any) *Server[I] {
	s.logger.printf("path={"+s.rootPath+"} "+format, v...)
	return s
}

func (s *Server[I]) LogTrace(v ...any) *Server[I] {
	s.logger.println(append([]any{"[TRACE] path={" + s.rootPath + "}"}, v...)...)
	return s
}

func (s *Server[I]) LogDebug(v ...any) *Server[I] {
	s.logger.println(append([]any{"[DEBUG] path={" + s.rootPath + "}"}, v...)...)
	return s
}

func (s *Server[I]) LogInfo(v ...any) *Server[I] {
	s.logger.println(append([]any{"[INFO] path={" + s.rootPath + "}"}, v...)...)
	return s
}

func (s *Server[I]) LogWarn(v ...any) *Server[I] {
	s.logger.println(append([]any{"[WARN] path={" + s.rootPath + "}"}, v...)...)
	return s
}

func (s *Server[I]) LogError(v ...any) *Server[I] {
	s.logger.println(append([]any{"[ERROR] path={" + s.rootPath + "}"}, v...)...)
	return s
}

func (s *Server[I]) LogFatal(v ...any) {
	s.logger.println(append([]any{"[FATAL] path={" + s.rootPath + "}"}, v...)...)
	os.Exit(1)
}

func (s *Server[I]) LogTracef(format string, v ...any) *Server[I] {
	s.logger.printf("[TRACE] path={"+s.rootPath+"} "+format, v...)
	return s
}

func (s *Server[I]) LogDebugf(format string, v ...any) *Server[I] {
	s.logger.printf("[DEBUG] path={"+s.rootPath+"} "+format, v...)
	return s
}

func (s *Server[I]) LogInfof(format string, v ...any) *Server[I] {
	s.logger.printf("[INFO] path={"+s.rootPath+"} "+format, v...)
	return s
}

func (s *Server[I]) LogWarnf(format string, v ...any) *Server[I] {
	s.logger.printf("[WARN] path={"+s.rootPath+"} "+format, v...)
	return s
}

func (s *Server[I]) LogErrorf(format string, v ...any) *Server[I] {
	s.logger.printf("[ERROR] path={"+s.rootPath+"} "+format, v...)
	return s
}

func (s *Server[I]) LogFatalf(format string, v ...any) {
	s.logger.printf("[FATAL] path={"+s.rootPath+"} "+format, v...)
	os.Exit(1)
}

func (s *Server[I]) LoggerOutput() io.Writer {
	return s.logger.out
}

func (s *Server[I]) SetLoggerOutput(w io.Writer) {
	s.logger.SetOutput(w)
}

func (s *Server[I]) Shutdown(ctx context.Context) (err error) {
	if s.httpServer == nil {
		return nil
	}

	if ctx == nil {
		ctx = context.Background()
	}

	s.mongoClients.ClearAllClients()
	s.startTime = time.Time{}

	defer func() {
		s.LogTrace("**********", "server shutdown, err=", err, "**********")
	}()

	err = s.httpServer.Shutdown(ctx)
	return
}

func (s *Server[I]) IsListening() bool {
	return s.startTime == time.Time{}
}

func (s *Server[I]) ServeOpenAPI(path, indexFilePath, rawDocFilePath string) {
	s.GET(path, func(i I) Result {
		i.SetContentType("text/html")
		return i.ServeStaticFile(indexFilePath, http.StatusOK)
	}, noBody)

	s.GET(path+"/raw", func(i I) Result {
		contentType := "text/yaml"
		if strings.HasSuffix(rawDocFilePath, ".json") {
			contentType = "application/json"
		}

		i.SetContentType(contentType)
		return i.ServeStaticFile(rawDocFilePath, http.StatusOK)
	}, noBody)
}

func (s *Server[I]) ServeDefaultOpenAPI(path, rawDocFilePath string) {
	s.GET(path, func(i I) Result {
		i.SetContentType("text/html")
		return i.WrapOk([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1"/>
    <title>API Documentation</title>
    <link rel="preconnect" href="https://fonts.googleapis.com"/>
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin/>
    <link
            href="https://fonts.googleapis.com/css2?family=IBM+Plex+Mono:ital,wght@0,100;0,200;0,300;0,400;0,500;0,600;0,700;1,100;1,200;1,300;1,400;1,500;1,600;1,700&display=swap"
            rel="stylesheet"
    />
    <link
            href="https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:ital,wght@0,200;0,300;0,400;0,500;0,600;0,700;0,800;1,200;1,300;1,400;1,500;1,600;1,700;1,800&display=swap"
            rel="stylesheet"
    />
    <script
            type="module"
            src="https://cdn.jsdelivr.net/npm/rapidoc@9.2.0/dist/rapidoc-min.js"
            integrity="sha256-zKHbtf55GvlWwNiTYfoDmiXInEyFLp08JnhU8Gmv49k="
            crossorigin="anonymous"
    ></script>
</head>
<body>
<rapi-doc
        spec-url="`+path+`/raw"
		show-header="false"
        id="thedoc"
        theme = "dark"
        render-style="view"
        schema-style="table"
        show-method-in-nav-bar = "true"
        use-path-in-nav-bar = "true"
        show-components = "true"
        show-info = "true"
        show-header = "false"
        allow-search = "false"
        allow-advanced-search = "true"
        allow-spec-url-load="false"
        allow-spec-file-download="false"
        allow-server-selection = "true"
        allow-authentication	= "true"
        update-route="false"
        match-type="regex"
        persist-auth="true"
></rapi-doc>
</body>
</html>
`), http.StatusOK)
	}, noBody)

	s.GET(path+"/raw", func(i I) Result {
		contentType := "text/yaml"
		if strings.HasSuffix(rawDocFilePath, ".json") {
			contentType = "application/json"
		}

		i.SetContentType(contentType)
		return i.ServeStaticFile(rawDocFilePath, http.StatusOK)
	}, noBody)
}

func getDefaultErrorCodes() Map[int, string] {
	return map[int]string{
		http.StatusBadRequest:          "ERR_BAD_REQUEST",
		http.StatusInternalServerError: "ERR_INTERNAL_SERVER",
		http.StatusTooManyRequests:     "ERR_TOO_MANY_REQUESTS",
		http.StatusNotFound:            "ERR_NOT_FOUND",
		http.StatusFound:               "ERR_ALREADY_EXIST",
		http.StatusConflict:            "ERR_CONFLICT",
		http.StatusForbidden:           "ERR_FORBIDDEN",
		http.StatusNotImplemented:      "ERR_NOT_IMPLEMENTED",
		http.StatusNotAcceptable:       "ERR_NOT_NOT_ACCEPTABLE",
	}
}
