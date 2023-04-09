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
	if h := r.specialFixedPathLookup(method, path); h != nil {
		return h, httprouter.Params{}, true
	}

	return r.Router.Lookup(method, path)
}

func (r Router) specialFixedPathLookup(_, path string) httprouter.Handle {
	if len(r.specialFixedRoutes) != 0 {
		fixedPath := strings.ToLower(strings.Trim(path, " /\\\n\t"))

		return r.specialFixedRoutes[fixedPath]
	}

	return nil
}

func (r Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if handler := r.specialFixedPathLookup(req.Method, req.URL.Path); handler != nil {
		handler(w, req, httprouter.Params{})
		return
	}

	r.Router.ServeHTTP(w, req)
}
