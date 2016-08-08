package q

import (
	"crypto/tls"
	"net"
	"os"
	"strconv"
	"strings"

	"time"

	"github.com/kataras/q/errors"

	"github.com/q-contrib/letsencrypt"
)

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// ----------------------------------Addr-----------------------------------------------
// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

// Default values for base Server conf
const (
	// defaultServerAddrHostname returns the default hostname which is 0.0.0.0
	defaultServerAddrHostname = "0.0.0.0"
	// defaultServerAddrPort returns the default port which is 8080
	defaultServerAddrPort = 8080
)

var (
	// defaultServerAddr the default server addr which is: 0.0.0.0:8080
	defaultServerAddr = defaultServerAddrHostname + ":" + strconv.Itoa(defaultServerAddrPort)
)

// parseHost tries to convert a given string to an address which is compatible with net.Listener and server
func parseHost(addr string) string {
	// check if addr has :port, if not do it +:80 ,we need the hostname for many cases
	a := addr
	if a == "" {
		// check for os environments
		if oshost := os.Getenv("ADDR"); oshost != "" {
			a = oshost
		} else if oshost := os.Getenv("HOST"); oshost != "" {
			a = oshost
		} else if oshost := os.Getenv("HOSTNAME"); oshost != "" {
			a = oshost
			// check for port also here
			if osport := os.Getenv("PORT"); osport != "" {
				a += ":" + osport
			}
		} else if osport := os.Getenv("PORT"); osport != "" {
			a = ":" + osport
		} else {
			a = defaultServerAddr
		}
	}
	if portIdx := strings.IndexByte(a, ':'); portIdx == 0 {
		if a[portIdx:] == ":https" {
			a = defaultServerAddrHostname + ":443"
		} else {
			// if contains only :port	,then the : is the first letter, so we dont have setted a hostname, lets set it
			a = defaultServerAddrHostname + a
		}
	}

	/* changed my mind, don't add 80, this will cause problems on unix listeners, and it's not really necessary because we take the port using parsePort
	if portIdx := strings.IndexByte(a, ':'); portIdx < 0 {
		// missing port part, add it
		a = a + ":80"
	}*/

	return a
}

// parseHostname receives an addr of form host[:port] and returns the hostname part of it
// ex: localhost:8080 will return the `localhost`, mydomain.com:8080 will return the 'mydomain'
func parseHostname(addr string) string {
	idx := strings.IndexByte(addr, ':')
	if idx == 0 {
		// only port, then return 0.0.0.0
		return "0.0.0.0"
	} else if idx > 0 {
		return addr[0:idx]
	}
	// it's already hostname
	return addr
}

// parsePort receives an addr of form host[:port] and returns the port part of it
// ex: localhost:8080 will return the `8080`, mydomain.com will return the '80'
func parsePort(addr string) int {
	if portIdx := strings.IndexByte(addr, ':'); portIdx != -1 {
		afP := addr[portIdx+1:]
		p, err := strconv.Atoi(afP)
		if err == nil {
			return p
		} else if afP == "https" { // it's not number, check if it's :https
			return 443
		}
	}
	return 80
}

const schemeHTTPS = "https://"
const schemeHTTP = "http://"

func parseScheme(domain string) string {
	// pure check
	if strings.HasPrefix(domain, schemeHTTPS) || parsePort(domain) == 443 {
		return schemeHTTPS
	}
	return schemeHTTP
}

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// ----------------------------------ServerListener-------------------------------------
// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

// Errors introduced by listener.
var (
	errProtocolNotSupported = errors.New("The protocol: %s is not supported for address: %s.")
	errProtocolUnix         = errors.New("Use newUNIXListener instead")
	errParseTLS             = errors.New("Couldn't load TLS, certFile=%q, keyFile=%q. Trace: %s")
	errRemoveUnix           = errors.New("Unexpected error when trying to remove unix socket file. Addr: %s. Trace: %s")
	errChmod                = errors.New("Cannot chmod %#o for %q. Trace: %s")
)

func newListener(protocol string, addr string) (net.Listener, error) {
	if protocol != "tcp4" && protocol != "tcp6" {
		return nil, errProtocolNotSupported.Format(protocol, addr)
	}
	if protocol == "unix" {
		return nil, errProtocolUnix.Return()
	}

	ln, err := net.Listen(protocol, addr)
	if err != nil {
		return nil, err
	}

	return ln, err
}

func newTLSListener(ln net.Listener, certFile, keyFile string) (net.Listener, error) {

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, errParseTLS.Format(certFile, keyFile, err)
	}

	tlsConfig := &tls.Config{
		Certificates:             []tls.Certificate{cert},
		PreferServerCipherSuites: true,
	}
	return tls.NewListener(ln, tlsConfig), nil
}

func newTLSListenerWithConfig(ln net.Listener, tlsCfg *tls.Config) (net.Listener, error) {
	return tls.NewListener(ln, tlsCfg), nil
}

