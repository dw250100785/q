// Q Web framework for Go Programming Language
//
// Usage:
//
// package main
//
// import "github.com/kataras/q"
//
// func main() {
//
// q.Q{
//   Host: "mydomain.com:80",
//   Request: q.Request{
//     Entries: q.Entries{
//       // Favicon
//       q.Entry{Path: "/favicon.ico", Parser: q.Favicon{"./favicon.ico"}},
//       // Static dir
//       q.Entry{Path: "/public", Parser: q.Dir{Directory: "./assets", Gzip: true}},
//       // http://mydomain.com
//       q.Entry{Method: q.MethodGet, Path: "/", Handler: indexHandler},
//       // Routes grouping
//       q.Entry{Path: "/users", Begin: q.Handlers{auth}, Entries: q.Entries{
//         // http://mydomain.com/users
//         q.Entry{Method :q.MethodGet, Path: "/", Handler: usersHandler},
//         // http://mydomain.com/users/signout
//         q.Entry{Method: q.MethodPost, Path: "/signout",Handler: signoutPostHandler},
//         // http://mydomain.com/users/profile/1
//         q.Entry{Method: q.MethodGet, Path: "/profile/:id", Handler: userProfileHandler},
//       }},
//       // Register a subdomain
//       q.Entry{Path: "api.",Entries: q.Entries{
//           // http://api.mydomain.com/users/1
//         q.Entry{Method: q.MethodGet, Path: "/users/:id", Handler: userByIdHandler},
//       }},
//       // wildcard subdomain
//       q.Entry{Path: "*.", Entries:q.Entries{
//         //q.Entry{...},
//       }},
//     },
//   }}.Go()
// }

package q

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	// Version of the Q Web Framework
	Version = "0.0.3"
	// currently not used but keep it here for any case :), Iris' habits never ends
	banner = `    ____
  / __ \
 | |  | |
 | |  | |
 | |__| |
  \___\_\  ` + Version + `
`
)

// Q contains the http server with it's mux, websocket api, sessions manager, template engines, response engines, general configuration,
// internal Q events, the logger  and all developer's helpers functions.
//
// Basically , iteral Q is the whole web framework.
type Q struct {
	// EventEmmiter allows to .Emit custom-internal events and catch these events with .On(evt,func(data ...interface{}))
	EventEmmiter
	// start  http server

	// Host is the listening address of form: 'host:port'
	Host string // required at all cases
	// optional, normally is setted by Host if DisableServer is false, otherwise you can use it like a 'virtual' scheme, if you use nginx or caddy to serve q
	Scheme string
	//unix socket, server listening
	Mode os.FileMode
	// for manual listen tls
	CertFile, KeyFile string
	// if true then the .Go is not listens and serves, it prepares the net/http handler to be used inside a custom handler, the Host should be given in any case for smooth experience.
	DisableServer bool
	listener      *ServerListener // the only reason it's exists as field is to be able to close the http(net) listener
	// end http server

	// Charset used to render/send responses to the client
	Charset string
	// If true then templates and response engines will be rendered using Gzip compression,
	// you can still disable each render's gzip with: q.RenderOptions{"gzip": false} on the context.Render func
	Gzip bool
	// If true then you get some logs on specific cases, only for errors mostly.
	DevMode bool
	// TimeFormat default time format for any kind of datetime parsing
	TimeFormat string
	// StaticCacheDuration expiration duration for INACTIVE file handlers
	StaticCacheDuration time.Duration

	Logger     *log.Logger
	Events     Events
	Request    Request
	Templates  Templates
	templates  *templateEngines
	Responses  Responses
	responses  *responseEngines
	Session    Session
	sessions   *sessionsManager
	Websockets Websockets
	SSH        SSH
	Tester     Tester
}

// builder
func (q *Q) build() {
	q.Host = parseHost(q.Host)

	if q.TimeFormat == "" {
		q.TimeFormat = "Mon, 02 Jan 2006 15:04:05 GMT"
	}

	if q.StaticCacheDuration == 0 {
		q.StaticCacheDuration = 20 * time.Second
	}

	// logger
	if q.Logger == nil {
		q.Logger = log.New(os.Stdout, "[Q] ", log.LstdFlags)
	}

	// events
	q.EventEmmiter = &eventEmmiter{}
	q.Events.copyTo(q.EventEmmiter)
	q.Emit("build", q) // the one and only built'n event

	// templates
	q.templates = &templateEngines{
		helpers: map[string]interface{}{
			"url":     q.URL,
			"urlpath": q.Path,
		},
		reload: q.DevMode,
	}

	q.Templates.loadTo(q.templates)

	// responses
	q.responses = &responseEngines{}
	q.Responses.loadTo(q.responses)

	// websockets
	q.Websockets.copyTo(&q.Request.Entries)

	// sessions
	q.sessions = q.Session.newManager()

	// request & handler
	q.Request.build(q.Host)

	// SSH (builds the commands and starts the ssh server (if DisableServer is false) (yes, before the http server))
	q.SSH.bindTo(q)
}

func (q *Q) runServer() error {
	// start the http server
	underlineServer := &http.Server{Handler: q}
	q.listener = newServerListener(underlineServer)

	if q.CertFile != "" && q.KeyFile != "" {
		// means manualy tls
		return q.listener.ListenTLSManual(q.Host, q.CertFile, q.KeyFile)
	} else if parsePort(q.Host) == 443 {
		// means automatic tls
		// so let's start the http server first, which will redirect to https://+q.Host/$PATH, or no, I will make a functon which will return a new q
		// which will automatically redirect to this 'secure' q, like a fake 'proxy'
		return q.listener.ListenTLS(q.Host)
	} else if q.Mode > 0 {
		// means unix
		return q.listener.ListenUNIX(q.Host, q.Mode)
	}
	// just listen and serve http
	return q.listener.Listen(q.Host)
}

