package q

import (
	"strconv"
	"strings"
)

// Proxy , useful when you use the main q with https:// and want to redirect all http:// requests to the main https://
// or the opossite
// starts a new server with a single handler to do the redirection
//
// it's a blocking function, which you can call with a 'go routine'
//
// note: It is not a real proxy, use it only when you want to redirect from one host to another, it's very pure implementation but it does the job
// note: no security checks for http to https, if anything special needed, it can be handled by developer on the q-instance 'Begin handlers(middleware) field.
func Proxy(fakeHost string, redirectSchemeAndHost string) {
	Q{
		Host: fakeHost,
		Request: Request{
			Handler: func(ctx *Context) {
				// override the handler and redirect all requests to this addr
				redirectTo := redirectSchemeAndHost
				fakehost := ctx.Request.Host
				path := ctx.Path()                     // problem with subdomains mysubsubdomain.mysubdomain.mydomain.com this will redirect on https://127.0.0.1 and that's a problem.
				if strings.Count(fakehost, ".") >= 3 { // propably a subdomain, pure check but doesn't matters don't worry
					if sufIdx := strings.LastIndexByte(fakehost, '.'); sufIdx > 0 {
						// check if the last part is a number instead of .com/.gr...
						// if it's number then it's propably is 0.0.0.0 or 127.0.0.1... so it shouldn' use  subdomain
						if _, err := strconv.Atoi(fakehost[sufIdx+1:]); err != nil {
							// it's not number then process the try to parse the subdomain
							redirectScheme := parseScheme(redirectSchemeAndHost)
							realHost := strings.Replace(redirectSchemeAndHost, redirectScheme, "", 1)
							redirectHost := strings.Replace(ctx.Request.Host, fakeHost, realHost, 1)
							redirectTo = redirectScheme + redirectHost + path
							ctx.Redirect(redirectTo, StatusMovedPermanently)
							return
						}
					}
				}
				if path != "/" {
					redirectTo += path
				}
				ctx.Redirect(redirectTo, StatusMovedPermanently)
			},
		},
	}.Go()
}
