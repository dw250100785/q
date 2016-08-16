package q

import (
	"bytes"
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/kataras/q/errors"
)

// errHandler returns na error with message: 'Passed argument is not func(*Context) neither an object which implements the q.Handler with Serve(ctx *q.Context)
// It seems to be a  +type Points to: +pointer.'
var errHandler = errors.New("Passed argument is not func(*Context) neither an object which implements the q.Handler func(ctx *Context)\n It seems to be a  %T Points to: %v.")

type (
	// Handler the main Q Handler type.
	Handler func(*Context) // we don't use an interface as I did with Iris, keep things simple as possible
	// Handlers is just a slice of Handlers []func(c *Context)
	Handlers []Handler
)

// ToHandler converts an net/http.Handler or http.HandlerFunc to a q.Handler
func ToHandler(handler interface{}) Handler {
	if httpHandler, ok := handler.(http.Handler); ok {
		return func(ctx *Context) {
			httpHandler.ServeHTTP(ctx.ResponseWriter, ctx.Request)
		}
	} else if httpHandlerFunc, ok := handler.(func(http.ResponseWriter, *http.Request)); ok {
		return func(ctx *Context) {
			httpHandlerFunc(ctx.ResponseWriter, ctx.Request)
		}
	} else if httpHandlerWithNext, ok := handler.(func(http.ResponseWriter, *http.Request, http.Handler)); ok {
		// some middleware(s) are using this form to handle the next handler, q doesn't uses this form but I hope this method will make these type of middleware(s) to work with Q
		return func(ctx *Context) {
			httpHandlerWithNext(ctx.ResponseWriter, ctx.Request, nil)
			// to http handler: (ctx.handlers[ctx.pos+1])), no need, let's put nil as the third parameter
		}
	} else if handlerFuncAlready, ok := handler.(func(*Context)); ok {
		return handlerFuncAlready
	}
	panic(errHandler.Format(handler, handler))
}

const (
	// parameterStartByte is very used on the node, it's just contains the byte for the ':' rune/char
	parameterStartByte = byte(':')
	// slashByte is just a byte of '/' rune/char
	slashByte = byte('/')
	// slash is just a string of "/"
	slash = "/"
	// matchEverythingByte is just a byte of '*" rune/char
	matchEverythingByte = byte('*')

	isStatic entryCase = iota
	isRoot
	hasParams
	matchEverything
)

type (
	// PathParameter is a struct which contains Key and Value, used for named path parameters
	PathParameter struct {
		Key   string
		Value string
	}

	// PathParameters type for a slice of PathParameter
	// Tt's a slice of PathParameter type, because it's faster than map
	PathParameters []PathParameter

	// entryCase is the type which the type of muxEntryusing in order to determinate what type (parameterized, anything, static...) is the perticular node
	entryCase uint8

	// muxEntry is the node of a tree of the routes,
	// in order to learn how this is working, google 'trie' or watch this lecture: https://www.youtube.com/watch?v=uhAUk63tLRM
	// this method is used by the BSD's kernel also
	muxEntry struct {
		part        string
		entryCase   entryCase
		hasWildNode bool
		tokens      string
		nodes       []*muxEntry
		handlers    Handlers
		precedence  uint64
		paramsLen   uint8
	}
)

var (
	errMuxEntryConflictsWildcard         = errors.New("Router: Path's part: '%s' conflicts with wildcard '%s' in the route path: '%s' !")
	errMuxEntryhandlersAlreadyExists     = errors.New("Router: handlers were already registered for the path: '%s' !")
	errMuxEntryInvalidWildcard           = errors.New("Router: More than one wildcard found in the path part: '%s' in route's path: '%s' !")
	errMuxEntryConflictsExistingWildcard = errors.New("Router: Wildcard for route path: '%s' conflicts with existing children in route path: '%s' !")
	errMuxEntryWildcardUnnamed           = errors.New("Router: Unnamed wildcard found in path: '%s' !")
	errMuxEntryWildcardInvalidPlace      = errors.New("Router: Wildcard is only allowed at the end of the path, in the route path: '%s' !")
	errMuxEntryWildcardConflictshandlers = errors.New("Router: Wildcard  conflicts with existing handlers for the route path: '%s' !")
	errMuxEntryWildcardMissingSlash      = errors.New("Router: No slash(/) were found before wildcard in the route path: '%s' !")
)

