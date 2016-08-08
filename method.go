package q

import "strings"

// Common HTTP methods.
//
// Unless otherwise noted, these are defined in RFC 7231 section 4.3.
const (
	MethodGet     = "GET"
	MethodHead    = "HEAD"
	MethodPost    = "POST"
	MethodPut     = "PUT"
	MethodPatch   = "PATCH" // RFC 5789
	MethodDelete  = "DELETE"
	MethodConnect = "CONNECT"
	MethodOptions = "OPTIONS"
	MethodTrace   = "TRACE"
)

var (
	// MethodsAll "GET", "POST", "PUT", "DELETE", "CONNECT", "HEAD", "PATCH", "OPTIONS", "TRACE"
	MethodsAll = [...]string{MethodGet, MethodPost, MethodPut, MethodDelete, MethodConnect, MethodHead, MethodPatch, MethodOptions, MethodTrace}

	// methodGetBytes "GET"
	methodGetBytes = []byte(MethodGet)
	// methodPostBytes "POST"
	methodPostBytes = []byte(MethodPost)
	// methodPutBytes "PUT"
	methodPutBytes = []byte(MethodPut)
	// methodDeleteBytes "DELETE"
	methodDeleteBytes = []byte(MethodDelete)
	// methodConnectBytes "CONNECT"
	methodConnectBytes = []byte(MethodConnect)
	// methodHeadBytes "HEAD"
	methodHeadBytes = []byte(MethodHead)
	// methodPatchBytes "PATCH"
	methodPatchBytes = []byte(MethodPatch)
	// methodOptionsBytes "OPTIONS"
	methodOptionsBytes = []byte(MethodOptions)
	// methodTraceBytes "TRACE"
	methodTraceBytes = []byte(MethodTrace)
)

// parseMethod receives a method, if empty or invalid method then returns "" else returns the receiver
func parseMethod(s string) string {
	s = strings.ToUpper(s)
	for _, v := range MethodsAll {
		if v == s {
			return v
		}
	}

	return ""
}
