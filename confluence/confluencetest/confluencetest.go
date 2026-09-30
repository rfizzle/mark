// Package confluencetest runs a fake Confluence Cloud site for tests.
package confluencetest

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/kovetskiy/mark/v16/confluence"
)

// BaseURL is the Cloud base URL of the APIs returned by NewAPI. Its host is
// on atlassian.net, so IsCloud reports true.
const BaseURL = "https://example.atlassian.net/wiki"

// NewAPI starts a test server running handler and returns an API for BaseURL
// whose requests are all sent to that server. The server is closed when the
// test ends.
func NewAPI(t testing.TB, handler http.Handler) *confluence.API {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	target, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse test server URL: %v", err)
	}

	client := &http.Client{Transport: rewriteHost{target: target, next: server.Client().Transport}}
	return confluence.NewAPIWithClient(BaseURL, "user", "token", client)
}

// ReadOnly wraps handler and fails the test if a request other than GET
// reaches it.
func ReadOnly(t testing.TB, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected write request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "writes not allowed", http.StatusMethodNotAllowed)
			return
		}
		handler.ServeHTTP(w, r)
	})
}

// rewriteHost sends requests to target, keeping their path and query.
type rewriteHost struct {
	target *url.URL
	next   http.RoundTripper
}

func (rt rewriteHost) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Scheme = rt.target.Scheme
	req.URL.Host = rt.target.Host
	req.Host = rt.target.Host
	return rt.next.RoundTrip(req)
}