// Get returns a value from a key inside this Parameters
// If no parameter with this key given then it returns an empty string
func (params PathParameters) Get(key string) string {
	for _, p := range params {
		if p.Key == key {
			return p.Value
		}
	}
	return ""
}

// String returns a string implementation of all parameters that this PathParameters object keeps
// hasthe form of key1=value1,key2=value2...
func (params PathParameters) String() string {
	var buff bytes.Buffer
	for i := range params {
		buff.WriteString(params[i].Key)
		buff.WriteString("=")
		buff.WriteString(params[i].Value)
		if i < len(params)-1 {
			buff.WriteString(",")
		}

	}
	return buff.String()
}

// ParseParams receives a string and returns PathParameters (slice of PathParameter)
// received string must have this form:  key1=value1,key2=value2...
func ParseParams(str string) PathParameters {
	_paramsstr := strings.Split(str, ",")
	if len(_paramsstr) == 0 {
		return nil
	}

	params := make(PathParameters, 0) // PathParameters{}

	//	for i := 0; i < len(_paramsstr); i++ {
	for i := range _paramsstr {
		idxOfEq := strings.IndexRune(_paramsstr[i], '=')
		if idxOfEq == -1 {
			//error
			return nil
		}

		key := _paramsstr[i][:idxOfEq]
		val := _paramsstr[i][idxOfEq+1:]
		params = append(params, PathParameter{key, val})
	}
	return params
}

// getParamsLen returns the parameters length from a given path
func getParamsLen(path string) uint8 {
	var n uint
	for i := 0; i < len(path); i++ {
		if path[i] != ':' && path[i] != '*' { // ParameterStartByte & MatchEverythingByte
			continue
		}
		n++
	}
	if n >= 255 {
		return 255
	}
	return uint8(n)
}

