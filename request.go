package q

import (
	"fmt"
	"io"
	"net/http/pprof"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Entries TODO:
type Entries []Entry // we do like this because we want to be able to use other types except the Entry {}, like StaticEntry{} and so on, without need to call a function
// it's lowercase but user can be easly understand what to use ('Entry')

// Add appends to the entries, it doesn't register the entry to the mux, this function should be called before .Go,
// it's used internally to set different various of entries, such as the Websocket client source
func (entries *Entries) Add(entr ...Entry) {
	pEntries := *entries
	*entries = append(pEntries, entr...)
}

// ByName return an entry by it's name
//
// Note: It doesn't return the same thing as Request.GetEntr which returns CompiledEntry or Request.mux.lookup which returns *route, this is for unregistered entries, before .registerEntry on mux.
// used internaly by Q.
func (entries *Entries) ByName(entryName string) *Entry {
	pEntries := *entries
	for i := range pEntries {
		if pEntries[i].Name == entryName {
			return &pEntries[i]
		}
	}
	return nil
}

// EntryParser allows to pass a Entry-like struct-iteral into the Request.Entries
// useful when needed to set custom Entry's fields on a user-defined Entry
// see fs.go for more
type EntryParser interface {
	ParseEntry(Entry) Entry
}

// Entry the entry which contains the routes infromation to be registered
type Entry struct {
	// Name, optional, the name of the entry, used for .URL/.Path/context.RedirectTo/ inside templates: {{ url }}, {{ urlpath}}
	Name string
	// The http method
	Method string
	Head   bool // set to true if you want this entry to be valid on HEAD http method also, defaults to false, useful when the entry serves static files
	// The request path
	Path string // if empty then this will be available using all http methods
	// Middleware before Handler
	Begin Handlers
	// Middleware after the Handler
	Done Handlers
	// The main entry's Handler
	Handler Handler
	// Any children entries, use it to group routes with the same prefix and middleware
	Entries Entries
	// Parser is the method which can be used to change the fields of a user-defined Entry
	// look fs.go for more
	Parser EntryParser
	// subdomains is a compiled private field, used to map the entry with specific subdomains, inside mux
	subdomain string
}

// doParse returns the converted Entry from the Parser or itself if Parser is nil
func (e Entry) doParse() Entry {
	if e.Parser != nil {
		return e.Parser.ParseEntry(e)
	}
	return e
}

// Errors is used inside Request.Errors fields
// catch custom http status code errors via Handlers
// it's map[int]Handler
type Errors map[int]Handler

// Request the iteral which keeps all router's configuration, register routes/entries, set middleware with Begin & Done & set custom http errors with the Errors field:
type Request struct {
	DisablePathCorrection bool
	DisablePathEscape     bool
	// AllowMethodOptions if setted to true to allow all routes to be served to the client when http method 'OPTIONS', useful when user uses the Cors middleware
	// defaults to false
	AllowMethodOptions bool
	// Custom http errors handlers
	Errors Errors
	// Middleware before any entry's main handler
	Begin Handlers
	// Middleware after any entry's main handler
	Done Handlers
	// if !=nil then this is used for the main router
	// if !=nil then the .Entry/.Entries,context.RedirectTo & all q's static handler will not work, you have to build them by yourself.
	Handler Handler
	// The Routes
	Entries Entries
	//
	mux         *serveMux // can be nil if Handler is setted by user
	contextPool sync.Pool
}

func (req *Request) build(host string) {
	if req.Errors == nil {
		req.Errors = make(map[int]Handler, 0)
	}

	for _, statusCode := range statusCodesAll {
		if req.Errors[statusCode] == nil && statusCode != StatusOK && statusCode != StatusPermanentRedirect && statusCode != StatusTemporaryRedirect && statusCode != StatusAccepted {
			// register the default error handler if not registed by the user
			func(statusCode int) {
				errHandler := func(ctx *Context) {
					ctx.SetStatusCode(statusCode) // use custom-func for set the status code in order to store it
					io.WriteString(ctx.ResponseWriter, statusText[statusCode])
					ctx.StopExecution() // don't run next middleware, if the user wants to change this behavior he/she can just add an error handler to the specific http status code
				}
				req.Errors[statusCode] = errHandler
			}(statusCode)
		}
	}

	if req.Handler == nil {
		// make use of the q's default mux
		req.mux = newServeMux()
		for i := range req.Entries {
			req.registerEntry(req.Entries[i])
		}
		req.mux.allowOptions = req.AllowMethodOptions
		// we set & build the handler, to the buildHandler whcich is called at the end of build all, because the mux is useful on other Q's internally components, like websocket
		req.Handler = req.mux.Handler(!req.DisablePathEscape, !req.DisablePathCorrection, host)
	}
}

// no need to change anything inside user-defined entries, we change them and register the entry immediatly
func (req *Request) registerEntry(e Entry) {
	entry := e.doParse()
	if len(entry.Entries) > 0 {
		// subdomain or party
		for i := range entry.Entries {
			r := entry.Entries[i].doParse()

			subdomain := ""
			// check it's path
			// only subdomains ends with '.'
			if entry.Path[len(entry.Path)-1] == '.' {
				subdomain = entry.Path[0:strings.LastIndexByte(entry.Path, '.')+1] + entry.subdomain // +1 because we need the, +entry.subdomain to be able for children unlimited subdomains.
			}

			r.Method = parseMethod(r.Method)
			r.subdomain = subdomain
			// set the begin,done, first parent's after children's
			r.Begin = append(entry.Begin, r.Begin...)
			r.Done = append(entry.Done, r.Done...)
			if subdomain == "" {
				// it's party, then set the full path also
				r.Path = entry.Path + r.Path // do not use the filepath package here, because if the dev has CorrectPath false this will break the last slash
			}
			req.registerEntry(r)
		}
		return
	}

	handlersLen := len(req.Begin) + len(entry.Begin) + len(entry.Done) + len(req.Done)
	if entry.Handler != nil {
		handlersLen++
	}
	handlers := make(Handlers, handlersLen, handlersLen)
	j := 0

	for p := range req.Begin {
		handlers[j] = req.Begin[p]
		j++
	}

	for p := range entry.Begin {
		handlers[j] = entry.Begin[p]
		j++
	}
	if entry.Handler != nil {
		handlers[j] = entry.Handler
		j++
	}

	for p := range entry.Done {
		handlers[j] = entry.Done[p]
		j++
	}

	for p := range req.Done {
		handlers[j] = req.Done[p]
		j++
	}
	method := parseMethod(entry.Method)
	path := entry.Path
	if len(path) == 0 {
		path = "/"
	}
	if method != "" {
		req.mux.register(method, entry.subdomain, path, handlers)
		if entry.Head {
			req.mux.register(MethodHead, entry.subdomain, path, handlers)
		}
	} else {
		// register to all http methods
		for _, m := range MethodsAll {
			req.mux.register(m, entry.subdomain, path, handlers)
		}
	}
}

// CompiledEntry is the parsed entry, contains the full path (if children of party or subdomain's)
// available,only, after the server has been fully builded and is running
type CompiledEntry interface {
	Name() string
	Subdomain() string
	Method() string
	Path() string
	Handlers() Handlers
}

// GetEntry returns the (registered) CompiledEntry found by its Name
func (req *Request) GetEntry(entryName string) CompiledEntry {
	if r := req.mux.lookup(entryName); r != nil {
		return r
	}
	return nil
}

// GetEntries returns all (registered) CompiledEntries
func (req *Request) GetEntries() (entries []CompiledEntry) {
	for i := range req.mux.lookups {
		entries = append(entries, req.mux.lookups[i])
	}
	return
}

// Built'n Request's EntryParsers

// Profile is the pprof profile entry
type Profile struct{}

// ParseEntry returns the pprof entry, implements the EntryParser
func (s Profile) ParseEntry(e Entry) Entry {
	if e.Path == "" {
		e.Path = "/pprof"
	}

	e.Path += "/*action"

	indexHandler := ToHandler(pprof.Index)
	cmdlineHandler := ToHandler(pprof.Cmdline)
	profileHandler := ToHandler(pprof.Profile)
	symbolHandler := ToHandler(pprof.Symbol)
	goroutineHandler := ToHandler(pprof.Handler("goroutine"))
	heapHandler := ToHandler(pprof.Handler("heap"))
	threadcreateHandler := ToHandler(pprof.Handler("threadcreate"))
	debugBlockHandler := ToHandler(pprof.Handler("block"))

	h := func(ctx *Context) {
		ctx.SetContentType(contentHTML + "; charset=" + ctx.q.Charset)
		action := ctx.Param("action")
		if len(action) > 1 {
			if strings.Contains(action, "cmdline") {
				cmdlineHandler((ctx))
			} else if strings.Contains(action, "profile") {
				profileHandler(ctx)
			} else if strings.Contains(action, "symbol") {
				symbolHandler(ctx)
			} else if strings.Contains(action, "goroutine") {
				goroutineHandler(ctx)
			} else if strings.Contains(action, "heap") {
				heapHandler(ctx)
			} else if strings.Contains(action, "threadcreate") {
				threadcreateHandler(ctx)
			} else if strings.Contains(action, "debug/block") {
				debugBlockHandler(ctx)
			}
		} else {
			indexHandler(ctx)
		}
	}

	e.Handler = h
	return e
}

// NewLoggerHandler is a basic Logger middleware/Handler (not an Entry Parser)
func NewLoggerHandler(writer io.Writer, calculateLatency ...bool) Handler {
	shouldNext := false
	if len(calculateLatency) > 0 {
		shouldNext = calculateLatency[0]
	}
	return func(ctx *Context) {
		var date, status, ip, method, path string
		var latency time.Duration
		var startTime, endTime time.Time
		path = ctx.Path()
		method = ctx.Request.Method

		startTime = time.Now()
		if shouldNext {
			ctx.ForceNext()
		}

		endTime = time.Now()
		latency = endTime.Sub(startTime)
		date = endTime.Format("01/02 - 15:04:05")

		status = strconv.Itoa(ctx.StatusCode()) // if ctx.SetStatusCode doesn't call itself then this will be always 200, default error handlers uses that so it should be ok
		ip = ctx.RemoteAddr()

		//finally print the logs to the ssh
		writer.Write([]byte(fmt.Sprintf("%s %v %4v %s %s %s \n", date, status, latency, ip, method, path)))
	}
}
