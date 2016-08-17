package q

import (
	"mime"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/kataras/q/errors"
)

// StaticHandler serves bytes, memory cached
// a good example of this is how the websocket server uses that to auto-register the javascript client side source
func StaticHandler(contentType string, content []byte) Handler {
	modtime := time.Now()
	return func(ctx *Context) {
		modtimeStr := modtime.UTC().Format(ctx.q.TimeFormat)
		if t, err := time.Parse(ctx.q.TimeFormat, ctx.RequestHeader(ifModifiedSince)); err == nil && modtime.Before(t.Add(ctx.q.StaticCacheDuration)) {
			ctx.ResponseWriter.Header().Del(contentType)
			ctx.ResponseWriter.Header().Del(contentLength)
			ctx.ResponseWriter.WriteHeader(StatusNotModified)
			return
		}
		ctx.SetContentType(contentType)
		ctx.ResponseWriter.Header().Set(lastModified, modtimeStr)
		ctx.ResponseWriter.Write(content)
	}
}

// StripPrefix returns a handler that serves HTTP requests
// by removing the given prefix from the request URL's Path
// and invoking the handler h. StripPrefix handles a
// request for a path that doesn't begin with prefix by
// replying with an HTTP 404 not found error.
func StripPrefix(prefix string, h Handler) Handler {
	if prefix == "" {
		return h
	}
	return func(ctx *Context) {
		if p := strings.TrimPrefix(ctx.Request.URL.Path, prefix); len(p) < len(ctx.Request.URL.Path) {
			ctx.Request.URL.Path = p
			h(ctx)
		} else {
			ctx.NotFound()
		}
	}
}

// File serves content binary bytes, it's the simplest form of the static entries
type File struct {
	ContentType string
	Content     []byte
}

// ParseEntry returns the converted entry, implements the EntryParser
func (s File) ParseEntry(e Entry) Entry {
	e.Head = true
	e.Handler = StaticHandler(s.ContentType, s.Content)
	return e
}

// Dir serves a directory as web resource
// accepts the Entry and the Directory (string) if Entry.Path is empty then the Directory's path is used (should be relative in order this to work)
// usage: q.Dir{ Entry{Method: "GET", Path: "/publicarea"}, Directory: "./static/myfiles", Gzip: true, StripPrefix: "/static/"}
// Note: it doesn't checks if system path is exists, use it with your own risk, otherwise you can use the http.FileServer method, which is different of what I try to do here.
type Dir struct {
	// Entry , edw 9a borouse na dexete idi ena entry kai na to kanei static se fash me ta upoloipa properties, oxi kai asximh idea..xmm
	Directory string
	Gzip      bool
	// StripPrefix is not needed in the most cases
	// for example
	// q.Entry{Method: q.MethodGet, Path: "/js", Parser: q.Dir{Directory: "./static/js"}}
	// Request to: http://localhost/js/chat.js ,it will serve it, without the StripPrefix.
	// The only reason it exists is if you have problems with the default behavior
	StripPrefix string
}

// ParseEntry returns the converted Entry, implements the EntryParser
func (s Dir) ParseEntry(e Entry) Entry {
	if s.Directory == "" {
		return e
	}
	sep := string(os.PathSeparator)
	s.Directory = strings.Replace(s.Directory, "/", sep, -1)

	reqPath := ""
	if e.Path == "" {
		reqPath = strings.Replace(s.Directory, sep, slash, -1) // replaces any \ to /
		reqPath = strings.Replace(reqPath, "//", slash, -1)    // for any case, replaces // to /
		reqPath = strings.Replace(reqPath, ".", "", -1)        // replace any dots (./mypath -> /mypath)
	} else {
		reqPath = e.Path
	}
	e.Head = true
	e.Path = reqPath + "/*file"
	h := func(ctx *Context) {
		filepath := ctx.Param("file")
		if filepath == "" {
			ctx.NotFound()
			return
		}

		spath := strings.Replace(filepath, slash, sep, -1)
		spath = path.Join(s.Directory, spath)

		if !directoryExists(spath) { //yes, we check everytime, this has its bad and its good, you have an alternative, to use the http.FileServer if this behavior does't suits your case.
			ctx.NotFound()
			return
		}

		ctx.ServeFileContent(spath, s.Gzip)
	}

	if s.StripPrefix != "" {
		h = StripPrefix(s.StripPrefix, h)
	}

	e.Handler = h

	return e
}

// Favicon serves static favicon
// accepts two fields, the Entry and the Favicon system path
// Favicon (string), declare the system directory path of the __.ico
// Entry.Path (string), it's the route's path, by default this is the "/favicon.ico" because some browsers tries to get this by default first,
// you can declare your own path if you have more than one favicon (desktop, mobile and so on)
//
// this func will add a route for you which will static serve the /yuorpath/yourfile.ico to the /yourfile.ico (nothing special that you can't handle by yourself)
//
// panics on error
type Favicon struct {
	Favicon string
}

