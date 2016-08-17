package q

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	"net/http"
	"net/url"

	"github.com/kataras/q/errors"
	"github.com/q-contrib/formBinder"
)

const (
	// DefaultUserAgent default to 'q' but it is not used anywhere yet
	defaultUserAgent = "q"
	// ContentType represents the header["Content-Type"]
	contentType = "Content-Type"
	// ContentLength represents the header["Content-Length"]
	contentLength = "Content-Length"
	// contentEncodingHeader represents the header["Content-Encoding"]
	contentEncodingHeader = "Content-Encoding"
	// varyHeader represents the header "Vary"
	varyHeader = "Vary"
	// acceptEncodingHeader represents the header key & value "Accept-Encoding"
	acceptEncodingHeader = "Accept-Encoding"
	// ContentHTML is the  string of text/html response headers
	contentHTML = "text/html"
	// ContentBinary header value for binary data.
	contentBinary = "application/octet-stream"
	// ContentJSON header value for JSON data.
	contentJSON = "application/json"
	// ContentJSONP header value for JSONP data.
	contentJSONP = "application/javascript"
	// ContentText header value for Text data.
	contentText = "text/plain"
	// ContentXML header value for XML data.
	contentXML = "text/xml"

	// contentMarkdown custom key/content type, the real is the text/html
	contentMarkdown = "text/markdown"

	// LastModified "Last-Modified"
	lastModified = "Last-Modified"
	// IfModifiedSince "If-Modified-Since"
	ifModifiedSince = "If-Modified-Since"
	// ContentDisposition "Content-Disposition"
	contentDisposition = "Content-Disposition"

	// stopExecutionPosition used inside the Context, is the number which shows us that the context's middleware manualy stop the execution
	stopExecutionPosition = 1993
	// used inside GetFlash to store the lifetime request flash messages
	flashMessagesStoreContextKey = "_q_flash_messages_"
	flashMessageCookiePrefix     = "_q_flash_message_"
	cookieHeaderID               = "Cookie: "
	cookieHeaderIDLen            = len(cookieHeaderID)
	httpStatusCodeContextKey     = "_q_http_status_code_"
)

// errors

var (
	errTemplateExecute  = errors.New("Unable to execute a template. Trace: %s")
	errFlashNotFound    = errors.New("Unable to get flash message. Trace: Cookie does not exists")
	errSessionNil       = errors.New("Unable to set session, Config().Session.Provider is nil, please refer to the docs!")
	errNoForm           = errors.New("Request has no any valid form")
	errWriteJSON        = errors.New("Before JSON be written to the body, JSON Encoder returned an error. Trace: %s")
	errRenderMarshalled = errors.New("Before +type Rendering, MarshalIndent returned an error. Trace: %s")
	errReadBody         = errors.New("While trying to read %s from the request body. Trace %s")
	errServeContent     = errors.New("While trying to serve content to the client. Trace %s")
)

type requestValue struct {
	key   []byte
	value interface{}
}

type requestValues []requestValue

func (r *requestValues) Set(key string, value interface{}) {
	args := *r
	n := len(args)
	for i := 0; i < n; i++ {
		kv := &args[i]
		if string(kv.key) == key {
			kv.value = value
			return
		}
	}

	c := cap(args)
	if c > n {
		args = args[:n+1]
		kv := &args[n]
		kv.key = append(kv.key[:0], key...)
		kv.value = value
		*r = args
		return
	}

	kv := requestValue{}
	kv.key = append(kv.key[:0], key...)
	kv.value = value
	*r = append(args, kv)
}

func (r *requestValues) Get(key string) interface{} {
	args := *r
	n := len(args)
	for i := 0; i < n; i++ {
		kv := &args[i]
		if string(kv.key) == key {
			return kv.value
		}
	}
	return nil
}

func (r *requestValues) Reset() {
	args := *r
	n := len(args)
	for i := 0; i < n; i++ {
		v := args[i].value
		// close any values which implements the Closer, some of the ORM packages does that.
		if vc, ok := v.(io.Closer); ok {
			vc.Close()
		}
	}
	*r = (*r)[:0]
}

