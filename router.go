// Copyright (c) 2022 Furzoom.com, All rights reserved.
// Author: Furzoom, mn@furzoom.com

package router

import "net/http"

// HandlerFunc is a function that can be registered to a route to handle HTTP
// requests. Like http.HandlerFunc, but has a third parameter for the route
// parameter.
type HandlerFunc func(http.ResponseWriter, *http.Request, map[string]string)
