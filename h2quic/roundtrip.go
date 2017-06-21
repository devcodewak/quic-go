package h2quic

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"

	quic "github.com/phuslu/quic-go"

	"runtime"

	"golang.org/x/net/lex/httplex"
)

type roundTripCloser interface {
	http.RoundTripper
	io.Closer
}

// RoundTripper implements the http.RoundTripper interface
type RoundTripper struct {
	mutex sync.Mutex

	// DisableCompression, if true, prevents the Transport from
	// requesting compression with an "Accept-Encoding: gzip"
	// request header when the Request contains no existing
	// Accept-Encoding value. If the Transport requests gzip on
	// its own and gets a gzipped response, it's transparently
	// decoded in the Response.Body. However, if the user
	// explicitly requested gzip it is not automatically
	// uncompressed.
	DisableCompression bool

	// TLSClientConfig specifies the TLS configuration to use with
	// tls.Client. If nil, the default configuration is used.
	TLSClientConfig *tls.Config

	// DialAddr specifies an optional function for quic.DailAddr.
	// If this value is nil, it will default to net.DialAddr for the client.
	DialAddr func(hostname string, tlsConfig *tls.Config, config *quic.Config) (quic.Session, error)

	// QuicConfig is the quic.Config used for dialing new connections.
	// If nil, reasonable default values will be used.
	QuicConfig *quic.Config

	clients map[string]roundTripCloser
}

var _ roundTripCloser = &RoundTripper{}

// RoundTrip does a round trip
func (r *RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL == nil {
		closeRequestBody(req)
		return nil, errors.New("quic: nil Request.URL")
	}
	if req.URL.Host == "" {
		closeRequestBody(req)
		return nil, errors.New("quic: no Host in request URL")
	}
	if req.Header == nil {
		closeRequestBody(req)
		return nil, errors.New("quic: nil Request.Header")
	}

	if req.URL.Scheme == "https" {
		for k, vv := range req.Header {
			if !httplex.ValidHeaderFieldName(k) {
				return nil, fmt.Errorf("quic: invalid http header field name %q", k)
			}
			for _, v := range vv {
				if !httplex.ValidHeaderFieldValue(v) {
					return nil, fmt.Errorf("quic: invalid http header field value %q for key %v", v, k)
				}
			}
		}
	} else {
		closeRequestBody(req)
		return nil, fmt.Errorf("quic: unsupported protocol scheme: %s", req.URL.Scheme)
	}

	if req.Method != "" && !validMethod(req.Method) {
		closeRequestBody(req)
		return nil, fmt.Errorf("quic: invalid method %q", req.Method)
	}

	hostname := authorityAddr("https", hostnameFromRequest(req))
	return r.getClient(hostname).RoundTrip(req)
}

// CloseConnections remove clients according the net.Addr
func (r *RoundTripper) CloseConnections(f func(addr net.Addr, idle bool) bool) {
	hosts := []string{}
	for host, c := range r.clients {
		session := c.(*client).session
		if session != nil && f(session.RemoteAddr(), false) {
			session.CloseLocal(errors.New("h2quic: CloseConnections called"))
			hosts = append(hosts, host)
		}
	}
	for _, host := range hosts {
		delete(r.clients, host)
	}
}

func (r *RoundTripper) getClient(hostname string) http.RoundTripper {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.clients == nil {
		runtime.SetFinalizer(r, finalizer)
		r.clients = make(map[string]roundTripCloser)
	}

	client, ok := r.clients[hostname]
	if !ok {
		client = newClient(hostname, r.TLSClientConfig, &roundTripperOpts{DisableCompression: r.DisableCompression, DialAddr: r.DialAddr}, r.QuicConfig)
		r.clients[hostname] = client
	}
	return client
}

func finalizer(r *RoundTripper) {
	_ = r.Close()
}

// Close closes the QUIC connections that this RoundTripper has used
func (r *RoundTripper) Close() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	for _, client := range r.clients {
		if err := client.Close(); err != nil {
			return err
		}
	}
	r.clients = nil
	return nil
}

func closeRequestBody(req *http.Request) {
	if req.Body != nil {
		req.Body.Close()
	}
}

func validMethod(method string) bool {
	/*
				     Method         = "OPTIONS"                ; Section 9.2
		   		                    | "GET"                    ; Section 9.3
		   		                    | "HEAD"                   ; Section 9.4
		   		                    | "POST"                   ; Section 9.5
		   		                    | "PUT"                    ; Section 9.6
		   		                    | "DELETE"                 ; Section 9.7
		   		                    | "TRACE"                  ; Section 9.8
		   		                    | "CONNECT"                ; Section 9.9
		   		                    | extension-method
		   		   extension-method = token
		   		     token          = 1*<any CHAR except CTLs or separators>
	*/
	return len(method) > 0 && strings.IndexFunc(method, isNotToken) == -1
}

// copied from net/http/http.go
func isNotToken(r rune) bool {
	return !httplex.IsTokenRune(r)
}