type (
	// Map is just a conversion for a map[string]interface{}
	// should not be used inside Render when PongoEngine is used.
	Map map[string]interface{}

	// Context the bridge between q's functionality and a client's request
	Context struct {
		ResponseWriter http.ResponseWriter
		Request        *http.Request
		values         requestValues
		Params         PathParameters
		q              *Q
		//keep track all registed middleware (handlers)
		handlers Handlers
		session  *sessionStore
		// pos is the position number of the Context, look .Serve & .Cancel to understand
		pos int
	}
)

// Serve calls the middleware
func (ctx *Context) Serve() {
	ctx.handlers[ctx.pos](ctx)
	ctx.pos++
	//run the next
	if ctx.pos < len(ctx.handlers) {
		ctx.Serve()
	}
}

// ForceNext forces to serve the next handler
func (ctx *Context) ForceNext() {
	ctx.pos++

	if ctx.pos < len(ctx.handlers) {
		ctx.handlers[ctx.pos](ctx)
	}
}

// StopExecution just sets the .pos to 255 in order to  not move to the next middlewares(if any)
func (ctx *Context) StopExecution() {
	ctx.pos = stopExecutionPosition
}

// Cancel just sets the .pos to 255 in order to  not move to the next middlewares(if any)
// same as StopExecution
func (ctx *Context) Cancel() {
	ctx.StopExecution()
}

//

// IsStopped checks and returns true if the current position of the Context is 255, means that the StopExecution has called
func (ctx *Context) IsStopped() bool {
	return ctx.pos == stopExecutionPosition
}

// GetHandlerName as requested returns the stack-name of the function which the Middleware is setted from
func (ctx *Context) GetHandlerName() string {
	return runtime.FuncForPC(reflect.ValueOf(ctx.handlers[len(ctx.handlers)-1]).Pointer()).Name()
}

// Q returns the q web framework 'station' instance
func (ctx *Context) Q() *Q {
	return ctx.q
}

/* Request */

// Param returns the string representation of the key's path named parameter's value
func (ctx *Context) Param(key string) string {
	return ctx.Params.Get(key)
}

// ParamInt returns the int representation of the key's path named parameter's value
func (ctx *Context) ParamInt(key string) (int, error) {
	return strconv.Atoi(ctx.Param(key))
}

// ParamInt64 returns the int64 representation of the key's path named parameter's value
func (ctx *Context) ParamInt64(key string) (int64, error) {
	return strconv.ParseInt(ctx.Param(key), 10, 64)
}

// URLParams returns a map of a list of each url(query) parameter
func (ctx *Context) URLParams() url.Values {
	return ctx.Request.URL.Query()
}

// URLParam returns the get parameter from a request , if any
func (ctx *Context) URLParam(key string) string {
	return ctx.Request.URL.Query().Get(key)
}

// URLParamInt returns the url query parameter as int value from a request ,  returns error on parse fail
func (ctx *Context) URLParamInt(key string) (int, error) {
	return strconv.Atoi(ctx.URLParam(key))
}

// URLParamInt64 returns the url query parameter as int64 value from a request ,  returns error on parse fail
func (ctx *Context) URLParamInt64(key string) (int64, error) {
	return strconv.ParseInt(ctx.URLParam(key), 10, 64)
}

// Path returns the full escaped path as string
func (ctx *Context) Path() string {
	return ctx.RequestPath(true)
}

// RequestPath returns the requested path
func (ctx *Context) RequestPath(escape bool) string {
	if escape {
		return ctx.Request.URL.EscapedPath()
	}
	return ctx.Request.URL.Path
}

// RequestIP gets just the Remote Address from the client.
func (ctx *Context) RequestIP() string {
	if ip, _, err := net.SplitHostPort(strings.TrimSpace(ctx.Request.RemoteAddr)); err == nil {
		return ip
	}
	return ""
}

// RemoteAddr is like RequestIP but it checks for proxy servers also, tries to get the real client's request IP
func (ctx *Context) RemoteAddr() string {
	header := ctx.RequestHeader("X-Real-Ip")
	realIP := strings.TrimSpace(header)
	if realIP != "" {
		return realIP
	}
	realIP = ctx.RequestHeader("X-Forwarded-For")
	idx := strings.IndexByte(realIP, ',')
	if idx >= 0 {
		realIP = realIP[0:idx]
	}
	realIP = strings.TrimSpace(realIP)
	if realIP != "" {
		return realIP
	}
	return ctx.RequestIP()
}

