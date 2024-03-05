// Copyright (c) 2024 Furzoom.com, All rights reserved.
// Author: Furzoom, mn@furzoom.com

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