// ParseEntry returns the favicon entry, implements the EntryParser
func (favicon Favicon) ParseEntry(e Entry) Entry {
	favPath := favicon.Favicon
	f, err := os.Open(favPath)
	if err != nil {
		panic(errDirectoryFileNotFound.Format(favPath, err.Error()))
	}
	defer f.Close()
	fi, _ := f.Stat()
	if fi.IsDir() { // if it's dir the try to get the favicon.ico
		fav := path.Join(favPath, "favicon.ico")
		f, err = os.Open(fav)
		if err != nil {
			//we try again with .png
			favicon.Favicon = path.Join(favPath, "favicon.png")
			return favicon.ParseEntry(e)
		}
		favPath = fav
		fi, _ = f.Stat()
	}

	cType := typeByExtension(favPath)
	// copy the bytes here in order to cache and not read the ico on each request.
	cacheFav := make([]byte, fi.Size())
	if _, err = f.Read(cacheFav); err != nil {
		panic(errDirectoryFileNotFound.Format(favPath, "Couldn't read the data bytes for Favicon: "+err.Error()))
	}

	h := func(ctx *Context) {
		modtime := fi.ModTime().UTC().Format(ctx.q.TimeFormat)
		if t, err := time.Parse(ctx.q.TimeFormat, ctx.RequestHeader(ifModifiedSince)); err == nil && fi.ModTime().Before(t.Add(ctx.q.StaticCacheDuration)) {
			ctx.ResponseWriter.Header().Del(contentType)
			ctx.ResponseWriter.Header().Del(contentLength)
			ctx.ResponseWriter.WriteHeader(StatusNotModified)
			return
		}

		ctx.ResponseWriter.Header().Set(contentType, cType)
		ctx.ResponseWriter.Header().Set(lastModified, modtime)
		ctx.ResponseWriter.Write(cacheFav)
	}

	e.Handler = h

	if e.Path == "" {
		e.Path = "/favicon" + path.Ext(fi.Name()) //we could use the filename, but because standards is /favicon.ico/.png.
	}

	// for the Head http method
	e.Head = true
	if e.Method == "" && e.Method != MethodHead {
		e.Method = MethodGet
	}
	return e
}

// errDirectoryFileNotFound returns an error with message: 'Directory or file %s couldn't found. Trace: +error trace'
var errDirectoryFileNotFound = errors.New("Directory or file %s couldn't found. Trace: %s")

// DirectoryExists returns true if a directory(or file) exists, otherwise false
func directoryExists(dir string) bool {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return false
	}
	return true
}

// TypeByExtension returns the MIME type associated with the file extension ext.
// The extension ext should begin with a leading dot, as in ".html".
// When ext has no associated type, TypeByExtension returns "".
//
// Extensions are looked up first case-sensitively, then case-insensitively.
//
// The built-in table is small but on unix it is augmented by the local
// system's mime.types file(s) if available under one or more of these
// names:
//
//   /etc/mime.types
//   /etc/apache2/mime.types
//   /etc/apache/mime.types
//
// On Windows, MIME types are extracted from the registry.
//
// Text types have the charset parameter set to "utf-8" by default.
func typeByExtension(fullfilename string) (t string) {
	ext := filepath.Ext(fullfilename)
	//these should be found by the windows(registry) and unix(apache) but on windows some machines have problems on this part.
	if t = mime.TypeByExtension(ext); t == "" {
		// no use of map here because we will have to lock/unlock it, by hand is better, no problem:
		if ext == ".json" {
			t = "application/json"
		} else if ext == ".zip" {
			t = "application/zip"
		} else if ext == ".3gp" {
			t = "video/3gpp"
		} else if ext == ".7z" {
			t = "application/x-7z-compressed"
		} else if ext == ".ace" {
			t = "application/x-ace-compressed"
		} else if ext == ".aac" {
			t = "audio/x-aac"
		} else if ext == ".ico" { // for any case
			t = "image/x-icon"
		} else {
			t = contentBinary
		}
	}
	return
}

// GetParentDir returns the parent directory(string) of the passed targetDirectory (string)
func getParentDir(targetDirectory string) string {
	lastSlashIndex := strings.LastIndexByte(targetDirectory, os.PathSeparator)
	//check if the slash is at the end , if yes then re- check without the last slash, we don't want /path/to/ , we want /path/to in order to get the /path/ which is the parent directory of the /path/to
	if lastSlashIndex == len(targetDirectory)-1 {
		lastSlashIndex = strings.LastIndexByte(targetDirectory[0:lastSlashIndex], os.PathSeparator)
	}

	parentDirectory := targetDirectory[0:lastSlashIndex]
	return parentDirectory
}
