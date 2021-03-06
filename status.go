package httpc

import "net/http"

// StatusFn is func for comparing expected status code against
// an expected status code.
type StatusFn func(statusCode int) bool

// StatusIn checks whether the response's status code matches at least 1
// of the input status codes provided.
func StatusIn(status ...int) StatusFn {
	return func(statusCode int) bool {
		for _, code := range status {
			if code == statusCode {
				return true
			}
		}
		return false
	}
}

// StatusInRange checks the response's status code is in in the range
// provided.
func StatusInRange(low, high int) StatusFn {
	return func(statusCode int) bool {
		return low <= statusCode && high >= statusCode
	}
}

// StatusNotIn checks whether the response's status code does match any
// of the input status codes provided.
func StatusNotIn(status ...int) StatusFn {
	return func(statusCode int) bool {
		return !StatusIn(status...)(statusCode)
	}
}

// StatusNot compares the response's status code to verify it does match
// the status code input.
func StatusNot(status int) StatusFn {
	return func(statusCode int) bool {
		return status != statusCode
	}
}

// StatusOK compares the response's status code to match Status OK.
func StatusOK() StatusFn {
	return func(status int) bool {
		return http.StatusOK == status
	}
}

// StatusCreated compares the response's status code to match Status Created.
func StatusCreated() StatusFn {
	return func(status int) bool {
		return http.StatusCreated == status
	}
}

// StatusNoContent compares the response's status code to match Status No Content.
func StatusNoContent() StatusFn {
	return func(status int) bool {
		return http.StatusNoContent == status
	}
}

// StatusNotFound compares the response's status code to match Status Not Found.
func StatusNotFound() StatusFn {
	return func(status int) bool {
		return http.StatusNotFound == status
	}
}

// StatusUnprocessableEntity compares the response's status code to match Status Unprocessable Entity.
func StatusUnprocessableEntity() StatusFn {
	return func(status int) bool {
		return http.StatusUnprocessableEntity == status
	}
}

// StatusInternalServerError compares the response's status code to match Status Internal Server Error.
func StatusInternalServerError() StatusFn {
	return func(status int) bool {
		return http.StatusInternalServerError == status
	}
}

func statusMatches(status int, fns []StatusFn) bool {
	for _, fn := range fns {
		if fn(status) {
			return true
		}
	}
	return false
}