// RequestHeader returns the request header's value
// accepts one parameter, the key of the header (string)
// returns string
func (ctx *Context) RequestHeader(key string) string {
	return ctx.Request.Header.Get(key)
}

// FormValues returns a slice of string from post request's data
func (ctx *Context) FormValues(key string) []string {
	sepCommasValues := ctx.Request.Form.Get(key)
	listValues := strings.Split(sepCommasValues, ",")
	return listValues
}

// Subdomain returns the subdomain (string) of this request, if any
func (ctx *Context) Subdomain() (subdomain string) {
	host := ctx.Request.Host
	if index := strings.IndexByte(host, '.'); index > 0 {
		subdomain = host[0:index]
	}

	return
}

// Body reads & returns all request's body contents
func (ctx *Context) Body() ([]byte, error) {
	return ioutil.ReadAll(ctx.Request.Body)
}

// ReadJSON reads JSON from request's body
func (ctx *Context) ReadJSON(jsonObject interface{}) error {
	b, err := ctx.Body()
	if err != nil && err != io.EOF {
		return err
	}
	decoder := json.NewDecoder(strings.NewReader(string(b)))
	err = decoder.Decode(jsonObject)

	if err != nil && err != io.EOF {
		return errReadBody.Format("JSON", err.Error())
	}
	return nil
}

// ReadXML reads XML from request's body
func (ctx *Context) ReadXML(xmlObject interface{}) error {
	b, err := ctx.Body()
	if err != nil && err != io.EOF {
		return err
	}
	decoder := xml.NewDecoder(strings.NewReader(string(b)))
	err = decoder.Decode(xmlObject)
	if err != nil && err != io.EOF {
		return errReadBody.Format("XML", err.Error())
	}

	return nil
}

// ReadForm binds the formObject  with the form data
// it supports any kind of struct
func (ctx *Context) ReadForm(formObject interface{}) error {
	err := ctx.Request.ParseForm()
	if err != nil {
		return err
	}
	reqCtx := ctx.Request
	// first check if we have multipart form
	if mfv := reqCtx.MultipartForm.Value; len(mfv) > 0 {
		//we have multipart form
		return errReadBody.With(formBinder.Decode(mfv, formObject))
	}

	// if no multipart and post arguments or get arguments (by GET FORM)
	postArgs := reqCtx.Form
	getArgs := reqCtx.URL.Query()

	if len(postArgs) > 0 {
		// if we have post args then we ignore the get args
		return errReadBody.With(formBinder.Decode(postArgs, formObject))
	} else if len(getArgs) > 0 {
		// if we don't have post args then we include the get args
		return errReadBody.With(formBinder.Decode(getArgs, formObject))
	}

	return errReadBody.With(errNoForm.Return())
}

/* Response */

// SetStatusCode sets the status code via ResponseWriter.WriteHeader
// we could make a custom ResponseWriter to hold the status code, but, keep things simple to the end-user
func (ctx *Context) SetStatusCode(code int) {
	prevCode := ctx.GetInt(httpStatusCodeContextKey)
	if prevCode == -1 { // if not setted before using SetStatusCode then set it.
		ctx.Set(httpStatusCodeContextKey, code)
		ctx.ResponseWriter.WriteHeader(code)
	}
}

// StatusCode returns the http status code, if no status code was written yet, then it returns 200
// works only when you setted the status code via context.SetStatusCode.
func (ctx *Context) StatusCode() int {
	if statusCode := ctx.GetInt(httpStatusCodeContextKey); statusCode > 0 {
		return statusCode
	}
	return StatusOK
}

// SetContentType sets the response writer's header key 'Content-Type' to a given value(s)
func (ctx *Context) SetContentType(value string) {
	ctx.ResponseWriter.Header().Set(contentType, value)
}

