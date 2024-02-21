package router

import (
	"net/http"
	"path"

	"github.com/julienschmidt/httprouter"
)

type Router struct {
	*httprouter.Router

	// prefix is the router prefix
	prefix string
}

func New() *Router {
	return &Router{
		Router: httprouter.New(),
		prefix: "/",
	}
}

func (router *Router) Group(prefix string) *Router {
	return &Router{
		Router: router.Router,
		prefix: path.Join(router.prefix, prefix),
	}
}

func (router *Router) Register(controllers ...Controller) {
	for _, controller := range controllers {
		controller.Subscribe(router)
	}
}

func (router *Router) Get(prefix string, handle httprouter.Handle) {
	router.Router.GET(path.Join(router.prefix, prefix), handle)
}

func (router *Router) Head(prefix string, handle httprouter.Handle) {
	router.Router.HEAD(path.Join(router.prefix, prefix), handle)
}

func (router *Router) Options(prefix string, handle httprouter.Handle) {
	router.Router.OPTIONS(path.Join(router.prefix, prefix), handle)
}

func (router *Router) Post(prefix string, handle httprouter.Handle) {
	router.Router.POST(path.Join(router.prefix, prefix), handle)
}

func (router *Router) Put(prefix string, handle httprouter.Handle) {
	router.Router.PUT(path.Join(router.prefix, prefix), handle)
}

func (router *Router) Patch(prefix string, handle httprouter.Handle) {
	router.Router.PATCH(path.Join(router.prefix, prefix), handle)
}

func (router *Router) Delete(prefix string, handle httprouter.Handle) {
	router.Router.DELETE(path.Join(router.prefix, prefix), handle)
}

func (router *Router) Handle(method, prefix string, handle httprouter.Handle) {
	router.Router.Handle(method, path.Join(router.prefix, prefix), handle)
}

func (router *Router) Handler(method, prefix string, handler http.Handler) {
	router.Router.Handler(method, path.Join(router.prefix, prefix), handler)
}

func (router *Router) HandlerFunc(method, prefix string, handler http.HandlerFunc) {
	router.Router.HandlerFunc(method, path.Join(router.prefix, prefix), handler)
}