func (q *Q) must(err error) {
	if err != nil {
		q.Logger.Panic(err)
	}
}

// Go builds [and runs the server]
// returns itself
func (q Q) Go() *Q {
	q.build()
	if !q.DisableServer {
		//	q.must(q.runServer())
		err := q.runServer()
		if err != nil {
			if q.SSH.IsListening() && strings.Contains(err.Error(), "use of closed network connection") { // propably manually restart
				ch := make(chan os.Signal)
				<-ch
			} else {
				q.Logger.Panic(err)
			}
		}
	}
	return &q
}

func (q *Q) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	ctx := q.acquireCtx(res, req)
	q.Request.Handler(ctx)
	q.releaseCtx(ctx)
}

func (q *Q) acquireCtx(res http.ResponseWriter, req *http.Request) *Context {
	v := q.Request.contextPool.Get()
	var ctx *Context
	if v == nil {
		ctx = &Context{
			ResponseWriter: res,
			Request:        req,
			q:              q,
		}
	} else {
		ctx = v.(*Context)
		ctx.Params = ctx.Params[0:0]
		ctx.ResponseWriter = res
		ctx.Request = req
		ctx.values.Reset()
		ctx.handlers = nil
		ctx.pos = 0
		ctx.session = nil
	}

	return ctx
}

func (q *Q) releaseCtx(ctx *Context) {
	q.Request.contextPool.Put(ctx)
}

// Below are some top-level helpers functions

// TemplateString accepts a template filename, its context data and returns the result of the parsed template (string)
// if any error returns empty string
func (q *Q) TemplateString(name string, binding interface{}, options ...map[string]interface{}) string {
	res, err := q.templates.getBy(name).executeToString(name, binding, options...)
	if err != nil {
		return ""
	}
	return res
}

// ResponseString returns the string of a response engine,
// does not render it to the client
// returns empty string on error
func (q *Q) ResponseString(keyOrContentType string, obj interface{}, options ...map[string]interface{}) string {
	res, err := q.responses.getBy(keyOrContentType).toString(obj, options...)
	if err != nil {
		return ""
	}
	return res
}

// Path used to check arguments with the route's named parameters and return the correct url
// if parse failed returns empty string
func (q *Q) Path(routeName string, args ...interface{}) string {
	r := q.Request.mux.lookup(routeName)
	if r == nil {
		return ""
	}

	argsLen := len(args)

	// we have named parameters but arguments not given
	if argsLen == 0 && r.formattedParts > 0 {
		return ""
	} else if argsLen == 0 && r.formattedParts == 0 {
		// it's static then just return the path
		return r.path
	}

	// we have arguments but they are much more than the named parameters

	// 1 check if we have /*, if yes then join all arguments to one as path and pass that as parameter
	if argsLen > r.formattedParts {
		if r.path[len(r.path)-1] == matchEverythingByte {
			// we have to convert each argument to a string in this case

			argsString := make([]string, argsLen, argsLen)

			for i, v := range args {
				if s, ok := v.(string); ok {
					argsString[i] = s
				} else if num, ok := v.(int); ok {
					argsString[i] = strconv.Itoa(num)
				} else if b, ok := v.(bool); ok {
					argsString[i] = strconv.FormatBool(b)
				} else if arr, ok := v.([]string); ok {
					if len(arr) > 0 {
						argsString[i] = arr[0]
						argsString = append(argsString, arr[1:]...)
					}
				}
			}

			parameter := strings.Join(argsString, slash)
			result := fmt.Sprintf(r.formattedPath, parameter)
			return result
		}
		// 2 if !1 return false
		return ""
	}

	arguments := args[0:]

	// check for arrays
	for i, v := range arguments {
		if arr, ok := v.([]string); ok {
			if len(arr) > 0 {
				interfaceArr := make([]interface{}, len(arr))
				for j, sv := range arr {
					interfaceArr[j] = sv
				}
				arguments[i] = interfaceArr[0]
				arguments = append(arguments, interfaceArr[1:]...)
			}

		}
	}

	return fmt.Sprintf(r.formattedPath, arguments...)
}

// URL returns the subdomain+ host + Path(...optional named parameters if route is dynamic)
// returns an empty string if parse is failed
func (q *Q) URL(routeName string, args ...interface{}) (url string) {
	r := q.Request.mux.lookup(routeName)
	if r == nil {
		return
	}

	if q.Scheme == "" {
		if (q.CertFile != "" && q.KeyFile != "") || parsePort(q.Host) == 443 || q.Host == ":https" {
			q.Scheme = schemeHTTPS
		} else {
			q.Scheme = schemeHTTP
		}
	}

	scheme := q.Scheme
	host := q.Host
	arguments := args[0:]

	// join arrays as arguments
	for i, v := range arguments {
		if arr, ok := v.([]string); ok {
			if len(arr) > 0 {
				interfaceArr := make([]interface{}, len(arr))
				for j, sv := range arr {
					interfaceArr[j] = sv
				}
				arguments[i] = interfaceArr[0]
				arguments = append(arguments, interfaceArr[1:]...)
			}

		}
	}

	// if it's dynamic subdomain then the first argument is the subdomain part
	if r.subdomain == dynamicSubdomainIndicator {
		if len(arguments) == 0 { // it's a wildcard subdomain but not arguments
			return
		}

		if subdomain, ok := arguments[0].(string); ok {
			host = subdomain + "." + host
		} else {
			// it is not array because we join them before. if not pass a string then this is not a subdomain part, return empty uri
			return
		}

		arguments = arguments[1:]
	}

	if parsedPath := q.Path(routeName, arguments...); parsedPath != "" {
		url = scheme + host + parsedPath
	}

	return
}
