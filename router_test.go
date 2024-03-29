// Copyright (c) 2024 Furzoom.com, All rights reserved.
// Author: Furzoom, mn@furzoom.com

package httprouter

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

type mockResponseWriter struct{}

func (m *mockResponseWriter) Header() (h http.Header) {
	return http.Header{}
}

func (m *mockResponseWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (m *mockResponseWriter) WriteString(s string) (n int, err error) {
	return len(s), nil
}

func (m *mockResponseWriter) WriteHeader(int) {}

func TestRouter(t *testing.T) {
	router := New()

	routed := false
	router.Handle("GET", "/user/:name", func(w http.ResponseWriter, r *http.Request, vars map[string]string) {
		routed = true
		want := map[string]string{"name": "gopher"}
		if !reflect.DeepEqual(vars, want) {
			t.Fatalf("wrong wildcard values: want %v, got %v", want, vars)
		}
	})

	w := new(mockResponseWriter)

	req, _ := http.NewRequest("GET", "/user/gopher", nil)
	router.ServeHTTP(w, req)

	if !routed {
		t.Fatalf("routing failed")
	}
}

func TestRouterAPI(t *testing.T) {
	var get, post, put, patch, delete, handlerFunc bool

	router := New()
	router.GET("/GET", func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
		get = true
	})
	router.POST("/POST", func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
		post = true
	})
	router.PUT("/PUT", func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
		put = true
	})
	router.PATCH("/PATCH", func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
		patch = true
	})
	router.DELETE("/DELETE", func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
		delete = true
	})
	router.HandlerFunc("GET", "/HandlerFunc", func(w http.ResponseWriter, r *http.Request) {
		handlerFunc = true
	})

	w := new(mockResponseWriter)

	r, _ := http.NewRequest("GET", "/GET", nil)
	router.ServeHTTP(w, r)
	if !get {
		t.Error("routing GET failed")
	}

	r, _ = http.NewRequest("POST", "/POST", nil)
	router.ServeHTTP(w, r)
	if !post {
		t.Error("routing POST failed")
	}

	r, _ = http.NewRequest("PUT", "/PUT", nil)
	router.ServeHTTP(w, r)
	if !put {
		t.Error("routing PUT failed")
	}

	r, _ = http.NewRequest("PATCH", "/PATCH", nil)
	router.ServeHTTP(w, r)
	if !patch {
		t.Error("routing PATCH failed")
	}

	r, _ = http.NewRequest("DELETE", "/DELETE", nil)
	router.ServeHTTP(w, r)
	if !delete {
		t.Error("routing DELETE failed")
	}

	r, _ = http.NewRequest("GET", "/HandlerFunc", nil)
	router.ServeHTTP(w, r)
	if !handlerFunc {
		t.Error("routing HandlerFunc failed")
	}
}

func TestRouterRoot(t *testing.T) {
	router := New()
	recv := catchPanic(func() {
		router.GET("noSlashRoot", nil)
	})

	if recv == nil {
		t.Errorf("registering path not beginning with '/' did not panic")
	}
}

func TestRouterNotFound(t *testing.T) {
	handlerFunc := func(_ http.ResponseWriter, _ *http.Request, _ map[string]string) {}

	router := New()
	router.GET("/path", handlerFunc)
	router.GET("/dir/", handlerFunc)

	testRoutes := []struct {
		route   string
		handler http.HandlerFunc
		code    int
		header  map[string]string
	}{
		{"/path/", NotFound, 301, map[string]string{"Location": "/path"}}, // TSR -/
		{"/dir", NotFound, 301, map[string]string{"Location": "/dir/"}},
		{"/../path", NotFound, 301, map[string]string{"Location": "/path"}},
		{"/nope", NotFound, 404, nil},
		{"/nope", nil, 404, nil},
	}

	for _, tr := range testRoutes {
		router.NotFound = tr.handler
		r, _ := http.NewRequest("GET", tr.route, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		failed := false
		if w.Code != tr.code {
			failed = true
		} else if w.Code == 301 {
			for k, v := range tr.header {
				if w.Header().Get(k) != v {
					failed = true
					break
				}
			}
		}
		if failed {
			t.Errorf("NotFound handling route %s failed: Code=%d, Header=%v", tr.route, w.Code, w.Header())
		}
	}
}

func TestRouterPanicHandler(t *testing.T) {
	router := New()
	panicHandled := false

	router.PanicHandler = func(w http.ResponseWriter, r *http.Request, p interface{}) {
		panicHandled = true
	}

	router.Handle("PUT", "/user/:name", func(_ http.ResponseWriter, _ *http.Request, _ map[string]string) {
		panic("oops!")
	})

	w := new(mockResponseWriter)
	req, _ := http.NewRequest("PUT", "/user/gopher", nil)

	defer func() {
		if rcv := recover(); rcv != nil {
			t.Fatal("handling panic failed")
		}
	}()

	router.ServeHTTP(w, req)

	if !panicHandled {
		t.Fatal("simulating failed")
	}
}

type mockFileSystem struct {
	opened bool
}

func (mfs *mockFileSystem) Open(name string) (http.File, error) {
	mfs.opened = true
	return nil, errors.New("this is just a mock")
}

func TestRouterFiles(t *testing.T) {
	router := New()
	mfs := &mockFileSystem{}

	recv := catchPanic(func() {
		router.ServeFiles("/noFilepath", mfs)
	})

	if recv == nil {
		t.Error("registering path not ending with '*filepath' did not panic")
	}

	router.ServeFiles("/*filepath", mfs)
	w := new(mockResponseWriter)
	r, _ := http.NewRequest("GET", "/fawicon.ico", nil)
	router.ServeHTTP(w, r)
	if !mfs.opened {
		t.Error("serving file failed")
	}
}
