package q

import (
	"net/http"
	"testing"

	"github.com/gavv/httpexpect"
)

// Tester configuration
type Tester struct {
	Host        string
	ExplicitURL bool
	Debug       bool
}

// NewTester Prepares and returns a new test framework based on the api
// is useful when you need to have more than one test framework for the same Q instance, otherwise you can use the yourQInstance.Tester(t *testing.T)
//
// receives two parameters
// the first is the Q instnace
// and the second is the testing.T
// returns a new *httpexpect.Expect
func NewTester(q *Q, t *testing.T) *httpexpect.Expect {

	baseURL := ""
	host := q.Tester.Host
	if host == "" {
		host = q.Host
	}

	if host == "" {
		host = "q.go:1993" // for 'virtual' host if user doesn't specify a Host from Tester configuration or Q iteral
	}

	if !q.Tester.ExplicitURL {
		scheme := schemeHTTP
		if (q.CertFile != "" && q.KeyFile != "") || parsePort(q.Host) == 443 || q.Host == ":https" {
			q.Scheme = schemeHTTPS
		}
		baseURL = scheme + host
	}

	// edw prosoxh giati borei to Handler na mhn exei ginei set akoma, ara to tester prepei na kaleite mono sto q.Test() oxi apo mono tou
	if q.Request.Handler == nil {
		q.Request.build(host)
	}

	testConfiguration := httpexpect.Config{
		BaseURL: baseURL,
		Client: &http.Client{
			Transport: httpexpect.NewBinder(q),
			Jar:       httpexpect.NewJar(),
		},
		Reporter: httpexpect.NewAssertReporter(t),
	}

	if q.Tester.Debug {
		testConfiguration.Printers = []httpexpect.Printer{
			httpexpect.NewDebugPrinter(t, true),
		}

	}
	return httpexpect.WithConfig(testConfiguration)
}