// SetHeader write to the response writer's header to a given key the given value(s)
//
// Note: If you want to send a multi-line string as header's value use: strings.TrimSpace first.
func (ctx *Context) SetHeader(key string, value string) {
	//value = strings.TrimSpace(v)
	ctx.ResponseWriter.Header().Set(key, value)
}

// Redirect redirect sends a redirect response the client
// accepts 2 parameters string and an optional int
// first parameter is the url to redirect
// second parameter is the http status should send, default is 302 (StatusFound), you can set it to 301 (Permant redirect), if that's nessecery
func (ctx *Context) Redirect(path string, statusCode ...int) {
	if len(path) == 0 {
		path = "/"
	}

	httpStatus := StatusFound // a 'temporary-redirect-like' wich works better than for our purpose
	if statusCode != nil && len(statusCode) > 0 && statusCode[0] > 0 {
		httpStatus = statusCode[0]
	}

	if httpStatus != StatusMovedPermanently && httpStatus != StatusFound &&
		httpStatus != StatusSeeOther && httpStatus != StatusTemporaryRedirect {
		httpStatus = StatusFound
	}

	urlToRedirect := path

	if path[0] == slashByte {
		// means relative
		ur, err := ctx.Request.URL.Parse(path)
		if err != nil {
			return
		}
		urlToRedirect = ur.EscapedPath()
	}

	ctx.SetHeader("Location", urlToRedirect)
	ctx.ResponseWriter.WriteHeader(httpStatus)
	ctx.StopExecution()
}

// RedirectTo does the same thing as Redirect but instead of receiving a uri or path it receives a route name
func (ctx *Context) RedirectTo(routeName string, args ...interface{}) {
	s := ctx.q.URL(routeName, args...)
	if s != "" {
		ctx.Redirect(s, StatusFound)
	}
}

// NotFound emits an error 404 to the client, using the custom http errors
// if no custom errors provided then it sends the default error message
func (ctx *Context) NotFound() {
	ctx.EmitError(StatusNotFound)
}

// Panic emits an error 500 to the client, using the custom http errors
// if no custom errors rpovided then it sends the default error message
func (ctx *Context) Panic() {
	ctx.EmitError(StatusInternalServerError)
}

// EmitError executes the custom error by the http status code passed to the function
func (ctx *Context) EmitError(statusCode int) {
	errHandler := ctx.q.Request.Errors[statusCode] //one reader, no need to lock this
	if errHandler != nil {
		errHandler(ctx)
	}
	ctx.Cancel()
}

// Write writes the data to the connection as part of an HTTP reply.
// If WriteHeader has not yet been called, Write calls WriteHeader(http.StatusOK)
// before writing the data.  If the Header does not contain a
// Content-Type line, Write adds a Content-Type set to the result of passing
// the initial 512 bytes of written data to DetectContentType.
func (ctx *Context) Write(b []byte) (int, error) {
	return ctx.ResponseWriter.Write(b)
}

// WriteString writes a string to the client, something like fmt.Printf but for the web
func (ctx *Context) WriteString(format string, a ...interface{}) {
	io.WriteString(ctx.ResponseWriter, fmt.Sprintf(format, a...))
}

func (ctx *Context) clientAllowsGzip() bool {
	if h := ctx.RequestHeader(acceptEncodingHeader); h != "" {
		for _, v := range strings.Split(h, ";") {
			if strings.Contains(v, "gzip") { // we do Contains because sometimes browsers has the q=, we don't use it atm. || strings.Contains(v,"deflate"){
				return true
			}
		}
	}

	return false
}

// WriteGzip accepts bytes, which will be compressed using gzip compression and will be sent to the client
func (ctx *Context) WriteGzip(b []byte) (int, error) {
	ctx.ResponseWriter.Header().Add(varyHeader, acceptEncodingHeader)
	n, err := WriteGzip(ctx.ResponseWriter, b)
	if err == nil {
		ctx.SetHeader("Content-Encoding", "gzip")
	}
	return n, err
}

