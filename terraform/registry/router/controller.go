package router

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// Endpointer is an interface with http handler methods.
type Endpointer interface {
	// Get is a shortcut for router.Handle(http.MethodGet, path, handle)
	Get(path string, handle httprouter.Handle)

	// Head is a shortcut for router.Handle(http.MethodHead, path, handle)
	Head(path string, handle httprouter.Handle)

	// Options is a shortcut for router.Handle(http.MethodOptions, path, handle)
	Options(path string, handle httprouter.Handle)

	// Post is a shortcut for router.Handle(http.MethodPost, path, handle)
	Post(path string, handle httprouter.Handle)

	// Put is a shortcut for router.Handle(http.MethodPut, path, handle)
	Put(path string, handle httprouter.Handle)

	// Patch is a shortcut for router.Handle(http.MethodPatch, path, handle)
	Patch(path string, handle httprouter.Handle)

	// Delete is a shortcut for router.Handle(http.MethodDelete, path, handle)
	Delete(path string, handle httprouter.Handle)

	// Handle registers a new request handle with the given path and method.
	Handle(method, path string, handle httprouter.Handle)

	// Handler is an adapter which allows the usage of an http.Handler as a request handle.
	Handler(method, path string, handler http.Handler)

	// HandlerFunc is an adapter which allows the usage of an http.HandlerFunc as a request handle.
	HandlerFunc(method, path string, handler http.HandlerFunc)
}

// Controller is an interface implemented by a controller.
type Controller interface {
	// Paths returns the relative Controller path.
	Prefix() string

	// Subscribe is the method called by the router to let the controller register its methods.
	Subscribe(Endpointer)
}
