// Copyright (c) 2024 Furzoom.com, All rights reserved.
// Author: Furzoom, mn@furzoom.com

package httprouter

// CleanPath is the URL version of path.Clean, it returns a canonical URL path
// for p, elimination . and .. elements.
//
// The following rules are applied iteratively until no further processing can
// be done:
//  1. Replace multiple slashes with a single slash.
//  2. Eliminate each . path name element (the current directory).
//  3. Eliminate each inner .. path name element (the parent directory).
//     along with the non-.. element that precedes it.
//  4. Eliminate .. elements that begin a routed path:
//     that is, replace "/.." by "/" at the beginning of a path.
func CleanPath(p string) string {
	if p == "" {
		return "/"
	}

	n := len(p)
	var buf []byte

	// Invariants:
	//		reading from path; r is index of next byte to process.
	//		writing to buf; w is index of next byte to write.

	// Path must start with '/'
	r := 1
	w := 1

	if p[0] != '/' {
		r = 0
		buf = make([]byte, n+1)
		buf[0] = '/'
	}

	trailing := n > 2 && p[n-1] == '/'

	for r < n {
		switch {
		case p[r] == '/':
			// Empty path element, trailing slash is added after the end
			r++

		case p[r] == '.' && r+1 == n:
			trailing = true
			r++

		case p[r] == '.' && p[r+1] == '/':
			// . element
			r++

		case p[r] == '.' && p[r+1] == '.' && (r+2 == n || p[r+2] == '/'):
			// .. element: remove to last /
			r += 2

			if w > 1 {
				// can backtrack
				w--

				if buf == nil {
					for w > 1 && p[w] != '/' {
						w--
					}
				} else {
					for w > 1 && buf[w] != '/' {
						w--
					}
				}
			}

		default:
			// Real path element. Add slash if needed.
			if w > 1 {
				bufApp(&buf, p, w, '/')
				w++
			}

			// Copy elements.
			for r < n && p[r] != '/' {
				bufApp(&buf, p, w, p[r])
				w++
				r++
			}
		}
	}

	// Re-append trailing slash.
	if trailing && w > 1 {
		bufApp(&buf, p, w, '/')
		w++
	}

	if buf == nil {
		return p[:w]
	}

	return string(buf[:w])
}

// Internal helper to lazily create a buffer if necessary.
func bufApp(buf *[]byte, s string, w int, c byte) {
	if *buf == nil {
		if s[w] == c {
			return
		}
		*buf = make([]byte, len(s))
		copy(*buf, s[:w])
	}
	(*buf)[w] = c
}