// Render if no http status code was written before, the StatusOK(200) will be sent
// builds up & send the response from the specified template or a response engine.
// Note: the options: "gzip" and "charset" are built'n support by Iris, so you can pass these on any template engine or response engine
func (ctx *Context) Render(name string, binding interface{}, options ...map[string]interface{}) error {
	if strings.IndexByte(name, '.') > -1 { //we have template
		return ctx.q.templates.getBy(name).execute(ctx, name, binding, options...)
	}
	return ctx.q.responses.getBy(name).render(ctx, binding, options...)
}

// MustRender same as .Render but returns 500 internal server http status (error) if rendering fail
// builds up the response from the specified template or a response engine.
// Note: the options: "gzip" and "charset" are built'n support by Iris, so you can pass these on any template engine or response engine
func (ctx *Context) MustRender(name string, binding interface{}, options ...map[string]interface{}) {
	if err := ctx.Render(name, binding, options...); err != nil {
		ctx.Panic()
		ctx.q.Logger.Printf("MustRender panics for client with IP: %s On template: %s.Trace: %s\n", ctx.RemoteAddr(), name, err)
	}
}

// HTML writes html string
func (ctx *Context) HTML(htmlContents string) {
	if err := ctx.Render(contentHTML, htmlContents); err != nil {
		// if no response engine found for text/html
		ctx.SetContentType(contentHTML + "; charset=" + ctx.q.Charset)
		ctx.WriteString(htmlContents)
	}
}

// Data writes out the raw bytes as binary data.
func (ctx *Context) Data(v []byte) error {
	return ctx.Render(contentBinary, v)
}

// JSON marshals the given interface object and writes the JSON response.
func (ctx *Context) JSON(v interface{}) error {
	return ctx.Render(contentJSON, v)
}

// JSONP marshals the given interface object and writes the JSON response.
func (ctx *Context) JSONP(callback string, v interface{}) error {
	return ctx.Render(contentJSONP, v, map[string]interface{}{"callback": callback})
}

// Text writes out a string as plain text.
func (ctx *Context) Text(v string) error {
	return ctx.Render(contentText, v)
}

// XML marshals the given interface object and writes the XML response.
func (ctx *Context) XML(v interface{}) error {
	return ctx.Render(contentXML, v)
}

// MarkdownString parses the (dynamic) markdown string and returns the converted html string
func (ctx *Context) MarkdownString(markdownText string) string {
	return ctx.q.ResponseString(contentMarkdown, markdownText)
}

// Markdown parses and renders to the client a particular (dynamic) markdown string
// receives  the markdown string
func (ctx *Context) Markdown(markdown string) {
	ctx.HTML(ctx.MarkdownString(markdown))
}

// ServeContent serves content, headers are autoset
// receives four parameters, it's low-level function, instead you can use .ServeFile(string)
//
// You can define your own "Content-Type" header also, after this function call
func (ctx *Context) ServeContent(content io.ReadSeeker, filename string, modtime time.Time, gzipCompression bool) error {
	h := ctx.ResponseWriter.Header()
	if t, err := time.Parse(ctx.q.TimeFormat, ctx.RequestHeader(ifModifiedSince)); err == nil && modtime.Before(t.Add(1*time.Second)) {
		h.Del(contentType)
		h.Del(contentLength)
		ctx.ResponseWriter.WriteHeader(StatusNotModified)
		return nil
	}

	h.Set(contentType, typeByExtension(filename))
	h.Set(lastModified, modtime.UTC().Format(ctx.q.TimeFormat))

	var out io.Writer
	if gzipCompression && ctx.clientAllowsGzip() {
		h.Add(varyHeader, acceptEncodingHeader)
		h.Set(contentEncodingHeader, "gzip")
		gzipWriter := AcquireGzip(ctx.ResponseWriter)
		defer ReleaseGzip(gzipWriter)
		out = gzipWriter
	} else {
		out = ctx.ResponseWriter

	}
	_, err := io.Copy(out, content)

	return errServeContent.With(err)
}

// ServeFile serves a view file, to send a file ( zip for example) to the client with other filename
// you should use the SendFile(serverfilename,clientfilename)
// it just calls the http.ServeFile
//
// Use this when you want to serve css/js/... files to the client, for bigger files and 'force-download' use the SendFile
func (ctx *Context) ServeFile(filename string) {
	http.ServeFile(ctx.ResponseWriter, ctx.Request, filename)
}