// findLower returns the smaller number between a and b
func findLower(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

// add adds a muxEntry to the existing muxEntry or to the tree if no muxEntry has the prefix of
func (e *muxEntry) add(path string, handlers Handlers) error {
	fullPath := path
	e.precedence++
	numParams := getParamsLen(path)

	if len(e.part) > 0 || len(e.nodes) > 0 {
	loop:
		for {
			if numParams > e.paramsLen {
				e.paramsLen = numParams
			}

			i := 0
			max := findLower(len(path), len(e.part))
			for i < max && path[i] == e.part[i] {
				i++
			}

			if i < len(e.part) {
				node := muxEntry{
					part:        e.part[i:],
					hasWildNode: e.hasWildNode,
					tokens:      e.tokens,
					nodes:       e.nodes,
					handlers:    e.handlers,
					precedence:  e.precedence - 1,
				}

				for i := range node.nodes {
					if node.nodes[i].paramsLen > node.paramsLen {
						node.paramsLen = node.nodes[i].paramsLen
					}
				}

				e.nodes = []*muxEntry{&node}
				e.tokens = string([]byte{e.part[i]})
				e.part = path[:i]
				e.handlers = nil
				e.hasWildNode = false
			}

			if i < len(path) {
				path = path[i:]

				if e.hasWildNode {
					e = e.nodes[0]
					e.precedence++

					if numParams > e.paramsLen {
						e.paramsLen = numParams
					}
					numParams--

					if len(path) >= len(e.part) && e.part == path[:len(e.part)] {

						if len(e.part) >= len(path) || path[len(e.part)] == slashByte {
							continue loop
						}
					}
					return errMuxEntryConflictsWildcard.Format(path, e.part, fullPath)
				}

				c := path[0]

				if e.entryCase == hasParams && c == slashByte && len(e.nodes) == 1 {
					e = e.nodes[0]
					e.precedence++
					continue loop
				}
				for i := range e.tokens {
					if c == e.tokens[i] {
						i = e.precedenceTo(i)
						e = e.nodes[i]
						continue loop
					}
				}

				if c != parameterStartByte && c != matchEverythingByte {

					e.tokens += string([]byte{c})
					node := &muxEntry{
						paramsLen: numParams,
					}
					e.nodes = append(e.nodes, node)
					e.precedenceTo(len(e.tokens) - 1)
					e = node
				}
				e.addNode(numParams, path, fullPath, handlers)
				return nil

			} else if i == len(path) {
				if e.handlers != nil {
					return errMuxEntryhandlersAlreadyExists.Format(fullPath)
				}
				e.handlers = handlers
			}
			return nil
		}
	} else {
		e.addNode(numParams, path, fullPath, handlers)
		e.entryCase = isRoot
	}
	return nil
}

// addNode adds a muxEntry as children to other muxEntry
func (e *muxEntry) addNode(numParams uint8, path string, fullPath string, handlers Handlers) error {
	var offset int

	for i, max := 0, len(path); numParams > 0; i++ {
		c := path[i]
		if c != parameterStartByte && c != matchEverythingByte {
			continue
		}

		end := i + 1
		for end < max && path[end] != slashByte {
			switch path[end] {
			case parameterStartByte, matchEverythingByte:
				/*
				   panic("only one wildcard per path segment is allowed, has: '" +
				   	path[i:] + "' in path '" + fullPath + "'")
				*/
				return errMuxEntryInvalidWildcard.Format(path[i:], fullPath)
			default:
				end++
			}
		}

		if len(e.nodes) > 0 {
			return errMuxEntryConflictsExistingWildcard.Format(path[i:end], fullPath)
		}

		if end-i < 2 {
			return errMuxEntryWildcardUnnamed.Format(fullPath)
		}

		if c == parameterStartByte {

			if i > 0 {
				e.part = path[offset:i]
				offset = i
			}

			child := &muxEntry{
				entryCase: hasParams,
				paramsLen: numParams,
			}
			e.nodes = []*muxEntry{child}
			e.hasWildNode = true
			e = child
			e.precedence++
			numParams--

			if end < max {
				e.part = path[offset:end]
				offset = end

				child := &muxEntry{
					paramsLen:  numParams,
					precedence: 1,
				}
				e.nodes = []*muxEntry{child}
				e = child
			}

		} else {
			if end != max || numParams > 1 {
				return errMuxEntryWildcardInvalidPlace.Format(fullPath)
			}

			if len(e.part) > 0 && e.part[len(e.part)-1] == '/' {
				return errMuxEntryWildcardConflictshandlers.Format(fullPath)
			}

			i--
			if path[i] != slashByte {
				return errMuxEntryWildcardMissingSlash.Format(fullPath)
			}

			e.part = path[offset:i]

			child := &muxEntry{
				hasWildNode: true,
				entryCase:   matchEverything,
				paramsLen:   1,
			}
			e.nodes = []*muxEntry{child}
			e.tokens = string(path[i])
			e = child
			e.precedence++

			child = &muxEntry{
				part:       path[i:],
				entryCase:  matchEverything,
				paramsLen:  1,
				handlers:   handlers,
				precedence: 1,
			}
			e.nodes = []*muxEntry{child}

			return nil
		}
	}

	e.part = path[offset:]
	e.handlers = handlers

	return nil
}

// get is used by the Router, it finds and returns the correct muxEntry for a path
func (e *muxEntry) get(path string, _params PathParameters) (handlers Handlers, params PathParameters, mustRedirect bool) {
	params = _params
loop:
	for {
		if len(path) > len(e.part) {
			if path[:len(e.part)] == e.part {
				path = path[len(e.part):]

				if !e.hasWildNode {
					c := path[0]
					for i := range e.tokens {
						if c == e.tokens[i] {
							e = e.nodes[i]
							continue loop
						}
					}

					mustRedirect = (path == slash && e.handlers != nil)
					return
				}

				e = e.nodes[0]
				switch e.entryCase {
				case hasParams:

					end := 0
					for end < len(path) && path[end] != '/' {
						end++
					}

					if cap(params) < int(e.paramsLen) {
						params = make(PathParameters, 0, e.paramsLen)
					}
					i := len(params)
					params = params[:i+1]
					params[i].Key = e.part[1:]
					params[i].Value = path[:end]

					if end < len(path) {
						if len(e.nodes) > 0 {
							path = path[end:]
							e = e.nodes[0]
							continue loop
						}

						mustRedirect = (len(path) == end+1)
						return
					}

					if handlers = e.handlers; handlers != nil {
						return
					} else if len(e.nodes) == 1 {
						e = e.nodes[0]
						mustRedirect = (e.part == slash && e.handlers != nil)
					}

					return

				case matchEverything:
					if cap(params) < int(e.paramsLen) {
						params = make(PathParameters, 0, e.paramsLen)
					}
					i := len(params)
					params = params[:i+1]
					params[i].Key = e.part[2:]
					params[i].Value = path

					handlers = e.handlers
					return

				default:
					return
				}
			}
		} else if path == e.part {
			if handlers = e.handlers; handlers != nil {
				return
			}

			if path == slash && e.hasWildNode && e.entryCase != isRoot {
				mustRedirect = true
				return
			}

			for i := range e.tokens {
				if e.tokens[i] == slashByte {
					e = e.nodes[i]
					mustRedirect = (len(e.part) == 1 && e.handlers != nil) ||
						(e.entryCase == matchEverything && e.nodes[0].handlers != nil)
					return
				}
			}

			return
		}

		mustRedirect = (path == slash) ||
			(len(e.part) == len(path)+1 && e.part[len(path)] == slashByte &&
				path == e.part[:len(e.part)-1] && e.handlers != nil)
		return
	}
}

// precedenceTo just adds the priority of this muxEntry by an index
func (e *muxEntry) precedenceTo(index int) int {
	e.nodes[index].precedence++
	_precedence := e.nodes[index].precedence

	newindex := index
	for newindex > 0 && e.nodes[newindex-1].precedence < _precedence {
		tmpN := e.nodes[newindex-1]
		e.nodes[newindex-1] = e.nodes[newindex]
		e.nodes[newindex] = tmpN

		newindex--
	}

	if newindex != index {
		e.tokens = e.tokens[:newindex] +
			e.tokens[index:index+1] +
			e.tokens[newindex:index] + e.tokens[index+1:]
	}

	return newindex
}

//
//
//

type (
	route struct {
		// if no name given then it's the subdomain+path
		name           string
		subdomain      string
		method         string
		path           string
		handlers       Handlers
		formattedPath  string
		formattedParts int
	}

	bySubdomain []*route
)

// Sorting happens when the mux's request handler initialized
func (s bySubdomain) Len() int {
	return len(s)
}
func (s bySubdomain) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s bySubdomain) Less(i, j int) bool {
	return len(s[i].Subdomain()) > len(s[j].Subdomain())
}

