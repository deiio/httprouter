// Copyright (c) 2024 Furzoom.com, All rights reserved.
// Author: Furzoom, mn@furzoom.com

package httprouter

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

func printChildren(n *node, prefix string) {
	fmt.Printf("%s%s[%d](%s)  %+v \r\n", prefix, n.path, len(n.children), string(n.indices), n.handle)
	for l := len(n.path); l > 0; l-- {
		prefix += " "
	}
	for _, child := range n.children {
		printChildren(child, prefix)
	}
}

// Used as a workaround since we can't compare functions ro their address
var fakeHandlerValue string

func fakeHandler(val string) Handle {
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
		handler, vars, _ := tree.getValue("GET", request.path)

		if handler == nil {
			if !request.nilHandler {
				t.Errorf("handle mismatch for route '%s': expected non-nil handle", request.path)
			}
		} else if request.nilHandler {
			t.Errorf("handle mismatch for route '%s': expected nil handle", request.route)
		} else {
			handler(nil, nil, nil)
			if fakeHandlerValue != request.route {
				t.Errorf("handle mismatch for route '%s': wrong handle (%s != %s)", request.route, fakeHandlerValue, request.route)
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
		"/ab",
		"/doc/",
		"/doc/go_faq.html",
		"/doc/go1.html",
		"/say/hi",
		"/say/hello",
	}

	for _, route := range routes {
		if err := n.addRoute("GET", route, fakeHandler(route)); err != nil {
			t.Fatalf("error inserting route '%s': %s", route, err)
		}
	}

	//printChildren(n, "")

	checkRequests(t, n, testRequests{
		{"/a", false, "/a", nil},
		{"/", true, "", nil},
		{"/hi", false, "/hi", nil},
		{"/contact", false, "/contact", nil},
		{"/co", false, "/co", nil},
		{"/con", true, "", nil},
		{"/cona", true, "", nil},
		{"/no", true, "", nil},
		{"/ab", false, "/ab", nil},
		{"/say/h", true, "", nil},
	})
}

func TestTreeWildcard(t *testing.T) {
	tree := &node{}

	routes := []string{
		"/",
		"/cmd/:tool/:sub",
		"/cmd/:tool/",
		"/src/*filepath",
		"/search/",
		"/search/:query",
		"/user_:name",
		"/user_:name/about",
		"/doc/",
		"/doc/go_faq.html",
		"/doc/go1.html",
	}

	for _, route := range routes {
		if err := tree.addRoute("GET", route, fakeHandler(route)); err != nil {
			t.Fatalf("error inserting route '%s': %s", route, err)
		}
	}

	//printChildren(tree, "")

	checkRequests(t, tree, testRequests{
		{"/", false, "/", nil},
		{"/cmd/test/", false, "/cmd/:tool/", map[string]string{"tool": "test"}},
		{"/cmd/test", true, "", map[string]string{"tool": "test"}},
		{"/cmd/test/3", false, "/cmd/:tool/:sub", map[string]string{"tool": "test", "sub": "3"}},
		{"/src/", true, "", nil},
		{"/src/some/file.png", false, "/src/*filepath", map[string]string{"filepath": "some/file.png"}},
		{"/search/", false, "/search/", nil},
		{"/search/some-图片-sh!t", false, "/search/:query", map[string]string{"query": "some-图片-sh!t"}},
		{"/search/some-图片-sh!t/", true, "", map[string]string{"query": "some-图片-sh!t"}},
		{"/user_gopher", false, "/user_:name", map[string]string{"name": "gopher"}},
		{"/user_gopher/about", false, "/user_:name/about", map[string]string{"name": "gopher"}},
	})
}

func TestWildcardConflict(t *testing.T) {
	tree := &node{}

	routes := []struct {
		path     string
		conflict bool
	}{
		{"/cmd/:tool/:sub", false},
		{"/cmd/vet", true},
		{"/src/*filepath", false},
		{"/src/*filepathx", true},
		{"/search/:query", false},
		{"/search/invalid", true},
		{"/user_:name", false},
		{"/user_x", true},
		{"/user_:name/", false},
		{"/id:id", false},
		{"/id/:id", true},
		{"/id:id/:name", false},
	}

	for _, route := range routes {
		err := tree.addRoute("GET", route.path, fakeHandler(route.path))
		if err == ErrWildcardConflict {
			if !route.conflict {
				t.Errorf("unexpected ErrWildcardConflict for route '%s'", route.path)
			}
		} else if err != nil {
			t.Errorf("unexpected error for route '%s': %v", route.path, err)
		} else if route.conflict {
			t.Errorf("no error for conflicting route '%s'", route.path)
		}
	}
}

func TestTreeChildConflict(t *testing.T) {
	tree := &node{}

	routes := []struct {
		path     string
		conflict bool
	}{
		{"/cmd/vet", false},
		{"/cmd/:tool/:sub", true},
		{"/src/AUTHOR", false},
		{"/src/*filepath", true},
		{"/user_x", false},
		{"/user_:name", true},
		{"/id/:id", false},
		{"/id:id", true},
		{"/:id", true},
		{"/*filepath", true},
	}

	for _, route := range routes {
		err := tree.addRoute("GET", route.path, fakeHandler(route.path))
		if err == ErrChildConflict {
			if !route.conflict {
				t.Errorf("unexpect ErrChildConfilct for route '%s'", route.path)
			}
		} else if err != nil {
			t.Errorf("unexpected error for route '%s': %v", route.path, err)
		} else if route.conflict {
			t.Errorf("no error for confliction route '%s'", route.path)
		}
	}
}

func TestTreeDuplicatePath(t *testing.T) {
	tree := &node{}

	routes := []string{
		"/",
		"/doc/",
		"/src/*filepath",
		"/search/:query",
		"/user_:name",
	}

	for _, route := range routes {
		if err := tree.addRoute("GET", route, fakeHandler(route)); err != nil {
			t.Fatalf("error inserting route '%s': %s", route, err)
		}

		// Add again.
		err := tree.addRoute("GET", route, nil)
		if err == ErrDuplicatePath {
			// Everything is fine.
		} else if err != nil {
			t.Errorf("unexpected err for duplicate route '%s': %v", route, err)
		} else {
			t.Fatalf("no error for duplicate route '%s'", route)
		}
	}

	checkRequests(t, tree, testRequests{
		{"/", false, "/", nil},
		{"/doc/", false, "/doc/", nil},
		{"/src/some/file.png", false, "/src/*filepath", map[string]string{"filepath": "some/file.png"}},
		{"/user_gopher", false, "/user_:name", map[string]string{"name": "gopher"}},
	})
}

func TestTreeEmptyWildcardName(t *testing.T) {
	tree := &node{}

	routes := []string{
		"/user:",
		"/user:/",
		"/cmd/:/",
		"/src/*",
	}

	for _, route := range routes {
		if err := tree.addRoute("GET", route, fakeHandler(route)); err != ErrEmptyWildcardName {
			t.Errorf("expected ErrEmptyWildcardName for route '%s'", route)
		}
	}
}

func TestTreeCatchAllConflict(t *testing.T) {
	tree := &node{}

	routes := []struct {
		path     string
		conflict bool
	}{
		{"/src/*filepath/x", true},
		{"/src2", false},
		{"/src2/*filepath/x", true},
	}

	for _, route := range routes {
		err := tree.addRoute("GET", route.path, fakeHandler(route.path))
		if err == ErrCatchAllConflict {
			if !route.conflict {
				t.Errorf("unexpected ErrCatchAllConflict for route '%s'", route.path)
			}
		} else if err != nil {
			t.Errorf("unexpected error for route '%s': %v", route.path, err)
		} else if route.conflict {
			t.Errorf("no error for conflict route '%s'", route.path)
		}
	}
}

func TestTreeTrailingSlashRedirect(t *testing.T) {
	tree := &node{}

	routes := []string{
		"/hi",
		"/b/",
		"/search/:query",
		"/cmd/:tool/",
		"/x",
		"/x/y",
		"/y/",
		"/y/z",
		"/0/:id",
		"/0/:id/1",
		"/1/:id/",
		"/1/:id/2",
		"/aa",
		"/a/",
		"/doc",
		"/doc/go_faq.html",
		"/doc/go1.html",
		"/no/a",
		"/no/b",
	}

	for _, route := range routes {
		if err := tree.addRoute("GET", route, fakeHandler(route)); err != nil {
			t.Fatalf("error inserting route '%s': %s", route, err)
		}
	}

	tsrRoutes := []string{
		"/hi/",
		"/b",
		"/search/gopher/",
		"/cmd/vet",
		"/x/",
		"/y",
		"/0/go/",
		"/1/go",
		"/a",
		"/doc/",
	}

	for _, route := range tsrRoutes {
		handler, _, tsr := tree.getValue("GET", route)
		if handler != nil {
			t.Fatalf("non-nil handle for TSR route '%s'", route)
		} else if !tsr {
			t.Errorf("expected TSR recommendation for route '%s'", route)
		}
	}

	noTsrRoutes := []string{
		"/",
		"/no",
		"/no/",
		"/_",
		"/_/",
	}

	for _, route := range noTsrRoutes {
		handler, _, tsr := tree.getValue("GET", route)
		if handler != nil {
			t.Errorf("non-nil handle for no-TSR route '%s'", route)
		} else if tsr {
			t.Errorf("expected no TSR recommendation for route '%s'", route)
		}
	}
}
