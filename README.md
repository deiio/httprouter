# HTTPRouter

> Reference from https://github.com/julienschmidt/httprouter.

HTTPRouter is a high performance HTTP request router
(also called *multiplexer* or just *mux" for short).

In contrast to the default mux of Go's net/http package, this router supports
variables in the routing pattern and matches against the request method.
It also scales better.

The router is optimized for best performance and a small memory footprint.
It scales well even with very long paths and a large number of routes.
A compressing dynamic trie (raidx tree) structure is used for efficient matching.

## Features

**Zero Garbage:** The matching and dispatching process generates zero bytes of 
garbage. In fact, the only heap allocations that are made, is by building the
map containing the variables key-value pairs. If the request path contains no
variables, not a single heap allocation is necessary.

**Best Performance:** [Benchmarks speak for themselves](https://github.com/deiio/httprouter-benchmark).
See below for technical details of the implementation.

**Variables in your routing pattern:** Stop parsing the requested URL path, just 
give the path segment a name and the router delivers the value to you. Because of
the design of the router, pattern variables are very cheap.

**Only explict matches:** With other routers / muxes, like [http.ServeMux](http://golang.org/pkg/net/http/#ServeMux), 
a requested URL path could match multiple patterns. Therefore, they have some
awkward patterns priority rules, like *longest match* for *first registered, first
matches*. By design, a request can only match exactly one or no route.

**Stop caring about trailing slashes:** Choose the URL style you like, the router
automatically redirects the client if a trailing slash is missing ro if there is one
extra. Of course, it only does so, if the new path has a handler. If you don't like
it, you can turn off this behavior.

**No more server crashes:** You can set a PanicHandler to deal with panics. The 
router then recovers and lets the PanicHandler log what happened and delivered
a nice error page.

Of course, you can also set a custom NotFound handler and serve files.

## Usage
This is just a quick introduction, view the [GoDoc](http://godoc.org/github.com/deiio/httprouter) 
for details.

Let's start with a trivial example:

```go

package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/deiio/httprouter"
)

func Index(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	fmt.Fprint(w, "Welcome!\n")
}

func Hello(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	fmt.Fprintf(w, "hello, %s!\n", vars["name"])
}

func main() {
	router := httprouter.New()
	router.GET("/index", Index)
	router.GET("/hello/:name", Hello)

	log.Fatal(http.ListenAndServe(":12345", router))
}
```

### Named parameters

As you see, `:name` is a *named parameter*.
The values are passed in map, therefore the value of `:name` is available in `vars["name"]`.

Named parameters only match a single path segment.

```
Pattern: /user/:user

 /user/gordon           match
 /user/you              match
 /user/gordon/profile   no match
 /user/                 no match
```

**Note:** Since this router has only explict matches, you can not register static routes
and parameters for the same path segment. For example, you can not register the pattern
`/user/new` and `/user/:user` at the same time.

### Catch-All routes

The seconds type are *catch-all* routes and have the form `*name`. Like the name suggest, 
the match everything. Therefore, they must always be at the **end** of the pattern:

```
Pattern: /src/*filepath

 /src/                      match
 /src/somefile.go           match
 /src/subdir/somefile.go    match
```

## How does it work?

The router relies on a tree structure which makes heavy use of *common prefixes*, it is
basically a *compare* [*prefix tree*](http://en.wikipedia.org/wiki/Trie) (or just [*Radix
tree*](http://en.wikipedia.org/Radix_tree)). Nodes with a common prefix also share a 
common parent. Here is a short example what the routing tree could look like:

```
Priority    Path                Handles
9           \                   map[GET:<1>]
3           ├s                  map[]
2           │├earch\            map[GET:<2>, POST:<3>,]
1           │├upport\           map[GET:<4>]
2           ├blog\              map[GET:<5>]
1           │    └:post         map[]
1           │         └\        map[GET:<6>]
2           ├about-us\          map[GET:<7>]
1           │        └team\     map[GET:<8>]
1           └contact\           map[GET:<9>]
└
```

Every `<num>` represents the memory address of a handler function. If you follow a path 
through the tree from the root to the leaf, you get the complete route path, e.g. `/search/`,
for which a GET- and POST-handler function are registered.

Since URL paths have a hierarchical structure and make use only of a limited set of
characters, it is very likely that there are a lot of common prefixes. This allows us
to easily reduce the routing into ever smaller problems.

Unlike hash-maps, a tree structure also allows us to use dynamic parts, e.g. the `:path`
above, which is just a placeholder for a post name, since we actually match against the
routing patterns instead of just comparing hashes. [As benchmarks show](https://github.com/deiio/httprouter-benchmark),
hash-map based routers also don't scale very well.

For even better scalability, the child nodes on each tree level are ordered by priority, where
the priority is just the number of handles registered in sub nodes (children, grandchildren, 
and so on...).

This helps in two ways:

1. Nodes which are part of the most routing paths are evaluated first. This helps to make as
many routes as possible to be reachable as fast as possible.
2. It is some sort of cost compensation. the longest reachable path (highest cost) can always
be evaluated first. The following scheme visualizes the tree structure. Nodes are evaluated
from top to bottom and from left to right.

```
├-------------
├-----------
├-------
├-----
├--
└--
```

# Where can I find middleware *X*?

This package just provides a very efficient request router with a few extra features.
The router is just a [http.Handler](http://golang.org/pkg/net/http/#Handler), you can
chain any http.Handler compatible middleware before the router, for example the
[Gorilla handlers](http://www.gorillatoolkit.org/pkg/handlers). Or you could [just write
your own](http://justinas.org/writing-http-middleware-in-go/), it's very easy.

Here is a quick example: Does your server serve multiple domains / hosts? You want to use
subdomains? Define a router per host!

```go
type HostSwitch map[string]http.Handler

func (hs HostSwitch) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if handler := hs[r.URL.Host]; handler != nil {
        handler.ServeHTTP(w, r)
		return
    }
	
	http.Error(w, "Forbidden", 403)  // Or redirect?
}

func main() {
    router := httprouter.New()
    router.GET("/", Index)
    router.GET("/hello/:name", Hello)
	
    hs := make(HostSwitch)
    hs["example.com"] = router
	
    log.Fatal(http.ListenAndServe(":12345", hs))
}
```