func newRoute(method string, subdomain string, path string, handlers Handlers) *route {
	r := &route{name: path + subdomain, method: method, subdomain: subdomain, path: path, handlers: handlers}
	r.formatPath()
	return r
}

func (r *route) formatPath() {
	// we don't care about performance here.
	n1Len := strings.Count(r.path, ":")
	isMatchEverything := len(r.path) > 0 && r.path[len(r.path)-1] == matchEverythingByte
	if n1Len == 0 && !isMatchEverything {
		// its a static
		return
	}
	if n1Len == 0 && isMatchEverything {
		//if we have something like: /mypath/anything/* -> /mypatch/anything/%v
		r.formattedPath = r.path[0:len(r.path)-2] + "%v"
		r.formattedParts++
		return
	}

	tempPath := r.path
	splittedN1 := strings.Split(r.path, "/")

	for _, v := range splittedN1 {
		if len(v) > 0 {
			if v[0] == ':' || v[0] == matchEverythingByte {
				r.formattedParts++
				tempPath = strings.Replace(tempPath, v, "%v", -1) // n1Len, but let it we don't care about performance here.
			}
		}

	}
	r.formattedPath = tempPath
}

func (r *route) setName(newName string) {
	r.name = newName
}

// implements the CompiledEntry interface
func (r route) Name() string {
	return r.name
}

