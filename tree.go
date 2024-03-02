// Copyright (c) 2022 Furzoom.com, All rights reserved.
// Author: Furzoom, mn@furzoom.com

package router

import "errors"

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

var (
	ErrDuplicatePath     = errors.New("duplicate Path")
	ErrEmptyWildcardName = errors.New("wildcards must be named with a non-empty name")
	ErrCatchAllConflict  = errors.New("CatchAlls are only allowed at the end of the path")
	ErrChildConflict     = errors.New("can't insert a wildcard route because this path has existing children")
	ErrWildcardConflict  = errors.New("conflict with wildcard route")
)

type node struct {
	key      string
	indices  []byte
	children []*node
	value    HandlerFunc
	//wildChild  bool
	//isParam    bool
	//isCatchAll bool
}

// addRoute adds a leaf with the given value to the path determined by the given key.
// Attention! Not concurrency-safe!
func (n *node) addRoute(key string, value HandlerFunc) error {
	if len(n.key) == 0 {
		return n.insertRoute(key, value)
	}
OUTER:
	for {
		i := 0
		for j := min(len(key), len(n.key)); i < j && key[i] == n.key[i]; i++ {
		}

		if i < len(n.key) {
			n.children = []*node{&node{
				key:      n.key[i:],
				indices:  n.indices,
				children: n.children,
				value:    n.value,
			}}
			n.indices = []byte{n.key[i]}
			n.key = key[:i]
			n.value = nil
		}

		if i < len(key) {
			key = key[i:]
			c := key[0]

			for i, index := range n.indices {
				if c == index {
					n = n.children[i]
					continue OUTER
				}
			}

			n.indices = append(n.indices, c)
			child := &node{}
			n.children = append(n.children, child)

			n = child
			return n.insertRoute(key, value)
		} else if i == len(key) {
			if n.value != nil {
				return ErrDuplicatePath
			}
			n.value = value
		}
		return nil
	}
}

func (n *node) insertRoute(key string, value HandlerFunc) error {
	n.key = key
	n.value = value
	return nil
}

func (n *node) getValue(key string) (value HandlerFunc, vars map[string]string, tsr bool) {
	return
}
