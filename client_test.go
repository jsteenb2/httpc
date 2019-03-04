package httpc_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jsteenb2/httpc"
)

func TestClient_Req(t *testing.T) {
	t.Run("no body", func(t *testing.T) {
		t.Run("basics", func(t *testing.T) {
			tests := []int{http.StatusOK, http.StatusAccepted}
			methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}

			for _, method := range methods {
				for _, status := range tests {
					fn := func(t *testing.T) {
						doer := new(fakeDoer)
						doer.doFn = func(req *http.Request) (*http.Response, error) {
							return stubResp(status), nil
						}

						client := httpc.New(doer)

						err := client.
							Req(method, "/foo").
							Success(func(statusCode int) bool {
								return status == statusCode
							}).
							Do(context.TODO())
						mustNoError(t, err)
					}

					t.Run(method+"/"+http.StatusText(status), fn)
				}
			}
		})

		t.Run("DELETE", func(t *testing.T) {
			doer := new(fakeDoer)
			doer.doFn = func(req *http.Request) (*http.Response, error) {
				return stubResp(http.StatusNoContent), nil
			}

			client := httpc.New(doer)

			err := client.
				DELETE("/foo").
				Success(httpc.StatusNoContent()).
				Do(context.TODO())
			mustNoError(t, err)
		})
	})

	t.Run("with response body", func(t *testing.T) {
		t.Run("GET", func(t *testing.T) {
			doer := new(fakeDoer)
			expected := foo{Name: "Name"}
			doer.doFn = func(r *http.Request) (*http.Response, error) {
				f := expected
				f.Method = r.Method
				return stubRespNBody(t, http.StatusOK, f), nil
			}

			client := httpc.New(doer)

			var fooResp foo
			err := client.
				GET("/foo").
				Success(httpc.StatusOK()).
				Decode(httpc.JSONDecode(&fooResp)).
				Do(context.TODO())
			mustNoError(t, err)

			expected.Method = "GET"
			equals(t, expected, fooResp)
		})

		t.Run("with request body", func(t *testing.T) {
			t.Run("POST", func(t *testing.T) {
				doer := newEchoDoer(t, http.StatusOK)

				client := httpc.New(doer)

				expected := foo{Name: "name", S: "string"}
				var fooResp foo
				err := client.
					POST("/foo").
					Body(expected).
					Success(httpc.StatusOK()).
					Decode(httpc.JSONDecode(&fooResp)).
					Do(context.TODO())
				mustNoError(t, err)

				expected.Method = "POST"
				equals(t, expected, fooResp)
			})

			t.Run("PATCH", func(t *testing.T) {
				doer := newEchoDoer(t, http.StatusOK)

				client := httpc.New(doer)

				expected := foo{Name: "name", S: "string"}
				var fooResp foo
				err := client.
					PATCH("/foo").
					Body(expected).
					Success(httpc.StatusOK()).
					Decode(httpc.JSONDecode(&fooResp)).
					Do(context.TODO())
				mustNoError(t, err)

				expected.Method = "PATCH"
				equals(t, expected, fooResp)
			})

			t.Run("PUT", func(t *testing.T) {
				doer := newEchoDoer(t, http.StatusOK)

				client := httpc.New(doer)

				expected := foo{Name: "name", S: "string"}
				var fooResp foo
				err := client.
					PUT("/foo").
					Body(expected).
					Success(httpc.StatusOK()).
					Decode(httpc.JSONDecode(&fooResp)).
					Do(context.TODO())
				mustNoError(t, err)

				expected.Method = "PUT"
				equals(t, expected, fooResp)
			})
		})
	})

	t.Run("with query params", func(t *testing.T) {
		t.Run("without duplicates", func(t *testing.T) {
			doer := new(fakeDoer)
			doer.doFn = func(req *http.Request) (*http.Response, error) {
				return stubResp(http.StatusOK), nil
			}

			client := httpc.New(doer)

			req := client.GET("/foo").Success(httpc.StatusOK())

			for i := 'A'; i <= 'Z'; i++ {
				req = req.QueryParam(string(i), string(i+26))
			}

			err := req.Do(context.TODO())
			mustNoError(t, err)

			mustEquals(t, 1, len(doer.args))
			httpReq := doer.args[0]
			params := httpReq.URL.Query()

			for i := 'A'; i <= 'Z'; i++ {
				equals(t, string(i+26), params.Get(string(i)))
			}
		})

		t.Run("with dpulicates", func(t *testing.T) {
			t.Run("duplicate entries last entry wins", func(t *testing.T) {
				doer := new(fakeDoer)
				doer.doFn = func(req *http.Request) (*http.Response, error) {
					return stubResp(http.StatusOK), nil
				}

				client := httpc.New(doer)

				err := client.
					GET("/foo").
					QueryParam("dupe", "val1").
					QueryParam("dupe", "val2").
					Success(httpc.StatusOK()).
					Do(context.TODO())
				mustNoError(t, err)

				mustEquals(t, 1, len(doer.args))
				httpReq := doer.args[0]
				params := httpReq.URL.Query()

				equals(t, "val2", params.Get("dupe"))
			})
		})

		t.Run("ignores unfulfilled pairs", func(t *testing.T) {
			doer := new(fakeDoer)
			doer.doFn = func(req *http.Request) (*http.Response, error) {
				return stubResp(http.StatusOK), nil
			}

			client := httpc.New(doer)

			err := client.
				GET("/foo").
				QueryParams("q1", "v1", "q2").
				Success(httpc.StatusOK()).
				Do(context.TODO())
			mustNoError(t, err)

			mustEquals(t, 1, len(doer.args))
			httpReq := doer.args[0]
			params := httpReq.URL.Query()
			equals(t, "", params.Get("q2"))
		})
	})

	t.Run("gob encoding", func(t *testing.T) {
		doer := new(fakeDoer)
		doer.doFn = func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       r.Body,
			}, nil
		}

		client := httpc.New(doer, httpc.WithEncoder(httpc.GobEncode()))

		expected := foo{Name: "name", S: "string"}
		var fooResp foo
		err := client.
			GET("/foo").
			Body(expected).
			Success(httpc.StatusOK()).
			Decode(httpc.GobDecode(&fooResp)).
			Do(context.TODO())
		mustNoError(t, err)

		equals(t, expected, fooResp)
	})

	t.Run("handling errors response body", func(t *testing.T) {
		type bar struct{ Name string }

		doer := new(fakeDoer)
		expected := bar{Name: "error"}
		doer.doFn = func(req *http.Request) (*http.Response, error) {
			return stubRespNBody(t, http.StatusNotFound, expected), nil
		}

		client := httpc.New(doer)

		var actual bar
		err := client.
			DELETE("/foo").
			Success(httpc.StatusNoContent()).
			OnError(httpc.JSONDecode(&actual)).
			Do(context.TODO())
		mustError(t, err)

		expectedErrResp := `response_body="{\"Name\":\"error\"}\n"`
		// test that verifies the resp body is still readable after err response fn does its read
		found := strings.Contains(err.Error(), `response_body="{\"Name\":\"error\"}\n"`)
		if !found {
			t.Errorf("error body not found: expected=%q got=%q", expectedErrResp, err.Error())
		}

		equals(t, expected, actual)
	})

	t.Run("retry", func(t *testing.T) {
		t.Run("sets retry", func(t *testing.T) {
			doer := new(fakeDoer)
			doer.doFn = func(*http.Request) (*http.Response, error) {
				return stubResp(http.StatusInternalServerError), nil
			}

			client := httpc.New(doer)

			err := client.
				DELETE("/foo").
				Success(httpc.StatusNoContent()).
				Retry(httpc.RetryStatus(httpc.StatusNotIn(http.StatusOK))).
				Retry(httpc.RetryStatus(httpc.StatusNotIn(http.StatusNoContent, http.StatusNotFound))).
				Do(context.TODO())
			mustError(t, err)

			isRetryErr(t, err)
		})

		t.Run("does not set retry", func(t *testing.T) {
			doer := new(fakeDoer)
			doer.doFn = func(request *http.Request) (*http.Response, error) {
				return stubResp(http.StatusUnprocessableEntity), nil
			}

			client := httpc.New(doer)

			err := client.
				DELETE("/foo").
				Success(httpc.StatusNoContent()).
				Retry(httpc.RetryStatus(httpc.StatusNotIn(http.StatusUnprocessableEntity))).
				Do(context.TODO())
			mustError(t, err)

			equals(t, false, retryErr(err))
		})

		t.Run("applies backoff on retry", func(t *testing.T) {
			doer := new(fakeDoer)
			doer.doFn = func(*http.Request) (*http.Response, error) {
				return stubResp(http.StatusNotFound), nil
			}

			boffer := httpc.NewConstantBackoff(time.Nanosecond, 3)
			client := httpc.New(doer, httpc.WithBackoff(boffer))

			err := client.
				DELETE("/foo").
				Success(httpc.StatusNoContent()).
				Retry(httpc.RetryStatus(httpc.StatusNotFound())).
				Do(context.TODO())
			mustError(t, err)

			isRetryErr(t, err)
			equals(t, 3, doer.doCallCount)
		})

		t.Run("does not backoff on non retry error", func(t *testing.T) {
			doer := new(fakeDoer)
			doer.doFn = func(*http.Request) (*http.Response, error) {
				return stubResp(http.StatusInternalServerError), nil
			}

			boffer := httpc.NewConstantBackoff(time.Nanosecond, 10)
			client := httpc.New(doer, httpc.WithBackoff(boffer))

			err := client.
				DELETE("/foo").
				Success(httpc.StatusNoContent()).
				Retry(httpc.RetryStatus(httpc.StatusNotFound())).
				Do(context.TODO())
			mustError(t, err)

			equals(t, false, retryErr(err))
			equals(t, 1, doer.doCallCount)
		})

		t.Run("retries on retryable response error", func(t *testing.T) {
			boffer := httpc.NewConstantBackoff(time.Nanosecond, 3)

			doer := new(fakeDoer)
			doer.doFn = func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("unexpected error")
			}

			client := httpc.New(doer, httpc.WithBackoff(boffer))

			err := client.
				DELETE("/foo").
				Success(httpc.StatusNoContent()).
				Retry(httpc.RetryResponseError(func(e error) error {
					return &fakeRetryErr{e}
				})).
				Do(context.TODO())
			mustError(t, err)

			isRetryErr(t, err)
			equals(t, 3, doer.doCallCount)
		})
	})

	t.Run("response error handled", func(t *testing.T) {
		doer := new(fakeDoer)
		doer.doFn = func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("unexpected error")
		}

		client := httpc.New(doer)

		var count int
		err := client.
			DELETE("/foo").
			Success(httpc.StatusOK()).
			Retry(httpc.RetryResponseError(func(e error) error {
				count++
				return e
			})).
			Do(context.TODO())
		mustError(t, err)

		equals(t, 1, count)
	})

	t.Run("headers", func(t *testing.T) {
		t.Run("client headers set on req", func(t *testing.T) {
			doer := new(fakeDoer)
			doer.doFn = func(req *http.Request) (*http.Response, error) {
				return stubResp(http.StatusOK), nil
			}

			var opts []httpc.ClientOptFn
			for i := 'A'; i <= 'Z'; i++ {
				opts = append(opts, httpc.WithHeader(string(i), string(i+26)))
			}
			client := httpc.New(doer, opts...)

			err := client.
				GET("/foo").
				Success(httpc.StatusOK()).
				Do(context.TODO())
			mustNoError(t, err)

			mustEquals(t, 1, len(doer.args))
			httpReq := doer.args[0]
			headers := httpReq.Header

			for i := 'A'; i <= 'Z'; i++ {
				equals(t, string(i+26), headers.Get(string(i)))
			}
		})

		t.Run("request header overwrites a client header", func(t *testing.T) {
			doer := new(fakeDoer)
			doer.doFn = func(req *http.Request) (*http.Response, error) {
				return stubResp(http.StatusOK), nil
			}

			client := httpc.New(doer, httpc.WithHeader("key", "value"))

			err := client.
				GET("/foo").
				Header("key", "new value").
				Success(httpc.StatusOK()).
				Do(context.TODO())
			mustNoError(t, err)

			mustEquals(t, 1, len(doer.args))
			httpReq := doer.args[0]
			headers := httpReq.Header

			equals(t, "new value", headers.Get("key"))
		})

		t.Run("non duplicates", func(t *testing.T) {
			doer := new(fakeDoer)
			doer.doFn = func(req *http.Request) (*http.Response, error) {
				return stubResp(http.StatusOK), nil
			}

			client := httpc.New(doer)

			req := client.
				GET("/foo")

			for i := 'A'; i <= 'Z'; i++ {
				req = req.Header(string(i), string(i+26))
			}

			err := req.Success(httpc.StatusOK()).
				Do(context.TODO())
			mustNoError(t, err)

			mustEquals(t, 1, len(doer.args))
			httpReq := doer.args[0]
			headers := httpReq.Header

			for i := 'A'; i <= 'Z'; i++ {
				equals(t, string(i+26), headers.Get(string(i)))
			}
		})

		t.Run("duplicate entries last entry wins", func(t *testing.T) {
			doer := new(fakeDoer)
			doer.doFn = func(req *http.Request) (*http.Response, error) {
				return stubResp(http.StatusOK), nil
			}

			client := httpc.New(doer)

			err := client.
				GET("/foo").
				Header("dupe", "val1").
				Header("dupe", "val2").
				Success(httpc.StatusOK()).
				Do(context.TODO())
			mustNoError(t, err)

			mustEquals(t, 1, len(doer.args))
			httpReq := doer.args[0]
			headers := httpReq.Header

			equals(t, "val2", headers.Get("dupe"))
		})
	})

	t.Run("sets content type", func(t *testing.T) {
		t.Run("on client", func(t *testing.T) {
			doer := new(fakeDoer)
			doer.doFn = func(req *http.Request) (*http.Response, error) {
				return stubResp(http.StatusOK), nil
			}

			client := httpc.New(doer, httpc.WithContentType("application/json"))

			err := client.
				GET("/foo").
				Success(httpc.StatusOK()).
				Do(context.TODO())
			mustNoError(t, err)

			mustEquals(t, 1, len(doer.args))
			httpReq := doer.args[0]
			headers := httpReq.Header

			equals(t, "application/json", headers.Get("Content-Type"))
		})

		t.Run("on request", func(t *testing.T) {
			doer := new(fakeDoer)
			doer.doFn = func(req *http.Request) (*http.Response, error) {
				return stubResp(http.StatusOK), nil
			}

			client := httpc.New(doer)

			err := client.
				GET("/foo").
				ContentType("application/json").
				Success(httpc.StatusOK()).
				Do(context.TODO())
			mustNoError(t, err)

			mustEquals(t, 1, len(doer.args))
			httpReq := doer.args[0]
			headers := httpReq.Header

			equals(t, "application/json", headers.Get("Content-Type"))
		})

		t.Run("request overwrite client content type", func(t *testing.T) {
			doer := new(fakeDoer)
			doer.doFn = func(req *http.Request) (*http.Response, error) {
				return stubResp(http.StatusOK), nil
			}

			client := httpc.New(doer, httpc.WithContentType("text/html"))

			err := client.
				GET("/foo").
				ContentType("application/json").
				Success(httpc.StatusOK()).
				Do(context.TODO())
			mustNoError(t, err)

			mustEquals(t, 1, len(doer.args))
			httpReq := doer.args[0]
			headers := httpReq.Header

			equals(t, "application/json", headers.Get("Content-Type"))
		})
	})

	t.Run("not found", func(t *testing.T) {
		t.Run("sets not found", func(t *testing.T) {
			doer := new(fakeDoer)
			doer.doFn = func(*http.Request) (*http.Response, error) {
				return stubResp(http.StatusNotFound), nil
			}

			client := httpc.New(doer)

			err := client.
				DELETE("/foo").
				Success(httpc.StatusNoContent()).
				NotFound(httpc.StatusNotFound()).
				Do(context.TODO())
			mustError(t, err)

			equals(t, true, notFoundErr(err))
		})

		t.Run("does not set not found", func(t *testing.T) {
			doer := new(fakeDoer)
			doer.doFn = func(*http.Request) (*http.Response, error) {
				return stubResp(http.StatusUnprocessableEntity), nil
			}

			client := httpc.New(doer)

			err := client.
				DELETE("/foo").
				Success(httpc.StatusNoContent()).
				NotFound(httpc.StatusNotFound()).
				Do(context.TODO())
			mustError(t, err)

			equals(t, false, notFoundErr(err))
		})
	})

	t.Run("exists", func(t *testing.T) {
		t.Run("sets exist", func(t *testing.T) {
			doer := new(fakeDoer)
			doer.doFn = func(*http.Request) (*http.Response, error) {
				return stubResp(http.StatusUnprocessableEntity), nil
			}

			client := httpc.New(doer)

			err := client.
				DELETE("/foo").
				Success(httpc.StatusNoContent()).
				Exists(httpc.StatusUnprocessableEntity()).
				Do(context.TODO())
			mustError(t, err)

			equals(t, true, existsErr(err))
		})

		t.Run("does not set exist", func(t *testing.T) {
			doer := new(fakeDoer)
			doer.doFn = func(*http.Request) (*http.Response, error) {
				return stubResp(http.StatusNotFound), nil
			}

			client := httpc.New(doer)

			err := client.
				DELETE("/foo").
				Success(httpc.StatusNoContent()).
				Exists(httpc.StatusUnprocessableEntity()).
				Do(context.TODO())
			mustError(t, err)

			equals(t, false, existsErr(err))
		})
	})

	t.Run("auth", func(t *testing.T) {
		t.Run("basic auth", func(t *testing.T) {
			doer := new(fakeDoer)
			doer.doFn = func(r *http.Request) (*http.Response, error) {
				u, p, ok := r.BasicAuth()
				if !ok {
					return stubResp(http.StatusInternalServerError), nil
				}
				f := foo{Name: u, S: p, Method: r.Method}
				return stubRespNBody(t, http.StatusOK, f), nil
			}

			client := httpc.New(doer, httpc.WithAuth(httpc.BasicAuth("user", "pass")))

			var actual foo
			err := client.
				GET("/foo").
				Success(httpc.StatusOK()).
				Decode(httpc.JSONDecode(&actual)).
				Do(context.TODO())
			mustNoError(t, err)

			equals(t, "user", actual.Name)
			equals(t, "pass", actual.S)
		})

		t.Run("token auth", func(t *testing.T) {
			doer := new(fakeDoer)
			doer.doFn = func(r *http.Request) (*http.Response, error) {
				f := foo{Name: r.Header.Get("Authorization"), Method: r.Method}
				return stubRespNBody(t, http.StatusOK, f), nil
			}

			client := httpc.New(doer, httpc.WithAuth(httpc.BearerTokenAuth("token")))

			var actual foo
			err := client.
				GET("/foo").
				Success(httpc.StatusOK()).
				Decode(httpc.JSONDecode(&actual)).
				Do(context.TODO())
			mustNoError(t, err)

			equals(t, "Bearer token", actual.Name)
		})
	})
}