func newUNIXListener(addr string, mode os.FileMode) (net.Listener, error) {
	if errOs := os.Remove(addr); errOs != nil && !os.IsNotExist(errOs) {
		return nil, errRemoveUnix.Format(addr, errOs.Error())
	}

	listener, err := net.Listen("unix", addr)

	if err != nil {
		return nil, err
	}

	if err = os.Chmod(addr, mode); err != nil {
		return nil, errChmod.Format(mode, addr, err.Error())
	}
	return listener, nil
}

type (
	// ServerBase created to make use of different type of todays and future servers which can be binded to the q without performance loses
	// last minite I changed my mind and q is only net/http web framework.
	//
	// I changed my mind because it would be difficult to help developers on each side (net/http) & fasthttp on the same framework
	ServerBase interface {
		Serve(net.Listener) error
	}

	// ServerListener TOOD:
	ServerListener struct {
		base     ServerBase
		listener net.Listener
		host     string // the full hostname:port
		hostname string
		port     int
	}

	// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
	// connections. It's used by ListenAndServe and ListenAndServeTLS so
	// dead TCP connections (e.g. closing laptop mid-download) eventually
	// go away.
	tcpKeepAliveListener struct {
		*net.TCPListener
	}
)

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(2 * time.Minute)
	return tc, nil
}

// newServerListener TODO:
func newServerListener(underlineServer ServerBase) *ServerListener {
	return &ServerListener{base: underlineServer}
}

// this was my first idea, to be able to use net/http and fasthttp inside this framework
// without lose any performance between conversions, this is a good idea, I keep that in comments for anyone who want to see how it can happens.
/*
// NewFasthttpServer TODO:
func NewFasthttpServer(Handler ServerHandler) *ServerListener {
	cpool := &contextPool{}
	underlineServer := &fasthttp.Server{
		Handler: func(reqCtx *fasthttp.RequestCtx) { //TOOD: here the mux
			ctx := cpool.acquire(reqCtx)
			Handler(ctx)
			cpool.release(ctx)
		}}

	return NewServerListener(underlineServer)
}

func (cp *contextPool) acquireFasthttp(reqCtx *fasthttp.RequestCtx) *context {
	v := cp.pool.Get()
	var ctx *context
	if v == nil {
		ctx = &context{
			RequestCtx: reqCtx,
		}
		//todo: ctx.framework = ... or ctx. server = ... an to valoume mesa ston server to contextPool, tha doume..
	} else {
		ctx = v.(*context)
	}
	//todos:
	// ctx.Params = ctx.Params[0:0]
	// ctx.Middleware = nil
	// ctx.sessions = nil
	// ctx.RequestCtx = nil
	return ctx
}
*/

func (s *ServerListener) setHost(addr string) string {
	h := parseHost(addr)
	s.hostname = parseHostname(h)
	s.port = parsePort(h)
	s.host = h
	return s.host
}

// Serve Starts the server on a particular listener
// calls the underline's server's Base.Serve:
func (s *ServerListener) Serve(ln net.Listener) error {
	s.listener = ln

	if s.host == "" {
		s.setHost(ln.Addr().String())
	}

	return s.base.Serve(ln)
}

// Listen start & listen to the server
// form of 'addr' is host:port
func (s *ServerListener) Listen(addr string) error {
	addr = s.setHost(addr)
	ln, err := newListener("tcp4", addr)
	if err != nil {
		return err
	}

	return s.Serve(tcpKeepAliveListener{ln.(*net.TCPListener)})
}

// ListenTLSManual start & listen to the server using provided SSL certification file and key file system paths
// form of 'addr' is host:port
func (s *ServerListener) ListenTLSManual(addr string, certFile string, keyFile string) error {
	addr = s.setHost(addr)
	tcpLn, err := newListener("tcp4", addr)
	if err != nil {
		return err
	}

	ln, err := newTLSListener(tcpLn, certFile, keyFile)
	if err != nil {
		return err
	}

	return s.Serve(tcpKeepAliveListener{ln.(*net.TCPListener)})
}

// ListenTLS start & listen to the server using automatic SSL
// form of 'addr' is host:port
func (s *ServerListener) ListenTLS(addr string) error {
	addr = s.setHost(addr)
	tcpLn, err := newListener("tcp4", addr)
	if err != nil {
		return err
	}

	var m letsencrypt.Manager
	if err = m.CacheFile("letsencrypt.cache"); err != nil {
		return err
	}

	tlsConfig := &tls.Config{GetCertificate: m.GetCertificate}
	ln, err := newTLSListenerWithConfig(tcpKeepAliveListener{tcpLn.(*net.TCPListener)}, tlsConfig)
	if err != nil {
		return err
	}

	return s.Serve(ln)
}

// ListenUNIX start & listen to the server using a 'socket file', only for unix
func (s *ServerListener) ListenUNIX(addr string, mode os.FileMode) error {
	addr = s.setHost(addr)
	ln, err := newUNIXListener(addr, mode)
	if err != nil {
		return err
	}

	return s.Serve(ln)
}

// Close terminates the server
// calls the listener's Close
func (s *ServerListener) Close() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}