func (r route) Subdomain() string {
	return r.subdomain
}

func (r route) Method() string {
	return r.method
}

func (r route) Path() string {
	return r.path
}

func (r route) Handlers() Handlers {
	return r.handlers
}

const (
	// subdomainIndicator where './' exists in a registed path then it contains subdomain
	subdomainIndicator = "./"
	// dynamicSubdomainIndicator where a registed path starts with '*.' then it contains a dynamic subdomain, if subdomain == "*." then its dynamic
	dynamicSubdomainIndicator = "*."
)

type (
	muxTree struct {
		method string
		// subdomain is empty for default-hostname routes,
		// ex: mysubdomain.
		subdomain string
		entry     *muxEntry
		next      *muxTree
	}

	serveMux struct {
		lookups []*route
		// if any of the trees contains not empty subdomain
		hosts        bool
		allowOptions bool // if setted to true to allow all routes to be served to the client when http method 'OPTIONS', useful when user uses the Cors middleware
		tree         *muxTree
		mu           sync.Mutex
	}
)

func newServeMux() *serveMux {
	mux := &serveMux{
		lookups: make([]*route, 0),
	}

	return mux
}

func (mux *serveMux) getTree(method string, subdomain string) (tree *muxTree) {
	tree = mux.tree
	for tree != nil {
		if tree.method == method && tree.subdomain == subdomain {
			return
		}
		tree = tree.next
	}
	// tree is nil here, return that.
	return
}

func (mux *serveMux) register(method string, subdomain string, path string, handlers Handlers) *route {
	mux.mu.Lock()
	defer mux.mu.Unlock()

	if subdomain != "" {
		mux.hosts = true
	}

	// add to the lookups, it's just a collection of routes information
	lookup := newRoute(method, subdomain, path, handlers)
	mux.lookups = append(mux.lookups, lookup)
	//fmt.Printf("mux.go:761:Method: %s, Path: %s, Handlers: %d\n", lookup.method, lookup.path, len(lookup.handlers))
	return lookup
}

func (mux *serveMux) lookup(routeName string) *route {
	for i := range mux.lookups {
		if r := mux.lookups[i]; r.name == routeName {
			return r
		}
	}
	return nil
}

// build collects all routes info and adds them to the registry in order to be served from the request handler
// this happens once when server is setting the mux's handler.
func (mux *serveMux) build() {
	mux.tree = nil
	sort.Sort(bySubdomain(mux.lookups))
	for _, r := range mux.lookups {
		// add to the registry tree
		tree := mux.getTree(r.method, r.subdomain)
		if tree == nil {
			//first time we register a route to this method with this domain
			tree = &muxTree{method: r.method, subdomain: r.subdomain, entry: &muxEntry{}, next: nil}
			if mux.tree == nil {
				// it's the first entry
				mux.tree = tree
			} else {
				// find the last tree and make the .next to the tree we created before
				lastTree := mux.tree
				for lastTree != nil {
					if lastTree.next == nil {
						lastTree.next = tree
						break
					}
					lastTree = lastTree.next
				}
			}
		}
		// I decide that it's better to explicit give subdomain and a path to it than registedPath(mysubdomain./something) now its: subdomain: mysubdomain., path: /something
		// we have different tree for each of subdomains, now you can use everything you can use with the normal paths ( before you couldn't set /any/*path)
		if err := tree.entry.add(r.path, r.handlers); err != nil {
			panic(err.Error())
		}
	}
}

