package httpc

// ClientOptFn sets keys on a client type.
type ClientOptFn func(Client) Client

// WithAuth sets the authorization func on the client type,
// and will be used as the default authFn for all requests
// from this client unless overwritten atn the request lvl.
func WithAuth(authFn AuthFn) ClientOptFn {
	return func(c Client) Client {
		c.authFn = authFn
		return c
	}
}

// WithBackoff sets teh backoff on the client.
func WithBackoff(b BackoffOptFn) ClientOptFn {
	return func(c Client) Client {
		c.backoff = b
		return c
	}
}

// WithBaseURL sets teh base url for all requests. Any path provided will be
// appended to this WithBaseURL.
func WithBaseURL(baseURL string) ClientOptFn {
	return func(c Client) Client {
		c.baseURL = baseURL
		return c
	}
}

// WithEncode sets the encode func for the client.
func WithEncode(fn EncodeFn) ClientOptFn {
	return func(c Client) Client {
		c.encodeFn = fn
		return c
	}
}