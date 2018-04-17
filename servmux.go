package coap

// ServeMux provides mappings from a common endpoint to handlers by
// request path.
type ServeMux struct {
	m map[string]muxEntry
}

type muxEntry struct {
	h       Handler
	pattern string
	network string
}

// NewServeMux creates a new ServeMux.
func NewServeMux() *ServeMux { return &ServeMux{m: make(map[string]muxEntry)} }

// Does path match pattern?
func pathMatch(pattern, path string) bool {
	if len(pattern) == 0 {
		// should not happen
		return false
	}
	n := len(pattern)
	if pattern[n-1] != '/' {
		return pattern == path
	}
	return len(path) >= n && path[0:n] == pattern
}

// Find a handler on a handler map given a path string
// Most-specific (longest) pattern wins
func (mux *ServeMux) match(path, network string) (h Handler, pattern string) {
	var n = 0
	for k, v := range mux.m {
		net := mux.m[path].network
		if !pathMatch(k, path) && net != network {
			continue
		}
		if h == nil || len(k) > n {
			n = len(k)
			h = v.h
			pattern = v.pattern
		}
	}
	return
}

func notFoundHandler(c *Conn, m Message) Message {
	if m.IsConfirmable() {
		return &DgramMessage{
			MessageBase{
				typ:  Acknowledgement,
				code: NotFound,
			},
		}
	}
	return nil
}

var _ = Handler(&ServeMux{})

// ServeCOAP handles a single COAP message.  The message arrives from
// the given listener having originated from the given UDPAddr.
//WARNING I SHOULD PROBABLY HANDLE ERRORS FOR Conn.Network()
func (mux *ServeMux) ServeCOAP(c *Conn, m Message) Message {
	n, _ := c.Network()
	h, _ := mux.match(m.PathString(), n)
	if h == nil {
		h, _ = funcHandler(notFoundHandler), ""
	}
	// TODO:  Rewrite path?
	return h.ServeCOAP(c, m)
}

// Handle configures a handler for the given path.
func (mux *ServeMux) Handle(n string, pattern string, handler Handler) {
	for pattern != "" && pattern[0] == '/' {
		pattern = pattern[1:]
	}

	if pattern == "" {
		panic("coap: invalid pattern " + pattern)
	}
	if handler == nil {
		panic("coap: nil handler")
	}
	if _, ok := mux.m[pattern]; ok {
		if mux.m[pattern].network == n {
			panic("coap: multiple registration for " + pattern + " on transport: " + n)
		}
	}
	mux.m[pattern] = muxEntry{h: handler, pattern: pattern, network: n}
}

// HandleFunc configures a handler for the given path.
func (mux *ServeMux) HandleFunc(network, pattern string,
	f func(c *Conn, m Message) Message) {
	mux.Handle(network, pattern, FuncHandler(f))
}
