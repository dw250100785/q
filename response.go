package q

import (
	"strings"

	"github.com/kataras/q/errors"
	"github.com/kataras/q/response/data"
	"github.com/kataras/q/response/json"
	"github.com/kataras/q/response/jsonp"
	"github.com/kataras/q/response/markdown"
	"github.com/kataras/q/response/text"
	"github.com/kataras/q/response/xml"
)

type (
	// Response engines' configuration
	Response struct {
		Engine      ResponseEngine
		Name        string
		ContentType string
	}
	// Responses a slice of Response engines' configuration
	Responses []Response
)

func (responses Responses) loadTo(r *responseEngines) {
	for _, res := range responses {
		r.add(res.Engine, res.ContentType)(res.Name)
	}
	// if no response engines setted for the default content-types
	// then add them
	for _, ctype := range defaultResponseKeys {
		if rengine := r.getBy(ctype); rengine == nil {
			// if not exists
			switch ctype {
			case contentText:
				r.add(text.New(), ctype)
			case contentBinary:
				r.add(data.New(), ctype)
			case contentJSON:
				r.add(json.New(), ctype)
			case contentJSONP:
				r.add(jsonp.New(), ctype)
			case contentXML:
				r.add(xml.New(), ctype)
			case contentMarkdown:
				r.add(markdown.New(), ctype)
			}
		}
	}
}

type (
	// ResponseEngine is the interface which all response engines should implement to send responses
	ResponseEngine interface {
		Response(interface{}, ...map[string]interface{}) ([]byte, error)
	}
	// ResponseEngineFunc is the alternative way to implement a ResponseEngine using a simple function
	ResponseEngineFunc func(interface{}, ...map[string]interface{}) ([]byte, error)

	// responseEngineMap is a wrapper with key (content type or name) values(engines) for the registered response engine
	// it contains all response engines for a specific contentType and two functions, render and toString
	// these will be used by the Q' context and Q' ResponseString, yes like TemplateToString
	// it's an internal struct, no need to be exported and return that on registration,
	// because the two top funcs will be easier to use by the user/dev for multiple engines
	responseEngineMap struct {
		values []ResponseEngine
		// this is used in order to the wrapper to be gettable by the responseEngines iteral,
		// if key is not a $content/type and contentType is not changed by the user/dev then the text/plain will be sent to the client
		key         string
		contentType string
	}
)

var (
	// markdown is custom type, used inside Q to initialize the defaults response engines if no other engine registered with these keys
	defaultResponseKeys = [...]string{contentText, contentXML, contentBinary, contentJSON, contentJSONP, contentMarkdown}
)

// Response returns  a response to the client(request's body content)
func (r ResponseEngineFunc) Response(obj interface{}, options ...map[string]interface{}) ([]byte, error) {
	return r(obj, options...)
}

var errNoResponseEngineFound = errors.New("No response engine found")

// on context: Send(contentType string, obj interface{}, ...options)

func (r *responseEngineMap) add(engine ResponseEngine) {
	r.values = append(r.values, engine)
}

// the gzip and charset options are built'n with Q
func (r *responseEngineMap) render(ctx *Context, obj interface{}, options ...map[string]interface{}) error {

	if r == nil {
		//render, but no response engine registered, this caused by context.RenderWithStatus, and responseEngines. getBy
		return errNoResponseEngineFound.Return()
	}

	var finalResult []byte

	for i, n := 0, len(r.values); i < n; i++ {
		result, err := r.values[i].Response(obj, options...)
		if err != nil { // fail on first the first error
			return err
		}
		finalResult = append(finalResult, result...)
	}

	gzipEnabled := ctx.q.Gzip
	charset := ctx.q.Charset
	if len(options) > 0 {
		gzipEnabled = getGzipOption(ctx, options[0]) // located to the template.go below the RenderOptions
		if chs := getCharsetOption(options[0]); chs != "" {
			charset = chs
		}
	}
	ctype := r.contentType

	if r.contentType != contentBinary { // set the charset only on non-binary data
		ctype += "; charset=" + charset
	}
	ctx.SetContentType(ctype)

	if gzipEnabled {
		if _, err := ctx.WriteGzip(finalResult); err != nil {
			return err
		}
		ctx.ResponseWriter.Header().Add("Content-Encoding", "gzip")
	} else {
		ctx.Write(finalResult)
	}

	return nil
}

func (r *responseEngineMap) toString(obj interface{}, options ...map[string]interface{}) (string, error) {
	if r == nil {
		//render, but no response engine registered, this caused by context.RenderWithStatus, and responseEngines. getBy
		return "", errNoResponseEngineFound.Return()
	}
	var finalResult []byte
	for i, n := 0, len(r.values); i < n; i++ {
		result, err := r.values[i].Response(obj, options...)
		if err != nil {
			return "", err
		}
		finalResult = append(finalResult, result...)
	}
	return string(finalResult), nil
}

type responseEngines struct {
	engines []*responseEngineMap
}

// add accepts a simple response engine with its content type or key, key should not contains a dot('.').
// if key is a content type then it's the content type, but if it not, set the content type from the returned function,
// if it not called/changed then the default content type text/plain will be used.
// different content types for the same key will produce bugs, as it should!
// one key has one content type but many response engines ( one to many)
// note that the func should be used on the same call
func (r *responseEngines) add(engine ResponseEngine, forContentTypesOrKeys ...string) func(string) {
	if r.engines == nil {
		r.engines = make([]*responseEngineMap, 0)
	}

	var engineMap *responseEngineMap
	for _, key := range forContentTypesOrKeys {
		if strings.IndexByte(key, '.') != -1 { // the dot is not allowed as key
			continue // skip this engine
		}

		defaultCtypeAndKey := contentText
		if len(key) == 0 {
			//if empty key, then set it to text/plain
			key = defaultCtypeAndKey
		}

		engineMap = r.getBy(key)
		if engineMap == nil {

			ctype := defaultCtypeAndKey
			if strings.IndexByte(key, slashByte) != -1 { // pure check, but developer should know the content types at least.
				// we have 'valid' content type
				ctype = key
			}
			// the context.Markdown works without it but with .Render we will have problems without this:
			if key == contentMarkdown { // remember the text/markdown is just a custom internal Q content type, which in reallity renders html
				ctype = contentHTML
			}
			engineMap = &responseEngineMap{values: make([]ResponseEngine, 0), key: key, contentType: ctype}
			r.engines = append(r.engines, engineMap)
		}
		engineMap.add(engine)
	}

	return func(theContentType string) {
		// and this
		if theContentType == contentMarkdown {
			theContentType = contentHTML
		}

		engineMap.contentType = theContentType
	}

}

func (r *responseEngines) getBy(key string) *responseEngineMap {
	for i, n := 0, len(r.engines); i < n; i++ {
		if r.engines[i].key == key {
			return r.engines[i]
		}

	}
	return nil
}
