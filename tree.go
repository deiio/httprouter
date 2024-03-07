// Copyright (c) 2024 Furzoom.com, All rights reserved.
// Author: Furzoom, mn@furzoom.com

package httprouter

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

type nodeType uint8

const (
	static nodeType = iota
	param
	catchAll
)

type node struct {
	path      string
	indices   []byte
	children  []*node
	wildChild bool
	nType     nodeType
	handle    map[string]Handle
	priority  uint32
}

func (n *node) incrementChildPrio(i int) int {
	prio := n.children[i].priority + 1
	// Adjust position (move to front).
	for j := i - 1; j >= 0 && n.children[j].priority < prio; j-- {
		n.children[i], n.children[j] = n.children[j], n.children[i]
		n.indices[i], n.indices[j] = n.indices[j], n.indices[i]
		i--
	}
	return i
}

// addRoute adds a node with the given handle to the path.
// Attention! Not concurrency-safe!
func (n *node) addRoute(method, path string, handle Handle) {
	if len(n.path) == 0 && len(n.children) == 0 {
		n.insertChild(method, path, handle)
		return
	}

	for {
		// Find the longest common prefix.
		// This also implies that the common prefix contains no ':' or '*'
		// since the existing path can't contain these chars.
		i := 0
		for j := min(len(path), len(n.path)); i < j && path[i] == n.path[i]; i++ {
		}

		// Split edge
		if i < len(n.path) {
			n.children = []*node{&node{
				path:      n.path[i:],
				indices:   n.indices,
				children:  n.children,
				handle:    n.handle,
				wildChild: n.wildChild,
				priority:  n.priority,
			}}
			n.indices = []byte{n.path[i]}
			n.path = path[:i]
			n.handle = nil
			n.wildChild = false
		}

		// Make new node a child of this node
		if i < len(path) {
			path = path[i:]

			if n.wildChild {
				n.priority++

				n = n.children[0]
				// Check if the wildcard matches.
				if len(path) >= len(n.path) && n.path == path[:len(n.path)] {
					// Check for longer wildcard, e.g. :name and :namex
					if len(n.path) >= len(path) || path[len(n.path)] == '/' {
						n.addRoute(method, path, handle)
						return
					}
				}
				panic("conflict with wildcard route")
			}

			c := path[0]

			if n.nType == param && c == '/' && len(n.children) == 1 {
				n.priority++
				n = n.children[0]
				n.addRoute(method, path, handle)
				return
			}

			// Check if a child with the next path byte exists.
			for i, index := range n.indices {
				if c == index {
					i = n.incrementChildPrio(i)
					n.priority++
					n = n.children[i]
					n.addRoute(method, path, handle)
					return
				}
			}

			// Otherwise insert it.
			if c != ':' && c != '*' {
				n.indices = append(n.indices, c)
				child := &node{}
				n.children = append(n.children, child)

				n.incrementChildPrio(len(n.indices) - 1)
				n.priority++
				n = child
			}

			n.insertChild(method, path, handle)
			return
		} else if i == len(path) {
			// Make node a (in-path) leaf.
			if n.handle == nil {
				n.handle = map[string]Handle{
					method: handle,
				}
				n.priority++
			} else {
				if n.handle[method] != nil {
					panic("a Handle is already registered for this method at this path")
				}
				n.handle[method] = handle
				n.priority++
			}
		}
		return
	}
}

