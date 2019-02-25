package httpc

// RetryFn will apply a retry policy to a request.
type RetryFn func(*Request) *Request

// RetryStatus appends a retry func to the Request.
func RetryStatus(fn StatusFn) RetryFn {
	return func(req *Request) *Request {
		req.retryStatusFns = append(req.retryStatusFns, fn)
		return req
	}
}

// RetryResponseError applies a retry on all response errors. The errors
// typically associated with request timeouts or oauth token error.
// This option useful when the oauth auth made me invalid or a request timeout
// is an issue.
func RetryResponseError(fn ResponseErrorFn) RetryFn {
	return func(r *Request) *Request {
		r.responseErrFn = fn
		return r
	}
}