func (mux *serveMux) Handler(escapePath bool, correctPath bool, host string) Handler {

	mux.build()
	// optimize this once once, we could do that: context.RequestPath(mux.escapePath), but we lose some nanoseconds on if :)
	getRequestPath := func(ctx *Context) string {
		return ctx.Request.URL.EscapedPath()
	}
	if escapePath {
		getRequestPath = func(ctx *Context) string { return ctx.Request.URL.Path }
	}

	methodEqual := func(treeMethod string, reqMethod string) bool {
		return treeMethod == reqMethod
	}

	// check for cors conflicts
	if mux.allowOptions {
		methodEqual = func(treeMethod string, reqMethod string) bool {
			return treeMethod == reqMethod || reqMethod == MethodOptions
		}
	}

	// we don't need the host to have mydomain.com:80 , we want just the mydomain.com
	host = parseHost(host)
	if portIdx := strings.IndexByte(host, ':'); portIdx > 0 {
		p := host[portIdx:]
		if p == ":80" {
			host = host[0:portIdx]
		}
	}

	return func(ctx *Context) {
		tree := mux.tree
		routePath := getRequestPath(ctx)
		for tree != nil {
			if !methodEqual(tree.method, ctx.Request.Method) {
				// we break any CORS OPTIONS method
				// but for performance reasons if user wants http method OPTIONS to be served
				// then must register it with .Options(...)
				tree = tree.next
				continue
			}
			// we have at least one subdomain on the root
			if mux.hosts && tree.subdomain != "" {
				requestHost := ctx.Request.Host // on net/http that gives the full host, no just the main host(name)

				if strings.Index(tree.subdomain, dynamicSubdomainIndicator) != -1 {
				} else {
					// mux.host = mydomain.com:8080, the subdomain for example is api.,
					// so the host must be api.mydomain.com:8080
					if tree.subdomain+host != requestHost {
						// go to the next tree, we have a subdomain but it is not the correct
						tree = tree.next
						continue
					}

				} //else {
				//("it's subdomain but the request is the same as the listening addr mux.host == requestHost =>" + mux.host + "=" + requestHost + " ____ and tree's subdomain was: " + tree.subdomain)
				//		tree = tree.next
				//		continue
				//}
			}
			handlers, params, mustRedirect := tree.entry.get(routePath, ctx.Params) // pass the parameters here for 0 allocation
			if handlers != nil {
				// ok we found the correct route, serve it and exit entirely from here
				ctx.Params = params
				ctx.handlers = handlers
				//ctx.Request.Header.SetUserAgentBytes(DefaultUserAgent)
				ctx.Serve()
				return
			} else if mustRedirect && correctPath && ctx.Request.Method != MethodConnect {

				reqPath := routePath
				pathLen := len(reqPath)

				if pathLen > 1 {

					if reqPath[pathLen-1] == '/' {
						reqPath = reqPath[:pathLen-1] //remove the last /
					} else {
						//it has path prefix, it doesn't ends with / and it hasn't be found, then just add the slash
						reqPath = reqPath + "/"
					}

					ctx.Redirect(reqPath, StatusMovedPermanently) //	StatusMovedPermanently
					// RFC2616 recommends that a short note "SHOULD" be included in the
					// response because older user agents may not understand 301/307.
					// Shouldn't send the response for POST or HEAD; that leaves GET.
					if tree.method == MethodGet {
						note := "<a href=\"" + HTMLEscape(reqPath) + "\">Moved Permanently</a>.\n"
						ctx.Write([]byte(note))
					}
					return
				}
			}
			// not found
			break
		}
		ctx.EmitError(StatusNotFound)
		return
	}
}

var htmlReplacer = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	// "&#34;" is shorter than "&quot;".
	`"`, "&#34;",
	// "&#39;" is shorter than "&apos;" and apos was not in HTML until HTML5.
	"'", "&#39;",
)

// HTMLEscape returns a string which has no valid html code
func HTMLEscape(s string) string {
	return htmlReplacer.Replace(s)
}
