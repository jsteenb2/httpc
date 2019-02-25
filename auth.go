package httpc

import "net/http"

// AuthFn adds authorization to an http request.
type AuthFn func(*http.Request) *http.Request

// BasicAuth sets the basic authFn on the request.
func BasicAuth(user, pass string) AuthFn {
	return func(r *http.Request) *http.Request {
		r.SetBasicAuth(user, pass)
		return r
	}
}

// BearerTokenAuth sets the token authentication on the request.
func BearerTokenAuth(token string) AuthFn {
	return func(r *http.Request) *http.Request {
		r.Header.Add("Authorization", "Bearer "+token)
		return r
	}
}
