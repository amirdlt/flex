package flex

import (
	"github.com/amirdlt/flex/util"
	"github.com/julienschmidt/httprouter"
)

type Router struct {
	apis map[string][]string
	*httprouter.Router
}

func (r Router) Routes() util.Map[string, []string] {
	return util.CopyMap(r.apis)
}

func (r Router) Handle(method, path string, handle httprouter.Handle) {
	r.apis[method] = append(r.apis[method], path)
	r.Router.Handle(method, path, handle)
}