type foo struct {
	Name   string
	S      string
	Method string
}

func stubRespNBody(t *testing.T, status int, v interface{}) *http.Response {
	t.Helper()

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		t.Fatal(err)
	}
	return &http.Response{
		StatusCode: status,
		Body:       ioutil.NopCloser(&buf),
	}
}

func stubResp(status int) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       ioutil.NopCloser(new(bytes.Buffer)),
	}
}

func isRetryErr(t *testing.T, err error) {
	t.Helper()
	if !retryErr(err) {
		t.Fatal("got a non retriable error: ", err)
	}
}

func retryErr(err error) bool {
	type retrier interface {
		Retry() bool
	}
	r, ok := err.(retrier)
	return ok && r.Retry()
}

func notFoundErr(err error) bool {
	type notFounder interface {
		NotFound() bool
	}
	nf, ok := err.(notFounder)
	return ok && nf.NotFound()
}

func existsErr(err error) bool {
	type exister interface {
		Exists() bool
	}
	ex, ok := err.(exister)
	return ok && ex.Exists()
}

type fakeDoer struct {
	doCallCount int
	doFn        func(*http.Request) (*http.Response, error)
	args        []*http.Request
}

func (f *fakeDoer) Do(r *http.Request) (*http.Response, error) {
	f.args = append(f.args, r)
	f.doCallCount++
	return f.doFn(r)
}

func newEchoDoer(t *testing.T, status int) *fakeDoer {
	doer := new(fakeDoer)
	doer.doFn = func(req *http.Request) (*http.Response, error) {
		var f foo
		if err := json.NewDecoder(req.Body).Decode(&f); err != nil {
			t.Fatal("error in decoder: ", err)
		}

		f.Method = req.Method
		return stubRespNBody(t, status, f), nil
	}
	return doer
}

func mustError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		return
	}
	t.Fatal("expected error but was none")
}

func mustNoError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	t.Fatalf("expected no error: got: %q", err)
}

func equals(t *testing.T, expected interface{}, actual interface{}) {
	t.Helper()
	if expected == actual {
		return
	}
	t.Errorf("expected: %v\tgot: %v", expected, actual)
}

func mustEquals(t *testing.T, expected interface{}, actual interface{}) {
	t.Helper()
	if expected == actual {
		return
	}
	t.Fatalf("expected: %v\tgot: %v", expected, actual)
}

type fakeRetryErr struct {
	error
}

func (f *fakeRetryErr) Retry() bool {
	return true
}
