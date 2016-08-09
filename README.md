
![Q powered by golang](https://github.com/q-contrib/logo/raw/master/q_logo_github_1_900_200.png)

<a href="https://travis-ci.org/kataras/q"><img src="https://img.shields.io/travis/kataras/q.svg?style=flat-square" alt="Build Status"></a>
<a href="https://github.com/kataras/q/blob/master/LICENSE"><img src="https://img.shields.io/badge/%20license-MIT%20%20License%20-E91E63.svg?style=flat-square" alt="License"></a>
<a href="https://github.com/kataras/q/releases"><img src="https://img.shields.io/badge/%20release%20-%20v0.0.1-blue.svg?style=flat-square" alt="Releases"></a>
<a href="#docs"><img src="https://img.shields.io/badge/%20docs-reference-5272B4.svg?style=flat-square" alt="Read me docs"></a>
<a href="https://kataras.rocket.chat/channel/q"><img src="https://img.shields.io/badge/%20community-chat-00BCD4.svg?style=flat-square" alt="Build Status"></a>
<a href="https://golang.org"><img src="https://img.shields.io/badge/powered_by-Go-3362c2.svg?style=flat-square" alt="Built with GoLang"></a>
<a href="#"><img src="https://img.shields.io/badge/platform-Any--OS-yellow.svg?style=flat-square" alt="Platforms"></a>

Fast & unique backend web framework for Go.  Easy to [learn](#docs),  while it's highly customizable.<br/>
Ideally suited for both experienced and novice Developers.


Quick view
-----------

```go
q.Q{
  Host: "mydomain.com:80",
  Request: q.Request{
    Entries: q.Entries{
      // Favicon
      q.Entry{Path: "/favicon.ico", Parser: q.Favicon{"./favicon.ico"}},
      // Static dir
      q.Entry{Path: "/public", Parser: q.Dir{Directory: "./assets", Gzip: true}},
      // http://mydomain.com
      q.Entry{Method: q.MethodGet, Path: "/", Handler: indexHandler},
      // Routes grouping
      q.Entry{Path: "/users", Begin: q.Handlers{auth}, Entries: q.Entries{
        // http://mydomain.com/users
        q.Entry{Method :q.MethodGet, Path: "/", Handler: usersHandler},
        // http://mydomain.com/users/signout
        q.Entry{Method: q.MethodPost, Path: "/signout",Handler: signoutPostHandler},
        // http://mydomain.com/users/profile/1
        q.Entry{Method: q.MethodGet, Path: "/profile/:id", Handler: userProfileHandler},
      }},
      // Register a subdomain  
      q.Entry{Path: "api.",Entries: q.Entries{
          // http://api.mydomain.com/users/1
        q.Entry{Method: q.MethodGet, Path: "/users/:id", Handler: userByIdHandler},
      }},
      // wildcard subdomain
      q.Entry{Path: "*.", Entries:q.Entries{
        //q.Entry{...},
      }},
    },
  }}.Go()
```

Features
------------
- Focus on simplicity and performance, one of the fastest net/http frameworks.
- Letsencrypt integration
- 100% compatible with standard net/http and all third-party middleware that are already spread   
- Robust routing, static, wildcard subdomains and routes.
- Websocket API, Sessions support out of the box
- View system supporting [6+](#templates) template engines
- Highly scalable response engines
- and many other surprises

<img src="https://raw.githubusercontent.com/iris-contrib/website/gh-pages/assets/arrowdown.png" width="72"/>


| Name        | Description           
| ------------------|:---------------------:|
| [JSON ](https://github.com/kataras/q/tree/master/response/json)      | JSON Response Engine (Default)
| [JSONP ](https://github.com/kataras/q/tree/master/response/jsonp)      | JSONP Response Engine (Default)
| [XML ](https://github.com/kataras/q/tree/master/response/xml)      | XML Response Engine (Default)
| [Markdown ](https://github.com/kataras/q/tree/master/response/markdown)      | Markdown Response Engine (Default)
| [Text](https://github.com/kataras/q/tree/master/response/text)      | Text Response Engine (Default)
| [Binary Data ](https://github.com/kataras/q/tree/master/response/data)      | Binary Data Response Engine (Default)
| [HTML/Default Engine ](https://github.com/kataras/q/tree/master/template/html)      | HTML Template Engine (Default)
| [Django Engine ](https://github.com/kataras/q/tree/master/template/django)      | Django Template Engine
| [Pug/Jade Engine ](https://github.com/kataras/q/tree/master/template/pug)      | Pug Template Engine
| [Handlebars Engine ](https://github.com/kataras/q/tree/master/template/handlebars)      | Handlebars Template Engine
| [Amber Engine ](https://github.com/kataras/q/tree/master/template/amber)      | Amber Template Engine
| [Markdown Engine ](https://github.com/kataras/q/tree/master/template/markdown)      | Markdown Template Engine
| [Basicauth Middleware ](https://github.com/q-contrib/middleware/tree/master/basicauth)      | HTTP Basic authentication
| [Cors Middleware ](https://github.com/q-contrib/middleware/tree/master/cors)      | Cross Origin Resource Sharing W3 specification
| [Secure Middleware ](https://github.com/q-contrib/middleware/tree/master/secure) |  Facilitates some quick security wins
| [I18n Middleware ](https://github.com/q-contrib/middleware/tree/master/i18n)      | Simple internationalization
| [Recovery Middleware ](https://github.com/q-contrib/middleware/tree/master/recovery) | Safety recover the station from panic
| [Logger Middleware ](https://github.com/q-contrib/middleware/tree/master/logger)      | Logs every request


##### Response & Template engines notes

These are the built'n engines, you can still create your own and share the repository with us via [chat][Chat].

##### Middleware notes

The middleware list shows only these are converted to work directly with q.Handler, but all net/http middleware are compatible, using the q.ToHandler function.
such as the [JWT Middleware](https://github.com/auth0/go-jwt-middleware) , can be converted using: `q.ToHandler(jwtMiddleware.CheckJWT)`

Installation
------------
The only requirement is the [Go Programming Language](https://golang.org/dl), at least v1.6

```bash
$ go get -u github.com/kataras/q
```

> If you have installation issues or you are connected to the Internet through China please, look below

- If you are connected to the Internet through **China**, you may experience installation issues. **Follow the below steps**:

1.  https://github.com/northbright/Notes/blob/master/Golang/china/get-golang-packages-on-golang-org-in-china.md

2. `$ go get github.com/kataras/q ` **without -u**

- If you have any problems installing Q, just delete the directory `$GOPATH/src/github.com/kataras/q` , open your shell and run `go get -u github.com/kataras/q` .



Docs
------------

### Handlers

Handlers are used to handle the client's request.

```go
type Handler(*Context)
```

```go
func myHandler(ctx *q.Context) {
  ctx.WriteString("Hello from %s", ctx.Path())
}

q.Entry{Method: q.MethodGet, Path: "/", Handler: myHandler} // don't bother, will be explained below, you care about the Handler field now.
```

Handlers is the type of middleware executed before or after the main entry's Handler, using the `Begin` and `Done` Entry's fields.

**Entry.Begin** and **Entry.Done** receives `Handlers`, a slice of `[]Handler`, if only one handler is needed then use the `q.Handlers{}` and pass the handler inside.

```go
type Handlers []Handler
```

```go
func myAuthMiddleware(ctx *q.Context) {
  // authentication/verification logic here
  authenticated := true
  // if not authenticated do not continue to the next hadlers
  if !authenticated {
    ctx.Cancel()
  }
}

func profileHandler(ctx *q.Context) {
  // get the value of the :id param from the Path
  profileID := ctx.ParamInt("id")
  // render a JSON response message to the browser, {"id": 1}
  ctx.JSON(q.Map{"id", profileID})
}

// Per-entry middleware Begin, Done
q.Entry{Method: q.MethodGet, Path: "/profile/:id", Begin: q.Handlers{myAuthMiddleware}, Handler: profileHandler}

// Per group of routes [or subdomain if path was "users."]
q.Entry{Path: "/users", Begin: q.Handlers{myAuthMiddleware}, Entries: q.Entries{
  q.Entry{Method :q.MethodGet, Path: "/profile/:id", Handler: profileHandler},
  //...
}}


// Global middleware (passed automatically to all Entries, including subdomains, group of routes...)

Request: q.Request {
  Begin: q.Handlers{myAuthMiddleware},
  Done : q.Handlers{},
  Entries: q.Entries{
    //...
  }
}
```



#### http.Handler

Say for example that we have found a net/http middleware that we want to use, type of:

```go
AnyNetHTTPHandlerFunc func(w http.ResponseWriter, req *http.Request)

type AnyNetHTTPHandler struct {}

func (a AnyNetHTTPHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
  //...
}
```

Q web framework can convert any `http.Handler` & `http.HandlerFunc` to `q.Handler` using the `q.ToHandler` func.

```go
q.Entry{Method: q.MethodGet, Path: "/path", Begin:q.Handlers{q.ToHandler(AnyNetHTTPHandlerFunc), q.ToHandler(AnyNetHTTPHandler)}, Handler:...}
```


### Transport Layer Security (TLS)

Q web framework makes easy to convert your web app to a secure website, scheme  `https://`.

```go
q.Q{Host: "mydomain.com:443"}.Go()
```

Yes, so simple, just pass the port `443` after your domain, and you will get [Letsencrypt.org](https://letsencrypt.org) integration, provides automatically SSL certification.

At the other hand if you have certification and key file, you can disable the letsencrypt integration

```go
q.Q{Host: "mydomain.com:443", CertFile: "fileCert.cert", KeyFile: "fileKey.key"}.Go()
```


Make use of [Proxy func](https://github.com/kataras/q/blob/master/proxy.go) to redirect all http://$PATH requests to https://$PATH.

```go
// this will redirect http://mydomain.com/$REQUESTED_PATH to the https://mydomain.com/$REQUESTED_PATH
go q.Proxy("mydomain.com:80", "https://mydomain.com")
```

## Events (custom app's internal)

Events can be used to communicate with your app's lifecycle and actions, you can register any custom event and listeners, Q provides only one built'n event which is the `build`, fired before building and running.

Here's how a listener can be registered

```go
q.Q{Host: "mydomain.com:80",
// Custom events, use it whenever you want in your app's lifecycle
Events: q.Events{
  // build is the one and only built'n event, you can setup your own to work with.
  // the build sends one parameter which is the current Q instance
  "build": q.EventListeners{beforeBuildEvent1},
  "mycustom": q.EventListeners{myCustomListener, myCustomSecondListener},
},
/* other fields...*/
}.Go()

// data ca be any type of messages that the q.Emit("event", anymessage{},here{},"message"), in this case the built'n event 'build' sends the current Q instance.
func beforeBuildEvent1(data ...interface{}) {
	myQ := data[0].(*q.Q)
	myQ.Logger.Println("Right before building the server, you can still use the myQ.Request.Entries.Add(Entry) to add an entry from a 'plugin' or something like this! ")
	// the build sends one parameter which is the current Q instance, let's grab it

	myQ.Request.Entries.Add(q.Entry{Method: "GET", Path: "/builded", Handler: func(ctx *q.Context) {
		ctx.HTML("<h1>Hello</h1>This entry/route builded just before the server ran,<br/><b>you can use that method to inject any other runtime-routes/entries you want to register to the Q web framework.</b>")
	}})
	// go to http://mydomain.com/builded or http://127.0.0.1/builded or whatever you setted as 'Host' field in the Q instance creation and
  // you will see that the entry is served like others
}
```

Events can be fired and call their registered listeners using the `$qinstance.Emit("eventname", OptionalDataToSend{})`

```go
var myQ *q.Q

// any code here...

myQ = q.Q{Host: "mydomain.com:80",
// Custom events, use it whenever you want in your app's lifecycle
Events: q.Events{
  "mycustomEvent": q.EventListeners{myCustomListener, myCustomSecondListener},
},
/* other fields...*/
}.Go()

// any code here...

func myCustomListener(data ...interface{}) {
  myQ.Logger.Println(" [1] mycustomEvent has been fired!")
}

func myCustomSecondListener(data ...interface{}) {
  myQ.Logger.Println(" [2] mycustomEvent has been fired!")
}

// any code here...

// fire using .Emit
myQ.Emit("mycustomEvent")
```

An event can be fired from a `Handler` also

```go
func myHandler(ctx *Context){
  ctx.Q().Emit("mycustomEvent")
}

```


## Request

We're already covered most of the `Request` field on the Handlers section, in few words Request is the field which you register your web app's Routes, their middleware, handlers, methods and so on.


```go
Request Request{
  DisablePathCorrection bool
  DisablePathEscape     bool
  // AllowMethodOptions if setted to true to allow all routes to be served to the client when http method 'OPTIONS', useful when user uses the Cors middleware
  // defaults to false
  AllowMethodOptions bool
  // Custom http errors handlers
  Errors map[int]func(*Context)
  // Middleware before any entry's main handler
  Begin []func(*Context)
  // Middleware after any entry's main handler
  Done []func(*Context)
  // if !=nil then this is used for the main router
  // if !=nil then the .Entry/.Entries,context.RedirectTo & all q's static handler will not work, you have to build them by yourself.
  Handler func(*Context)
  // The Routes
  Entries []Entry{
    // Name, optional, the name of the entry, used for .URL/.Path/context.RedirectTo inside templates: {{ url }}, {{ urlpath}}
    Name string
    // The http method
    Method string
    // set to true if you want this entry to be valid on HEAD http method also, defaults to false, useful when the entry serves static files
    Head   bool
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
    Parser interface{
      ParseEntry(e Entry) Entry
    }
  }
}

```

- Each `Entry` can have unlimited children `Etries`, these entries will use it's parents `Begin` & `Done` Handlers(Middleware) and the `Path` prefix.
- The `Entry.Method` can take one of these values: `"GET" | "POST" | "PUT" | "DELETE" | "PATCH" | "CONNECT" | "HEAD" | "OPTIONS"`,  they're just the `HTTP Methods`.
- If `Entry.Method` is not be setted then Q web framework will accept all client's http methods for that Entry.
- `Entry.Path`  & `Entry.Handler` cannot be nil.
- `Entry.Begin` handlers execution happens before the `Entry.Handler`.
- `Entry.Handler` is the 'main' Handler for a Route, its execution happens in the middle of Begin & Done handlers.
- `Entry.Done` handlers execution happens after the `Entry.Begin` & `Entry.Handler`.
- `Request` has `Begin` and `Done` fields also, if setted then these handlers are passed to all Entries.
- If `Entry.Path` ends with "." the Q web framework will act as it's a subdomain, if `Begin` & `Done` handlers are passed then are passed to its children Routes, the specific subdomain's routes. Handler field is not
- If `Entry` filled the `Entry.Entries`, has child routes, then it's Handler is ignored, only `Path`, `Begin` & `Done` matters at this situation, as it's logical, same for subdomains.
- `Entry.Parser` takes a interface as value, that interface should be implements the `ParseEntry(e Entry) Entry` function, look the `fs.go` file to see how you can use it.
Parser can change the user-defined Entry's fields, such as the Handler, Path , add middleware, or set a new Handler to a specific Entry.
- `Entry.Name` can be empty but if not then you can refer to this Entry/CompiledEntry(Route) to get it's full URL or Path by `$qinstnace.URL/Path("routeName", "pathParametersValues")`. Template engines also use this function to get assistance for {{ url }} and {{ urlpath }} helper functions, see below.


```go
q.Q{Host: "mydomain.com:80",
Request: q.Request{
  // global middleware execution happens before any entry's Begin and  Handler fields
  Begin:                 q.Handlers{myMiddleware},
  // global middleware execution happens after the entry's Begin and  Handler fields
  Done:                  q.Handlers{myMiddleware},
  Entries: q.Entries{
    // Static favicon first
    q.Entry{Path: "/favicon.ico", Parser: q.Favicon{"./assets/favicon.ico"}},
    q.Entry{Method: q.MethodGet, Path: "/home", Handler: myHandler},
    q.Entry{Path: "/users", Begin: q.Handlers{myMiddleware}, Done: q.Handlers{myMiddleware},
      Entries: q.Entries{
        // mydomain.com/users/signin
        q.Entry{Method: q.MethodGet, Path: "/signin", Handler: myHandler},
        // mydomain.com/users/signout
        q.Entry{Path: "/signout", Begin: q.Handlers{myMiddleware}, Done: q.Handlers{myMiddleware}, Handler: myHandler},
      }},
  },
}}.Go()

```


## Templates [optional field]

The `Templates` field is a slice of `q.Template` values, used to register custom or built'n template engines.

```go
type Template struct{
  Engine   interface{
    LoadDirectory(directory string, extension string) error
    LoadAssets(virtualDirectory string, virtualExtension string, assetFn func(name string) ([]byte, error), namesFn func() []string) error
    ExecuteWriter(out io.Writer, name string, binding interface{}, options ...map[string]interface{}) error
  }
  // location
  Directory string
  Extension string
  // binary (optional)
  Assets func(name string) ([]byte, error)
  Names  func() []string
}
```

The default template engine is the [html](https://github.com/kataras/q/tree/master/template/html) pointing on the `./templates` relative system directory and for files having the `.html` extension.
**6 built'n template engines** that you can register without need to install anything, you can build your own also, it's very easy, look [their source](https://github.com/kataras/q/tree/master/template) code to see how.

- [Standard html/template](https://github.com/kataras/q/tree/master/template/html)
- [Django](https://github.com/kataras/q/tree/master/template/django)
- [Pug/Jade](https://github.com/kataras/q/tree/master/template/pug)
- [Handlebars](https://github.com/kataras/q/tree/master/template/handlebars)
- [Amber](https://github.com/kataras/q/tree/master/template/amber)
- [Markdown](https://github.com/kataras/q/tree/master/template/markdown)


Each template engine contains these helper functions:
`{{ url "myRoutename" "anyPathParametersValues" }}`
`{{ urlpath "myRoutename" "anyPathParametersValues"}}`
`{{ render "partials/otherTemplate.html" }}`


All template engines supports `Gzip` and custom `Charset` setted per-render action to override the globals, using the `q.RenderOptions`, see below.

All template engines except Markdown & Pug have a `Layout` implementation also, not be explained here, you can view the [Iris' book also(https://kataras.gitbooks.io/iris/content/template-engines.html)], they are the same template engines.


- if `$qinstance.DevMode == true` reloads the templates on each request, defaults to false
- if `$qinstance.Gzip == true` compressed gzip contents to the client, the same for Response Engines also, defaults to false
- `$qinstance.Charset` defaults to "UTF-8", the same for Response Engines also

> when $qinstance.$field, we mean q.Q{ $field: }

- A template engine is registered by filling the `Template`'s `Engine` field
- Each `Template.Engine` has configuration, which you can find and read about them on the `kataras/q/template/$ENGINE/config.go` file, they are not explained at the README docs.
- Q web framework allows to register `more than one template engine` to use for, if file extension is different from the other.
- Template's `Execution/Render` can be done calling the `context.Render("templatefile.html", anyDataToSend{}, q.RenderOptions{"charset": "utf-8", "gzip": true})` inside a `q.Handler`.

Finally, let's see how we can register a template engine, you will be surprised

```go
//...
import (
  //...
  "github.com/kataras/q"
  "github.com/kataras/q/template/html"
)
//...
q.Q{
  Request: q.Request{
    Entries: q.Entries{
      q.Entry{Path: "/test_html", Handler: testHTML},
      q.Entry{Path: "/test_jade", Hadler: testJade},
    },
  },
  // ...
  Templates: q.Templates{
     /*this is the default engine, if you don't set any.*/
    q.Template{Engine: html.New(html.Config{Layout: "layouts/layout.html"}), Directory: "./templates", Extesion: ".html"},
     /*this is the second template engine, unlimited number of template engines to use*/
    q.Template{Egine: pug.New(pug.Config{/* the configuration is always optional*/ }), Directory: "./templates", Extension: ".jade"},
  },
//...

// say for example we have templates/hi.html and templates/hello.jade, let's see how we can render them one by one

// for /test_html
testHTML := func(ctx *q.Context) {
  ctx.Render("hi.html",nil)
}

// for /test_jade
testJade := func(ctx *q.Context) {
  ctx.Render("hello.jade",nil)
}
```

Get the template parsed result outside of Handler, useful when you want to send a rich e-mail and so on

```go
hiContents := $qinstance.TemplateString("hi.html",nil)
// or hiContents := ctx.Q().TemplateString("hi.html",nil) inside Handler, if you don't want to render the result, but just take it
```

## Responses [optional field]

The `Responses` field is a slice of `q.Response` values, used to register custom or built'n response engines

```go
type Response{
  Engine      interface{
    Response(interface{}, ...map[string]interface{}) ([]byte, error)
  }
  Name        string
  ContentType string  
}
```

The default response engines are the
- [JSON](https://github.com/kataras/q/tree/master/response/json), `Name` & `ContentType` fields to `"application/json"`
  - renders with `context.JSON` func
- [JSONP](https://github.com/kataras/q/tree/master/response/jsonp), `Name` & `ContentType` fields to `"application/javascript"`
  - renders with `context.JSONP` func
- [XML](https://github.com/kataras/q/tree/master/response/xml), `Name` & `ContentType` fields to `"text/xml"`
  - renders with `context.XML` func
- [Plain Text](https://github.com/kataras/q/tree/master/response/text), `Name` & `ContentType` fields to `"text/plain"`
  - renders with `context.Text` func
- [Binary Data](https://github.com/kataras/q/tree/master/response/data), `Name` & `ContentType` fields to `"application/octet-stream"`
  - renders with `context.Data` func
- [Markdown](https://github.com/kataras/q/tree/master/response/markdown), `Name` & `ContentType` fields to custom content type `"text/markdown"` which is converted to `"text/html"`
  - renders with `context.Markdown` func

> You don't have to register a response engine with the ContentType: ..."; charset=" , response engine adds this automatically

Custom and default response engines can be called ,also, by their names, through `context.Render("$Response.Name", interface{})`.

> when $Response.Name we mean the q.Response inside a: q.Q{Response: q.Response{ }}.

All response engines supports `Gzip` and custom `Charset` setted per-render action to override the globals, using the `q.RenderOptions`, see below.

- if `$qinstance.Gzip == true` compressed gzip contents to the client,  defaults to false
- `$qinstance.Charset` defaults to "UTF-8", the same for Response Engines also

> when $qinstance.$field, we mean q.Q{ $field: }.

- A response engine is registered by filling the `Response`'s `Engine` field
- Each `Response.Engine` has configuration, which you can find and read about them on the `kataras/q/response/$ENGINE/config.go` file, they are not explained at the README docs.
- Q web framework allows to register `more than one response engine` to use for, if `ContentType` field is different from the other.
- Response engines' results are appended to one, if more than one response engines has been registered with the same `ContentType` field (but different `Name` field).
- Response's `Render` can be done calling the `context.Render("application/json", q.Map{"name": "Q"})` inside a `q.Handler`.

Finally, let's see how we can register a response engine

```go
//...
import (
  //...
  "github.com/kataras/q"
  "github.com/kataras/q/response/json"
  "github.com/kataras/q/response/markdown"
)
//...
q.Q{
  Request: q.Request{
    Entries: q.Entries{
      q.Entry{Path: "/hi_json", Handler: testJSON},
      q.Entry{Path: "/hi_markdown", Hadler: testMarkdown},
    },
  },
  // ...
  Responses: q.Response{
     /*these are two of the six default engines */
    q.Respose{Engine: json.New(json.Config{/* the configuration is always optional*/ }}), ContentType: "application/json", Name: "application/json"},
     /*this is the second response engine, unlimited number of response engines to use*/
    q.Respose{Engine: markdown.New(), ContentType: "text/markdown", Name: "text/markdown"}, // text/markdown is generated to html and sent to browser as 'text/html'. So text/markdown is a Q-custom content type to split the text/html from text/markdown.
  },
//...

// for /hi_json
testJSON := func(ctx *q.Context) {
  ctx.Render("application/json", q.Map{"hi": "Q!"})
  // same as:
  // ctx.JSON(q.Map{"hi": "Q!",})
}

// for /hi_markdown
testMarkdown := func(ctx *q.Context) {
  ctx.Render("text/markdown", "# Hello Markdown from Q!", q.RenderOptions{"charset": "UTF-8"}) // Optional, RenderOptions's values can override the $qinstance.Charset/Gzip fields
  // same as:
  // ctx.Markdown("# Hello Markdown from Q!")
}
```

Get the response engine's result outside of Handler, useful when you want to send a rich e-mail and so on

```go
hiContents := $qinstance.ResponseString("text/markdown", "# Hello Markdown from Q outside of a Handler!")
// or hiContents := ctx.Q().ResponseString("text/markdown", "# Hello Markdown from Q outside of a Handler!") inside Handler, if you don't want to render the result, but just take it
```


## Session [optional field]

As the other features, Q's sessions are unique, so if you find any bug post it [here](https://github.com/kataras/q/issues) please.

- Cleans the temp memory when a sessions is iddle, and re-allocate it , fast, to the temp memory when it's necessary. Also most used/regular sessions are going front in the memory's list.

- Supports any type of database, currently only [redis](https://github.com/kataras/q/sessiondb/).

**A session can be defined as a server-side storage of information that is desired to persist throughout the user's interaction with the web site** or web application.

Instead of storing large and constantly changing information via cookies in the user's browser, **only a unique identifier is stored on the client side** (called a "session id"). This session id is passed to the web server every time the browser makes an HTTP request (ie a page link or AJAX request). The web application pairs this session id with it's internal database/memory and retrieves the stored variables for use by the requested page.

Session is a single field, doesn't takes more than one `q.Session`.

```go
Session  Session{
  // Cookie string, the session's client cookie name, for example: "mysessionid"
  Cookie string
  // DecodeCookie set it to true to decode the cookie key with base64 URLEncoding
  // Defaults to false
  DecodeCookie bool
  // Expires the duration of which the cookie must expires (created_time.Add(Expires)).
  // If you want to delete the cookie when the browser closes, set it to -1 but in this case, the server side's session duration is up to GcDuration
  //
  // Default infinitive/unlimited life duration(0)

  Expires time.Duration
  // GcDuration every how much duration(GcDuration) the memory should be clear for unused cookies (GcDuration)
  // for example: time.Duration(2)*time.Hour. it will check every 2 hours if cookie hasn't be used for 2 hours,
  // deletes it from backend memory until the user comes back, then the session continue to work as it was
  //
  // Default 2 hours
  GcDuration time.Duration

  // DisableSubdomainPersistence set it to true in order dissallow your q subdomains to have access to the session cookie
  // defaults to false
  DisableSubdomainPersistence bool
  // UseSessionDB registers a session database, you can register more than one
  // accepts a session database which implements a Load(sid string) map[string]interface{} and an Update(sid string, newValues map[string]interface{})
  // the only reason that a session database will be useful for you is when you want to keep the session's values/data after the app restart
  // a session database doesn't have write access to the session, it doesn't accept the context, so forget 'cookie database' for sessions, I will never allow that, for your protection.
  //
  // Note: Don't worry if no session database is registered, your context.Session will continue to work.
  Databases []interface{
    Load(string) map[string]interface{}
    Update(string, map[string]interface{})
  }
}
```

Let's see how we can register a session manager

```go
q.Q{Host:          "mydomain.com:80",
  // Cookie field is required
  // Expires field is optional, defaults to 23 years
  // GcDuration field is optional, defaults to 2 hous
  // Databases field is optional, needed only when u want to keep the sessions after http server's shutdown or restart.
  Session: q.Session{Cookie: "mysessionid", Expires: 4 * time.Hour, GcDuration: 2 * time.Hour, Databases: q.Databases{redis.New()}},
  // other fields here...
}.Go()
```

Get/Set/Clear per-user-session values using the `context.Session().Set/Get/Clear...` which returns the `SessionStore interface`:

```go
type SessionStore interface {
  ID() string
  Get(string) interface{}
  GetString(key string) string
  GetInt(key string) int
  GetAll() map[string]interface{}
  VisitAll(cb func(k string, v interface{}))
  Set(string, interface{})
  Delete(string)
  Clear()
}
```

```go
sessID := context.Session().ID()
object := context.Session().Get("user").(User) // User is custom struct only for the example
nameStr := context.Session().GetString("name")
ageInt := context.Session().GetInt("age")
all := context.Session().GetAll()

context.Session().Set("name", "Q")
context.Session().Delete("name")
context.Session().Clear() // removes all values
```

**A small example**

```go
package main

import (
  "github.com/kataras/q"
  "github.com/kataras/q/sessiondb/redis"
)

func main(){

q.Q{Host: "mydomain.com:80",
  Session: q.Session{Cookie: "mysessionid", Expires: 4 * time.Hour, GcDuration: 2 * time.Hour, Databases: q.Databases{redis.New()}},
  Request: q.Request{
    Entries: q.Entries{
      	q.Entry{Path: "/sessions", Entries: q.Entries{
          // http://mydomain.com/sessions/set
      		q.Entry{Method: q.MethodGet, Path: "/set", Handler: func(ctx *q.Context) {
      			key := "name"
      			val := "my Q"
      			ctx.Session().Set(key, val)
      			ctx.WriteString("Setted: %s=%s", key, val)
      		}},
          // http://mydomain.com/sessions/get
      		q.Entry{Method: q.MethodGet, Path: "/get", Handler: func(ctx *q.Context) {
      			key := "name"
      			val := ctx.Session().GetString(key)
      			ctx.WriteString("Setted: %s=%s", key, val)
      		}},
          // http://mydomain.com/sessions/clear
      		q.Entry{Method: q.MethodGet, Path: "/clear", Handler: func(ctx *q.Context) {
      			ctx.Session().Clear()
      			if len(ctx.Session().GetAll()) > 0 {
      				ctx.Text("Session().GetAll didn't worked!")
      				 else {
      				ctx.Text("All Session's values removed, but the cookie exists, use ctx.SessionDestroy() to remove all values and cookie and server-side store.")
      			}
      		}},
          // http://mydomain.com/sessions/destroy
      		q.Entry{Method: q.MethodGet, Path: "/destroy", Handler: func(ctx *q.Context) {
      			ctx.SessionDestroy()
      			if len(ctx.Session().GetAll()) > 0 {
      				ctx.Text("SessionDestroy() didn't worked!")
      			} else {
      				ctx.Text("Session destroyed.")
      			}
      		}},
      	}},
    },
  },
}.Go()

}
```

### Flash messages

**A flash message is used in order to keep a message in session through one request of the same user**. By default, it is removed from session after it has been displayed to the user. Flash messages are usually used in combination with HTTP redirections, because in this case there is no view, so messages can only be displayed in the request that follows redirection.

**A flash message has a name and a content (AKA key and value). It is an entry of a map**. The name is a string: often "notice", "success", or "error", but it can be anything. The content is usually a string. You can put HTML tags in your message if you display it raw. You can also set the message value to a number or an array: it will be serialized and kept in session like a string.

----

```go

// SetFlash sets a flash message, accepts 2 parameters the key(string) and the value(string)

// the value will be available on the NEXT request

SetFlash(key string, value string)

// GetFlash get a flash message by it's key

// returns the value as string and an error

//

// if the cookie doesn't exists the string is empty and the error is filled

// after the request's life the value is removed

GetFlash(key string) (value string, err error)

// GetFlashes returns all the flash messages for available for this request

GetFlashes() map[string]string

```


```go
func myHandler(ctx *q.Context) {
  q.SetFlash("message", "Hello") // message: "hello" is removed on the first request which done by THE SAME CLIENT, will get this value  
}

func myHandler2(ctx *q.Context){ // same client, user
  hello := q.GetFlash("message")
}
```

## Websockets [optional field]

**WebSocket is a protocol providing full-duplex communication channels over a single TCP connection**. The WebSocket protocol was standardized by the IETF as RFC 6455 in 2011, and the WebSocket API in Web IDL is being standardized by the W3C.

WebSocket is designed to be implemented in web browsers and web servers, but it can be used by any client or server application. The WebSocket Protocol is an independent TCP-based protocol. Its only relationship to HTTP is that its handshake is interpreted by HTTP servers as an Upgrade request. The WebSocket protocol makes more interaction between a browser and a website possible, **facilitating the real-time data transfer from and to the server**.

[Read more about Websockets on wikipedia](https://en.wikipedia.org/wiki/WebSocket)

```go
// WebsocketConnectionFunc is the callback which fires when a client/websocketConnection is connected to the websocketServer.
// Receives one parameter which is the WebsocketConnection
type WebsocketConnectionFunc func(WebsocketConnection)
```

**WebsocketConnectionFunc is the type of the `Weboskcet.Handler`**

```go
Websockets []Websocket{
  // Endpoint is the path which the websocket server will listen for clients/connections
  // Default value is empty string, if you don't set it the Websocket server is disabled.
  Endpoint string
  // Handler the main handler, when a client connects
  Handler WebsocketConnectionFunc
  // ClientSourcePath is the path which the javascript client-side library for Q websockets will be served
  // it's the request path, not the system path, there is no system path, the Q is automatically provides the source code without any needed system file.
  // Default value is /qws.js, means that you will have to import the 'yourdomain.com/qws.js' using the html script tag
  ClientSourcePath string

  Error       func(res http.ResponseWriter, req *http.Request, status int, reason error)
  CheckOrigin func(req *http.Request) bool
  // WriteTimeout time allowed to write a message to the connection.
  // Default value is 15 * time.Second
  WriteTimeout time.Duration
  // PongTimeout allowed to read the next pong message from the connection
  // Default value is 60 * time.Second
  PongTimeout time.Duration
  // PingPeriod send ping messages to the connection with this period. Must be less than PongTimeout
  // Default value is (PongTimeout * 9) / 10
  PingPeriod time.Duration
  // MaxMessageSize max message size allowed from connection
  // Default value is 1024
  MaxMessageSize int64

  // ReadBufferSize is the buffer size for the underline reader
  ReadBufferSize int
  // WriteBufferSize is the buffer size for the underline writer
  WriteBufferSize int
}
```


Each new client joins to the websocket server, the `Websocket.Handler` field's value(function) is fired, it sets the `WebsocketConnection` function's receiver parameter.
You work only with the `WebsocketConnection`, so give attention to **WebsocketConnection's functions**:


```go

// Receive from the client

On("anyCustomEvent", func(message string) {})

On("anyCustomEvent", func(message int){})

On("anyCustomEvent", func(message bool){})

On("anyCustomEvent", func(message anyCustomType){})

On("anyCustomEvent", func(){})

// Receive a native websocket message from the client

// compatible without need of import the q.Websocket 's client side source code(/qws.js) to the .html

OnMessage(func(message []byte){})

// Send to the client

Emit("anyCustomEvent", string)

Emit("anyCustomEvent", int)

Emit("anyCustomEvent", bool)

Emit("anyCustomEvent", anyCustomType)

// Send via native websocket way, compatible without need of import the q.Websocket 's client side source code(/qws.js) to the .html

EmitMessage([]byte("anyMessage"))

// Send to specific client(s)

To("otherConnectionId").Emit/EmitMessage...

To("anyCustomRoom").Emit/EmitMessage...

// Send to all opened connections/clients

To(q.All).Emit/EmitMessage...

// Send to all opened connections/clients EXCEPT this client(c)

To(q.NotMe).Emit/EmitMessage...

// Rooms, group of connections/clients

Join("anyCustomRoom")

Leave("anyCustomRoom")

// Fired when the connection is closed

OnDisconnect(func(){})

// Force-disconnect the client from the server-side
Disconnect() error

```

### Let's view a basic silly example:

**Server-side**

```go
// ./main.go
package main

import (
	"fmt"

	"github.com/kataras/q"
)

type clientPage struct {
	Title string
	Host  string
}

func main() {

	q.Q{Host: ":80",
		Websockets: q.Websockets{
			// Endpoint: the path which the websocket client should listen/registed to
			// ClientSourcePath: see the .html file <script> tag
			q.Websocket{Endpoint: "/my_endpoint", Handler: onConnection, ClientSourcePath: "/qws.js"},
		},
		Request: q.Request{
			Entries: q.Entries{
				q.Entry{Method: q.MethodGet, Path: "/js", Parser: q.Dir{Directory: "./static/js"}},
				q.Entry{Method: q.MethodGet, Path: "/", Handler: indexHandler},
			},
		},
	}.Go()
}

func indexHandler(ctx *q.Context) {
	ctx.MustRender("client.html", clientPage{"Client Page", ctx.Request.Host})
}

func onConnection(c q.WebsocketConnection) {
	room := "room1"
	c.Join(room)

	c.On("chat", func(message string) {
		// to all except this connection ->
		//c.To(q.Broadcast).Emit("chat", "Message from: "+c.ID()+"-> "+message)

		// to the client ->
		//c.Emit("chat", "Message from myself: "+message)

		//send the message to the whole room,
		//all connections are inside this room will receive this message
		c.To(room).Emit("chat", "From: "+c.ID()+": "+message)
	})

	c.OnDisconnect(func() {
		fmt.Printf("\nConnection with ID: %s has been disconnected!", c.ID())
	})

}
}

```

**Client-side**

```js

// ./static/js/chat.js. Requested at: http://127.0.0.1:80/js/chat.js
var messageTxt;
var messages;

$(function () {

	messageTxt = $("#messageTxt");
	messages = $("#messages");


	w = new Ws("ws://" + HOST + "/my_endpoint");
	w.OnConnect(function () {
		console.log("Websocket connection established");
	});

	w.OnDisconnect(function () {
		appendMessage($("<div><center><h3>Disconnected</h3></center></div>"));
	});

	w.On("chat", function (message) {
		appendMessage($("<div>" + message + "</div>"));
	});

	$("#sendBtn").click(function () {
		w.Emit("chat", messageTxt.val().toString());
		messageTxt.val("");
	});

})


function appendMessage(messageDiv) {
    var theDiv = messages[0];
    var doScroll = theDiv.scrollTop == theDiv.scrollHeight - theDiv.clientHeight;
    messageDiv.appendTo(messages);
    if (doScroll) {
        theDiv.scrollTop = theDiv.scrollHeight - theDiv.clientHeight;
    }
}

```

```html
<!-- ./templates/client.html -->
<html>

<head>
<title>{{ .Title}}</title>
</head>

<body>
	<div id="messages"
		style="border-width: 1px; border-style: solid; height: 400px; width: 375px;">

	</div>
	<input type="text" id="messageTxt" />
	<button type="button" id="sendBtn">Send</button>
	<script type="text/javascript">
		var HOST = {{.Host}}
	</script>
	<script src="js/vendor/jquery-2.2.3.min.js" type="text/javascript"></script>
	<!-- This is auto-serving by the Q web framework, you don't need to have this file in your disk-->
	<script src="/qws.js" type="text/javascript"></script>
	<!-- -->
	<script src="js/chat.js" type="text/javascript"></script>
</body>

</html>

```


Community
------------

If you'd like to discuss this package, or ask questions about it, feel free to

 * Post an issue or  idea [here](https://github.com/kataras/q/issues).
 * [Chat][Chat].


FAQ
------------
Explore [these questions](https://github.com/kataras/q/issues?q=label%3Aquestion) or navigate to the [community chat][Chat].

Philosophy
------------

The Q's philosophy is to provide robust tooling for HTTP, making it a great solution for single page applications, web sites, hybrids, or public HTTP APIs.

Q web framework does not force you to use any specific ORM or template engine. With support for the most used template engines, you can quickly craft the perfect application.



Testing
------------

I recommend writing your API tests using this new library, [httpexpect](https://github.com/gavv/httpexpect).
- test.go test files use the built'n Q's test framework which is based on httpexpect.

`NewTester(q *Q, t *testing.T) *httpexpect.Expect`


Versioning
------------

Current: **v.0.0.1**
>  Q is an active project

Read more about Semantic Versioning 2.0.0

-  http://semver.org/
-  https://en.wikipedia.org/wiki/Software_versioning
-  https://wiki.debian.org/UpstreamGuide#Releases_and_Versions

TODO
------------

- [ ] [rizla](https://github.com/kataras/rizla) monitor integration for re-build on code chages, when `DevMode` is true.
- [ ] Add unit tests where Cyclomatic complexity is high

People
------------

The author of Q is [@kataras](https://github.com/kataras).

I spend all my time in the construction of `Q`, therefore I have no income value.

If you,

- think that any information you obtained here is worth some money
- believe that `Q` worth to remains a highly active project

Feel free to send **any** amount through `paypal`

[![](https://www.paypalobjects.com/en_US/i/btn/btn_donateCC_LG.gif)](https://www.paypal.com/cgi-bin/webscr?cmd=_donations&business=makis%40ideopod%2ecom&lc=GR&item_name=Q%20web%20framework&item_number=qwebframeworkdonationid2016&currency_code=EUR&bn=PP%2dDonationsBF%3abtn_donateCC_LG%2egif%3aNonHosted)

**Thanks!**

Contributing
------------
If you are interested in contributing to the Q project, feel free to discuss new ideas or post bug reports [here](https://github.com/kataras/q/issues).

License
------------

This project is licensed under the MIT License.

License can be found [here](LICENSE).

[Travis Widget]: https://img.shields.io/travis/kataras/q.svg?style=flat-square
[Travis]: http://travis-ci.org/kataras/q
[License Widget]: https://img.shields.io/badge/license-MIT%20%20License%20-E91E63.svg?style=flat-square
[License]: https://github.com/kataras/q/blob/master/LICENSE
[Release Widget]: https://img.shields.io/badge/release-v0.0.1-blue.svg?style=flat-square
[Release]: https://github.com/kataras/q/releases
[Chat Widget]: https://img.shields.io/badge/community-chat-00BCD4.svg?style=flat-square
[Chat]: https://kataras.rocket.chat/channel/q
[Report Widget]: https://img.shields.io/badge/report%20card-A%2B-F44336.svg?style=flat-square
[Report]: http://goreportcard.com/report/kataras/q
[Documentation Widget]: https://img.shields.io/badge/documentation-reference-5272B4.svg?style=flat-square
[Documentation]: https://www.gitbook.com/book/kataras/q/details
[Examples Widget]: https://img.shields.io/badge/examples-repository-3362c2.svg?style=flat-square
[Examples]: https://github.com/q-contrib/examples
[Language Widget]: https://img.shields.io/badge/powered_by-Go-3362c2.svg?style=flat-square
[Language]: http://golang.org
[Platform Widget]: https://img.shields.io/badge/platform-Any--OS-gray.svg?style=flat-square
