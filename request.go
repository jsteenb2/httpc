package httpc

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

// ErrInvalidEncodeFn is an error that is returned when calling the Request Do and the
// encode function is not set.
var ErrInvalidEncodeFn = errors.New("no encode fn provided for body")

// ResponseErrorFn is a response error function that can be used to provide
// behavior when a response fails to "Do".
type ResponseErrorFn func(error) error

type kvPair struct {
	key   string
	value string
}

// Request is built up to create an http request.
type Request struct {
	Method, Addr string
	doer         Doer
	body         interface{}

	headers []kvPair
	params  []kvPair

	authFn        AuthFn
	encodeFn      EncodeFn
	decodeFn      DecodeFn
	onErrorFn     DecodeFn
	responseErrFn ResponseErrorFn

	notFoundFns    []StatusFn
	existsFns      []StatusFn
	retryStatusFns []StatusFn
	successFns     []StatusFn

	backoff BackoffOptFn
}

// Auth sets the authorization for hte request, overriding the authFn set
// by the client.
func (r *Request) Auth(authFn AuthFn) *Request {
	r.authFn = authFn
	return r
}

// Backoff sets the backoff of the Request.
func (r *Request) Backoff(b BackoffOptFn) *Request {
	r.backoff = b
	return r
}

// Body sets the body of the Request.
func (r *Request) Body(v interface{}) *Request {
	r.body = v
	return r
}

// ContentType sets the content type for the outgoing request.
func (r *Request) ContentType(cType string) *Request {
	r.headers = append(r.headers, kvPair{key: "Content-Type", value: cType})
	return r
}

// Decode sets the decoder func for the Request.
func (r *Request) Decode(fn DecodeFn) *Request {
	r.decodeFn = fn
	return r
}

// Exists appends a exists func to the Request.
func (r *Request) Exists(fn StatusFn) *Request {
	r.existsFns = append(r.existsFns, fn)
	return r
}

// Header adds a header to the request.
func (r *Request) Header(key, value string) *Request {
	r.headers = append(r.headers, kvPair{key: key, value: value})
	return r
}

// OnError provides a decode hook to decode a responses body.
func (r *Request) OnError(fn DecodeFn) *Request {
	r.onErrorFn = fn
	return r
}

// NotFound appends a not found func to the Request.
func (r *Request) NotFound(fn StatusFn) *Request {
	r.notFoundFns = append(r.notFoundFns, fn)
	return r
}

// QueryParam allows a user to set query params on their request. This can be
// called numerous times. Will add keys for each value that is passed in here.
// In the case of duplicate query param values, the last pair that is entered
// will be set and the former will not be available.
func (r *Request) QueryParam(key, value string) *Request {
	r.params = append(r.params, kvPair{key: key, value: value})
	return r
}

// QueryParams allows a user to set multiple query params at one time. Following
// the same rules as QueryParam. If a key is provided without a value, it will
// not be added to the request. If it is desired, pass in "" to add a query param
// with no string field.
func (r *Request) QueryParams(key, value string, pairs ...string) *Request {
	paramed := r.QueryParam(key, value)
	for index := 0; index < len(pairs)/2; index++ {
		i := index * 2
		pair := pairs[i : i+2]
		if len(pair) != 2 {
			return paramed
		}
		paramed = r.QueryParam(pair[0], pair[1])
	}
	return paramed
}

// Retry sets the retry policy(s) on the request.
func (r *Request) Retry(fn RetryFn) *Request {
	return fn(r)
}

// Success appends a success func to the Request.
func (r *Request) Success(fn StatusFn) *Request {
	r.successFns = append(r.successFns, fn)
	return r
}

// Do makes the http request and applies the backoff.
func (r *Request) Do(ctx context.Context) error {
	return retry(ctx, r.do, r.backoff)
}

func (r *Request) do(ctx context.Context) error {
	var body io.Reader
	if r.body != nil {
		if r.encodeFn == nil {
			return ErrInvalidEncodeFn
		}

		encodedBody, err := r.encodeFn(r.body)
		if err != nil {
			return NewClientErr(Err(err))
		}
		body = encodedBody
	}

	req, err := http.NewRequest(r.Method, r.Addr, body)
	if err != nil {
		return NewClientErr(Err(err))
	}
	req = req.WithContext(ctx)

	if len(r.headers) > 0 {
		for _, pair := range r.headers {
			req.Header.Set(pair.key, pair.value)
		}
	}

	if len(r.params) > 0 {
		params := req.URL.Query()
		for _, kv := range r.params {
			params.Set(kv.key, kv.value)
		}
		req.URL.RawQuery = params.Encode()
	}

	if r.authFn != nil {
		req = r.authFn(req)
	}

	resp, err := r.doer.Do(req)
	if err != nil {
		return r.responseErr(resp, err)
		return NewClientErr(Err(err), Resp(resp))
	}
	defer func() {
		drain(resp.Body)
	}()

	status := resp.StatusCode
	if !statusMatches(status, r.successFns) {
		opts := append([]ErrOptFn{Resp(resp)}, r.statusErrOpts(status)...)
		if r.onErrorFn != nil {
			var buf bytes.Buffer
			tee := io.TeeReader(resp.Body, &buf)
			if err := r.onErrorFn(tee); err != nil {
				opts = append(opts, Err(err))
			}
			resp.Body = ioutil.NopCloser(&buf)
		}
		return NewClientErr(opts...)
	}

	if r.decodeFn == nil {
		return nil
	}

	if err := r.decodeFn(resp.Body); err != nil {
		opts := []ErrOptFn{Err(err), Resp(resp)}
		if isRetryErr(err) {
			opts = append(opts, Retry())
		}
		return NewClientErr(opts...)
	}

	return nil
}

func (r *Request) statusErrOpts(status int) []ErrOptFn {
	var opts []ErrOptFn
	if statusMatches(status, r.retryStatusFns) {
		opts = append(opts, Retry())
	}
	if statusMatches(status, r.notFoundFns) {
		opts = append(opts, NotFound())
	}
	if statusMatches(status, r.existsFns) {
		opts = append(opts, Exists())
	}
	return opts
}

func (r *Request) responseErr(resp *http.Response, err error) error {
	if r.responseErrFn != nil {
		err = r.responseErrFn(err)
	}
	opts := []ErrOptFn{Err(err), Resp(resp)}
	if isRetryErr(err) {
		opts = append(opts, Retry())
	}
	return NewClientErr(opts...)
}

// drain reads everything from the ReadCloser and closes it
func drain(r io.ReadCloser) error {
	var msgs []string
	if _, err := io.Copy(ioutil.Discard, r); err != nil {
		msgs = append(msgs, err.Error())
	}
	if err := r.Close(); err != nil {
		msgs = append(msgs, err.Error())
	}
	return errors.New(strings.Join(msgs, "; "))
}

func isRetryErr(err error) bool {
	if err == nil {
		return false
	}
	r, ok := err.(retrier)
	return ok && r.Retry()
}
