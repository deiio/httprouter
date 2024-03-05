// Copyright (c) 2024 Furzoom.com, All rights reserved.
// Author: Furzoom, mn@furzoom.com

package main

import (
	"github.com/deiio/httprouter"
	"log"
	"net/http"
)

func main() {
	router := httprouter.New()
	router.ServeFiles("/*filepath", http.Dir("/Users/bytedance/workspace/projects/furzoom/httprouter"))

	log.Fatal(http.ListenAndServe(":12345", router))
}
