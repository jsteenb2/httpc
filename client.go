package httpc

import (
	"net/http"
	"strings"
)

// Doer is an abstraction around a http client.
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

// Client is the httpc client. The client sets the default backoff and encode func
// on the request that are created when making an http call. Those defaults can be
// overridden in the request builder.
type Client struct {
	baseURL string
	doer    Doer

	authFn   AuthFn
	encodeFn EncodeFn
	backoff  BackoffOptFn
}

// New returns a new client.
func New(doer Doer, opts ...ClientOptFn) *Client {
	c := Client{
		doer:     doer,
		encodeFn: JSONEncode(),
		backoff:  NewStopBackoff(),
	}

	for _, o := range opts {
		c = o(c)
	}
	return &c
}

// DELETE makes a delete request.
func (c *Client) DELETE(addr string) *Request {
	return c.Req(http.MethodDelete, addr)
}

// GET makes a get request.
func (c *Client) GET(addr string) *Request {
	return c.Req(http.MethodGet, addr)
}

// PATCH makes a patch request.
func (c *Client) PATCH(addr string) *Request {
	return c.Req(http.MethodPatch, addr)
}

// POST makes a post request.
func (c *Client) POST(addr string) *Request {
	return c.Req(http.MethodPost, addr)
}

// PUT makes a put request.
func (c *Client) PUT(addr string) *Request {
	return c.Req(http.MethodPut, addr)
}

// Req makes an http request.
func (c *Client) Req(method, addr string) *Request {
	address := c.baseURL + addr
	if !strings.HasSuffix(c.baseURL, "/") && !strings.HasPrefix(addr, "/") {
		address = c.baseURL + "/" + addr
	}
	return &Request{
		Method:   method,
		Addr:     address,
		doer:     c.doer,
		authFn:   c.authFn,
		encodeFn: c.encodeFn,
		backoff:  c.backoff,
	}
}