func (n *node) insertChild(method, path string, handle Handle) {
	var offset int

	// Find prefix until first wildcard (beginning with ':' or '*')
	for i, j := 0, len(path); i < j; i++ {
		if b := path[i]; b == ':' || b == '*' {
			// Check if this node existing children which would be
			// unreachable if we insert the wildcard here
			if len(n.children) > 0 {
				panic("wildcard route conflicts with existing children")
			}

			// Find wildcard end (either '/' or path end)
			k := i + 1
			for k < j && path[k] != '/' {
				k++
			}

			if k-i == 1 {
				panic("wildcards must be named with a non-empty name")
			}

			if b == ':' {
				// isParam.
				// Split path at the beginning of the wildcard
				if i > 0 {
					n.path = path[offset:i]
					offset = i
				}

				child := &node{
					nType: param,
				}

				n.children = []*node{child}
				n.wildChild = true
				n.priority++

				n = child

				// If the path doesn't end with the wildcard, then there will be
				// another non-wildcard subpath starting with '/'
				if k < j {
					n.path = path[offset:k]
					offset = k

					child := &node{}
					n.children = []*node{child}
					n.priority++

					n = child
				}
			} else {
				// CatchAll/
				if len(path) != k {
					panic("catchAlls are only allowed at the end of the path")
				}

				// Currently fixed width 1 for '/'.
				i--
				if path[i] != '/' {
					panic("no / before catchAll")
				}

				n.path = path[offset:i]

				// First node: catchAll node with empty path.
				child := &node{
					wildChild: true,
					nType:     catchAll,
				}
				n.children = []*node{child}
				n.indices = []byte{path[i]}
				n.priority++

				n = child

				// Second node: node holding the variable.
				child = &node{
					path: path[i:],
					handle: map[string]Handle{
						method: handle,
					},
					nType:    catchAll,
					priority: 1,
				}
				n.children = []*node{child}
				n.priority++
				return
			}
		}
	}

	// Insert remaining path part and handle to the leaf.
	n.path = path[offset:]
	n.handle = map[string]Handle{
		method: handle,
	}
	n.priority++
}

// getValue returns the handle registered with the given path(path). The values of
// wildcards are saved to a map.
// If no handle can be found, a TSR (trailing slash redirect) recommendation is
// made if a handle exists with an extra (without the) trailing slash for the
// given path.
func (n *node) getValue(method, path string) (handle Handle, vars map[string]string, tsr bool) {
	return n.getValueWithVars(method, path, nil)
}

func (n *node) getValueWithVars(method, path string, v map[string]string) (handle Handle, vars map[string]string, tsr bool) {
	vars = v
	// Walk the tree.
	for len(path) >= len(n.path) && path[:len(n.path)] == n.path {
		path = path[len(n.path):]
		if len(path) == 0 {
			// Check if this node has a handle registered  for the given node.
			if handle = n.handle[method]; handle != nil {
				return
			}

			// No handle found. Check if a handle for this path + a
			// trailing slash exist for trailing slash recommendation.
			for i, index := range n.indices {
				if index == '/' {
					n = n.children[i]
					tsr = n.path == "/" && n.handle != nil ||
						n.nType == catchAll && n.children[0].handle[method] != nil
					return
				}
			}

			// TODO: handle HTTP Error 405 - Method Not Allowed.
			// Return available methods.

			return
		}

		if n.wildChild {
			n = n.children[0]

			switch n.nType {
			case param:
				// Find param end (either '/' or path end).
				k := 0
				for k < len(path) && path[k] != '/' {
					k++
				}

				// Save param value.
				if vars == nil {
					vars = map[string]string{
						n.path[1:]: path[:k],
					}
				} else {
					vars[n.path[1:]] = path[:k]
				}

				// We need to go deeper.
				if k < len(path) {
					if len(n.children) > 0 {
						path = path[k:]
						n = n.children[0]
						continue
					} else {
						tsr = len(path) == k+1
						return
					}
				}

				if handle = n.handle[method]; handle != nil {
					return
				} else if len(n.children) == 1 {
					// No handle found. Check if a handle for this path + a
					// trailing slash exists for TSR recommendation.
					n = n.children[0]
					tsr = n.path == "/" && n.handle[method] != nil
				}

				// TODO: handle HTTP Error 405 - Method Not Allowed.
				// Return available methods.

				return
			case catchAll:

				// Catch all
				// Save CatchAll value
				if vars == nil {
					vars = map[string]string{
						n.path[2:]: path,
					}
				} else {
					vars[n.path[2:]] = path
				}

				handle = n.handle[method]
				return

			default:
				panic("unknown node type")
			}
		}

		c := path[0]

		for i, index := range n.indices {
			if c == index {
				n = n.children[i]
				return n.getValueWithVars(method, path, vars)
			}
		}

		// Nothing found. We can recommend to redirect to the save URL without
		// a trailing slash if a leaf exists for that path.
		tsr = path == "/" && n.handle[method] != nil
		return
	}

	// Nothing found. We can recommend to redirect to the same URL
	// without trailing slash if a leaf exists for that path.
	tsr = (len(path)+1 == len(n.path) && n.path[len(path)] == '/' && n.handle != nil) || (path == "/")
	return
}
