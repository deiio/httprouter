// Copyright (c) 2022 Furzoom.com, All rights reserved.
// Author: Furzoom, mn@furzoom.com

package router

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

func printChildren(n *node, prefix string) {
	fmt.Printf("%s%s[%d](%s)  %+v \r\n", prefix, n.key, len(n.children), string(n.indices), n.value)
	for l := len(n.key); l > 0; l-- {
		prefix += " "
	}
	for _, child := range n.children {
		printChildren(child, prefix)
	}
}

// Used as a workaround since we can't compare functions ro their address
var fakeHandlerValue string

func fakeHandler(val string) HandlerFunc {
	return func(http.ResponseWriter, *http.Request, map[string]string) {
		fakeHandlerValue = val
	}
}

type testRequests []struct {
	path       string
	nilHandler bool
	route      string
	vars       map[string]string
}

func checkRequests(t *testing.T, tree *node, requests testRequests) {
	for _, request := range requests {
		handler, vars, _ := tree.getValue(request.path)

		if handler == nil {
			if !request.nilHandler {
				t.Errorf("handler mismatch for route '%s': expected non-nil handler", request.path)
			}
		} else if request.nilHandler {
			t.Errorf("handler mismatch for route '%s': expected nil handler", request.route)
		} else {
			handler(nil, nil, nil)
			if fakeHandlerValue != request.route {
				t.Errorf("handler mismatch for route '%s': wrong handler (%s != %s)", request.route, fakeHandlerValue, request.route)
			}
		}

		if !reflect.DeepEqual(vars, request.vars) {
			t.Errorf("vars mismatch for route %s", request.path)
		}
	}
}

func TestTreeAddAndGet(t *testing.T) {
	n := &node{}
	routes := []string{
		"/hi",
		"/contact",
		"/co",
		"/c",
		"/a",
		"/ab/",
		"/doc/",
		"/doc/go_faq.html",
		"/doc/go1.html",
		"/say/hi",
		"/say/hello",
	}

	for _, route := range routes {
		if err := n.addRoute(route, fakeHandler(route)); err != nil {
			t.Fatalf("error inserting route '%s': %s", route, err)
		}
	}

	printChildren(n, "")

	checkRequests(t, n, testRequests{
		{"/a", false, "/a", nil},
		{"/", true, "", nil},
		{"/hi", false, "/hi", nil},
		{"/contact", true, "/contact", nil},
		{"/co", false, "/co", nil},
		{"/con", true, "", nil},
		{"/cona", true, "", nil},
		{"/no", true, "", nil},
		{"/ab", false, "/ab", nil},
		{"/say/h", true, "", nil},
	})
}
