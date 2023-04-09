package flex

import (
	. "github.com/amirdlt/flex/util"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"strings"
)

type Router struct {
	apis               map[string][]string
	specialFixedRoutes Map[string, httprouter.Handle]
	*httprouter.Router
}

func (r Router) Routes() Map[string, []string] {
	return CopyMap(r.apis)
}

func (r Router) Handle(method, path string, handle httprouter.Handle) {
	r.apis[method] = append(r.apis[method], path)
	r.Router.Handle(method, path, handle)
}

func (r Router) HandleSpecialFixedPath(method, path string, handle httprouter.Handle) {
	r.apis[method] = append(r.apis[method], path)
	path = strings.ToLower(strings.Trim(path, " /\\\n\t"))
	r.specialFixedRoutes[path] = handle
}

func (r Router) Lookup(method, path string) (httprouter.Handle, httprouter.Params, bool) {
	fixedPath := strings.ToLower(strings.Trim(path, " /\\\n\t"))
	if h, exist := r.specialFixedRoutes[fixedPath]; exist {
		return h, nil, true
	}

	return r.Router.Lookup(method, path)
}

func (r Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if handler, params, exist := r.Lookup(req.Method, req.URL.Path); exist {
		handler(w, req, params)
		return
	}

	r.Router.ServeHTTP(w, req)
}