// ServeFileContent serves a view file, to send a file ( zip for example) to the client you should use the SendFile(serverfilename,clientfilename)
// receives two parameters
// filename/path (string)
// gzipCompression (bool)
//
// You can define your own "Content-Type" header also, after this function call
// Note: this was the previous implementation of ctx.ServeFile
//
func (ctx *Context) ServeFileContent(filename string, gzipCompression bool) error {
	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("%d", 404)
	}
	defer f.Close()
	fi, _ := f.Stat()
	if fi.IsDir() {
		filename = path.Join(filename, "index.html")
		f, err = os.Open(filename)
		if err != nil {
			return fmt.Errorf("%d", 404)
		}
		fi, _ = f.Stat()
	}
	return ctx.ServeContent(f, fi.Name(), fi.ModTime(), gzipCompression)
}

// SendFile sends file for force-download to the client
//
// Use this instead of ServeFile to 'force-download' bigger files to the client
func (ctx *Context) SendFile(filename string, destinationName string) {
	ctx.ServeFile(filename)
	ctx.ResponseWriter.Header().Set(contentDisposition, "attachment;filename="+destinationName)
}

/* Storage */

// Get returns the user's value from a key
// if doesn't exists returns nil
func (ctx *Context) Get(key string) interface{} {
	return ctx.values.Get(key)
}

// GetFmt returns a value which has this format: func(format string, args ...interface{}) string
// if doesn't exists returns nil
func (ctx *Context) GetFmt(key string) func(format string, args ...interface{}) string {
	if v, ok := ctx.Get(key).(func(format string, args ...interface{}) string); ok {
		return v
	}
	return func(format string, args ...interface{}) string { return "" }

}

// GetString same as Get but returns the value as string
// if nothing founds returns empty string ""
func (ctx *Context) GetString(key string) string {
	if v, ok := ctx.Get(key).(string); ok {
		return v
	}

	return ""
}

// GetInt same as Get but returns the value as int
// if nothing founds returns -1
func (ctx *Context) GetInt(key string) int {
	if v, ok := ctx.Get(key).(int); ok {
		return v
	}

	return -1
}

// Set sets a value to a key in the values map
func (ctx *Context) Set(key string, value interface{}) {
	ctx.values.Set(key, value)
}

// VisitAllCookies takes a visitor which loops on each (request's) cookie key and value
//
// Note: the method ctx.Request.Header.VisitAllCookie by fasthttp, has a strange bug which I cannot solve at the moment.
// This is the reason which this function exists and should be used instead of fasthttp's built'n.
func (ctx *Context) VisitAllCookies(visitor func(key string, value string)) {
	for _, c := range ctx.Request.Cookies() {
		cookie := *c
		visitor(cookie.Name, cookie.Value)
	}
}

// GetCookie returns cookie's value by it's name
// returns empty string if nothing was found
func (ctx *Context) GetCookie(name string) string {
	c, err := ctx.Request.Cookie(name)
	if err == nil {
		return c.Value
	}
	return ""
}

// AddCookie adds a cookie
func (ctx *Context) AddCookie(cookie *http.Cookie) {
	if v := cookie.String(); v != "" {
		ctx.ResponseWriter.Header().Add("Set-Cookie", v)
	}
}

// AddCookieKV adds a cookie, receives the name(string) and the value(string) of a new cookie
func (ctx *Context) AddCookieKV(name string, value string) {
	cookie := AcquireCookie()
	cookie.Name = name
	cookie.Value = value
	ctx.AddCookie(cookie)
	ReleaseCookie(cookie)
}

// RemoveCookie deletes a cookie by it's name/key
func (ctx *Context) RemoveCookie(name string) {
	c, err := ctx.Request.Cookie(name)
	if err != nil {
		return
	}

	c.Expires = CookieExpireDelete
	c.MaxAge = -1
	c.Value = ""
	c.Path = "/"
	ctx.AddCookie(c)
}

