package httpc

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type retrier interface {
	Retry() bool
}

// HTTPErr is an error type that provides useful error messages that include
// both request and response bodies, status code of response and valid request
// parameters.
type HTTPErr struct {
	caller     string
	u          url.URL
	method     string
	errMsg     string
	respBody   string
	reqBody    string
	statusCode int
	retry      bool
	notFound   bool
	exists     bool
}

// NewClientErr is a constructor for a client error. The provided options
// allow the caller to set an optional retry.
func NewClientErr(opts ...ErrOptFn) error {
	var opt errOpt
	for _, o := range opts {
		opt = o(opt)
	}

	newClientErr := &HTTPErr{
		notFound: opt.notFound,
		exists:   opt.exists,
		retry:    opt.retry,
		caller:   opt.caller,
		errMsg:   "received unexpected response",
	}
	if opt.err != nil {
		newClientErr.errMsg = opt.err.Error()
	}

	if opt.resp == nil {
		return newClientErr
	}

	if req := opt.resp.Request; req != nil {
		newClientErr.u = *req.URL
		newClientErr.method = req.Method

		if req.Header != nil && strings.Contains(req.Header.Get("Content-Type"), "application/json") {
			if body, err := ioutil.ReadAll(req.Body); err == nil {
				newClientErr.respBody = string(body)
			}
		}
	}
	newClientErr.statusCode = opt.resp.StatusCode

	if body, err := ioutil.ReadAll(opt.resp.Body); err == nil {
		newClientErr.respBody = string(body)
	}
	opt.resp = nil
	opt.err = nil
	return newClientErr
}

// Error returns the full client error message.
func (e *HTTPErr) Error() string {
	parts := []string{e.errorBase()}

	if msg := e.errMsg; msg != "" {
		parts = append(parts, fmt.Sprintf("err=%q", msg))
	}

	if respBody := e.respBody; respBody != "" {
		parts = append(parts, fmt.Sprintf("response_body=%q", respBody))
	}

	if reqBody := e.reqBody; reqBody != "" {
		parts = append(parts, fmt.Sprintf("request_body=%q", reqBody))
	}

	return strings.Join(parts, " ")
}

// BackoffMessage provides a condensed error message that can be consumed during
// a backoff loop.
func (e *HTTPErr) BackoffMessage() string {
	return e.errorBase()
}

// Retry provides the retry behavior.
func (e *HTTPErr) Retry() bool {
	return e.retry
}

// NotFound provides the NotFounder behavior.
func (e *HTTPErr) NotFound() bool {
	return e.notFound
}

// Exists provides the Exister behavior.
func (e *HTTPErr) Exists() bool {
	return e.exists
}

func (e *HTTPErr) errorBase() string {
	var parts []string

	if e.statusCode != 0 {
		parts = append(parts, fmt.Sprintf("status=%d", e.statusCode))
	}

	if e.method != "" {
		parts = append(parts, fmt.Sprintf("method=%s", e.method))
	}

	if e.u.String() != "" {
		q := e.u.Query()
		if q.Get("access_token") != "" {
			q.Set("access_token", "REDACTED")
		}
		if q.Get("secret") != "" {
			q.Set("secret", "REDACTED")
		}
		e.u.RawQuery = q.Encode()
		parts = append(parts, fmt.Sprintf("url=%q", e.u.String()))
	}

	return strings.Join(parts, " ")
}

type errOpt struct {
	retry, notFound, exists bool

	err    error
	caller string
	resp   *http.Response
}

// ErrOptFn is a optional parameter that allows one to extend a client error.
type ErrOptFn func(o errOpt) errOpt

func Op(c string) ErrOptFn {
	return func(o errOpt) errOpt {
		o.caller = c
		return o
	}
}

func Err(err error) ErrOptFn {
	return func(o errOpt) errOpt {
		o.err = err
		return o
	}
}

func Resp(resp *http.Response) ErrOptFn {
	return func(o errOpt) errOpt {
		o.resp = resp
		return o
	}
}

// Retry sets the option and subsequent client error to retriable, retry=true.
func Retry() ErrOptFn {
	return func(o errOpt) errOpt {
		o.retry = true
		return o
	}
}

// NotFound sets the client error to NotFound, notFound=true.
func NotFound() ErrOptFn {
	return func(o errOpt) errOpt {
		o.notFound = true
		return o
	}
}

// Exists sets the client error to Exists, exists=true.
func Exists() ErrOptFn {
	return func(o errOpt) errOpt {
		o.exists = true
		return o
	}
}
