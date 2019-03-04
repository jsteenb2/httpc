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

	headers []kvPair

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

// Connect makes a connect http request.
func (c *Client) Connect(addr string) *Request {
	return c.Req(http.MethodConnect, addr)
}

// Delete makes a delete http request.
func (c *Client) Delete(addr string) *Request {
	return c.Req(http.MethodDelete, addr)
}

// Get makes a get http request.
func (c *Client) Get(addr string) *Request {
	return c.Req(http.MethodGet, addr)
}

// HEAD makes a head http request.
func (c *Client) Head(addr string) *Request {
	return c.Req(http.MethodHead, addr)
}

// Options makes a options http request.
func (c *Client) Options(addr string) *Request {
	return c.Req(http.MethodOptions, addr)
}

// Patch makes a patch http request.
func (c *Client) Patch(addr string) *Request {
	return c.Req(http.MethodPatch, addr)
}

// Post makes a post http request.
func (c *Client) Post(addr string) *Request {
	return c.Req(http.MethodPost, addr)
}

// Put makes a put http request.
func (c *Client) Put(addr string) *Request {
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
		headers:  c.headers,
		doer:     c.doer,
		authFn:   c.authFn,
		encodeFn: c.encodeFn,
		backoff:  c.backoff,
	}
}