// GetFlashes returns all the flash messages for available for this request
func (ctx *Context) GetFlashes() map[string]string {
	// if already taken at least one time, this will be filled
	if messages := ctx.Get(flashMessagesStoreContextKey); messages != nil {
		if m, isMap := messages.(map[string]string); isMap {
			return m
		}
	} else {
		flashMessageFound := false
		// else first time, get all flash cookie keys(the prefix will tell us which is a flash message), and after get all one-by-one using the GetFlash.
		flashMessageCookiePrefixLen := len(flashMessageCookiePrefix)
		ctx.VisitAllCookies(func(key string, value string) {
			if len(key) > flashMessageCookiePrefixLen {
				if key[0:flashMessageCookiePrefixLen] == flashMessageCookiePrefix {
					unprefixedKey := key[flashMessageCookiePrefixLen:]
					_, err := ctx.GetFlash(unprefixedKey) // this func will add to the list (flashMessagesStoreContextKey) also
					if err == nil {
						flashMessageFound = true
					}
				}

			}
		})
		// if we found at least one flash message then re-execute this function to return the list
		if flashMessageFound {
			return ctx.GetFlashes()
		}
	}
	return nil
}

func (ctx *Context) decodeFlashCookie(name string) (string, string) {
	cookieName := flashMessageCookiePrefix + name
	cookie, err := ctx.Request.Cookie(name)
	if err != nil {
		return "", ""
	}
	cookieValue, err := decodeCookieValue(cookie.Value)
	if err != nil {
		return "", ""
	}
	return cookieName, cookieValue
}

// GetFlash get a flash message by it's key
// returns the value as string and an error
//
// if the cookie doesn't exists the string is empty and the error is filled
// after the request's life the value is removed
func (ctx *Context) GetFlash(key string) (string, error) {

	// first check if flash exists from this request's lifetime, if yes return that else continue to get the cookie
	storeExists := false

	if messages := ctx.Get(flashMessagesStoreContextKey); messages != nil {
		m, isMap := messages.(map[string]string)
		if !isMap {
			return "", fmt.Errorf("Flash store is not a map[string]string. This suppose will never happen, please report this bug.")
		}
		storeExists = true // in order to skip the check later
		for k, v := range m {
			if k == key && v != "" {
				return v, nil
			}
		}
	}

	cookieName, cookieValue := ctx.decodeFlashCookie(key)
	if cookieValue == "" {
		return "", errFlashNotFound.Return()
	}
	// store this flash message to the lifetime request's local storage,
	// I choose this method because no need to store it if not used at all
	if storeExists {
		ctx.Get(flashMessagesStoreContextKey).(map[string]string)[key] = cookieValue
	} else {
		flashStoreMap := make(map[string]string)
		flashStoreMap[key] = cookieValue
		ctx.Set(flashMessagesStoreContextKey, flashStoreMap)
	}

	//remove the real cookie, no need to have that, we stored it on lifetime request
	ctx.RemoveCookie(cookieName)
	return cookieValue, nil
	//it should'b be removed until the next reload, so we don't do that: ctx.Request.Header.SetCookie(key, "")

}

// SetFlash sets a flash message, accepts 2 parameters the name(string) and the value(string)
// the value will be available on the NEXT request
func (ctx *Context) SetFlash(name string, value string) {
	//c := AcquireCookie()
	c := &http.Cookie{}
	c.Name = flashMessageCookiePrefix + name
	c.Value = encodeCookieValue(value)
	c.Path = "/"
	c.HttpOnly = true
	ctx.AddCookie(c)
	//ReleaseCookie(c)
}

// Session returns the current session
func (ctx *Context) Session() SessionStore {
	if ctx.q.sessions == nil { // this should never return nil but FOR ANY CASE, on future changes.
		return nil
	}

	if ctx.session == nil {
		ctx.session = ctx.q.sessions.start(ctx)
	}
	return ctx.session
}

// SessionDestroy destroys the whole session, calls the provider's destroy and remove the cookie
func (ctx *Context) SessionDestroy() {
	if sess := ctx.Session(); sess != nil {
		ctx.q.sessions.destroy(ctx)
	}

}

// Log calls Printf to print to the logger.
// Arguments are handled in the manner of fmt.Printf.
func (ctx *Context) Log(format string, v ...interface{}) {
	ctx.q.Logger.Printf(format+"\n", v...)
}
