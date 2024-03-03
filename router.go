// Copyright (c) 2024 Furzoom.com, All rights reserved.
// Author: Furzoom, mn@furzoom.com

// Package httprouter is a radix tree base high performance HTTP request router.
package httprouter

import (
	"errors"
	"net/http"
)

// Handle is a function that can be registered to a route to handle HTTP
// requests. Like http.HandlerFunc, but has a third parameter for the route
// parameter.
type Handle func(http.ResponseWriter, *http.Request, map[string]string)

// NotFound is the default HTTP handle func for routes that can't be matched
// with on existing route.
// NotFound tries to redirect to a canonical URL generated with CleanPath,
// Otherwise the request is delegated to http.NOTFOUND.
func NotFound(w http.ResponseWriter, req *http.Request) {
	if req.Method != "CONNECT" {
		if p := CleanPath(req.URL.Path); p != req.URL.Path && p != req.Referer() {
			http.Redirect(w, req, p, http.StatusMovedPermanently)
			return
		}
	}

	http.NotFound(w, req)
}

// Router is a http.Handler which can be used to dispatch requests to different
// handle functions via configurable routes.
type Router struct {
	node

	// Enables automatic redirection if the current route can't be match but
	// handle for the path with (without) the trailing slash exists.
	// For example if /foo/ is requested but a route only exists for /foo, the
	// client is redirected to /foo with http status code 301.
	RedirectTrailingSlash bool

	// Configurable handle func which is used when no matching route is found.
	// Default is the NotFound func of this package.
	NotFound http.HandlerFunc

	// Handler func to handle panics recovered from http handlers.
	// It should be used to generate an error page and return the http error code
	// "500 - Internal Server Error".
	// The handle can be used to keep your server from crashing because of
	// irrecoverable panics.
	PanicHandler func(http.ResponseWriter, *http.Request, interface{})
}

// Make sure the Router conforms with the http.Handler interface.
var _ http.Handler = New()

// New returns a new initialized Router.
// The router can be configured to also match the requested HTTP method or the
// requested Host.
func New() *Router {
	return &Router{
		RedirectTrailingSlash: true,
		NotFound:              NotFound,
	}
}

// GET is a shortcut for router.Handle("GET", path, handle)
func (r *Router) GET(path string, h Handle) error {
	return r.Handle("GET", path, h)
}

// POST is a shortcut for router.Handle("POST", path, handle)
func (r *Router) POST(path string, h Handle) error {
	return r.Handle("POST", path, h)
}

// PUT is a shortcut for router.Handle("PUT", path, handle)
func (r *Router) PUT(path string, h Handle) error {
	return r.Handle("PUT", path, h)
}

// DELETE is a shortcut for router.Handle("DELETE", path, handle)
func (r *Router) DELETE(path string, h Handle) error {
	return r.Handle("DELETE", path, h)
}

// Handle registers a new request handle with the given path and method.
func (r *Router) Handle(method, path string, handle Handle) error {
	if path[0] != '/' {
		return errors.New("path must begin with '/'")
	}
	return r.addRoute(method, path, handle)
}

func (r *Router) recv(w http.ResponseWriter, req *http.Request) {
	if rcv := recover(); rcv != nil {
		r.PanicHandler(w, req, rcv)
	}
}

// ServeHTTP implements the http.Handler interface.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if r.PanicHandler != nil {
		defer r.recv(w, req)
	}

	path := req.URL.Path

	if handle, vars, tsr := r.getValue(req.Method, path); handle != nil {
		handle(w, req, vars)
	} else if tsr && r.RedirectTrailingSlash && path != "/" {
		if path[len(path)-1] == '/' {
			http.Redirect(w, req, path[:len(path)-1], http.StatusMovedPermanently)
			return
		} else {
			http.Redirect(w, req, path+"/", http.StatusMovedPermanently)
			return
		}
	} else {
		// Handle 404
		r.NotFound(w, req)
	}
}
