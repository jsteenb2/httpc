# httpc

`httpc` is a simple http client that allows the consumer to build simple http calls and apply intelligent behavior driven backoffs.

## GET example with JSON decoder

```go
// server echos body and adds HTTP method name

type foo struct {
	Name   string
	Thing  string
	Method string
}

client := httpc.New(doer)

var fooResp foo
err := client.
    GET("/foo").
    Success(httpc.StatusOK()).
    Decode(httpc.JSONDecode(&fooResp)).
    Do(context.TODO())
mustNoError(t, err)

fmt.Printf("name=%q thing=%q method=%q", foo.Name, foo.Thing)
// Output:
// name="fooName" thing="fooThing" method="GET"
```

## POST example with JSON encoder/decoder

```go
// server echos body and adds HTTP method name

type foo struct {
	Name   string
	Thing  string
	Method string
}

client := httpc.New(doer)

expected := foo{Name: "name", Thing: "thing"}
var fooResp foo
err := client.
    POST(svr.URL + "/foo").
    Body(expected).
    Success(httpc.StatusOK()).
    Decode(httpc.JSONDecode(&fooResp)).
    Do(context.TODO())
mustNoError(t, err)

fmt.Printf("name=%q thing=%q method=%q", foo.Name, foo.Thing, foo.Method)
// Output:
// name="name" thing="thing" method="POST"
```

## PUT example with GOB encoder/decoder

```go
// server echos body and adds HTTP method name

type foo struct {
	Name   string
	Thing  string
	Method string
}

client := httpc.New(doer, httpc.Encode(httpc.GobEncode()))

expected := foo{Name: "name", Thing: "thing"}
var fooResp foo
err := client.
    PUT(svr.URL + "/foo").
    Body(expected).
    Success(httpc.StatusOK()).
    Decode(httpc.GobDecode(&fooResp)).
    Do(context.TODO())
mustNoError(t, err)

fmt.Printf("name=%q thing=%q method=%q", foo.Name, foo.Thing, foo.Method)
// Output:
// name="name" thing="thing" method="PUT"
```

## Backoffs

Backoffs are applied via a policy established by the client (defaults for request) or the backoff provided via the Request's Backoff method.

The backoffs run as long as the response is retriable. The retry behavior is prescribe via the `Retry` method, matching on the response codes provided.

```go
type fakeDoer struct {
	doCallCount int
	doFn        func(*http.Request) (*http.Response, error)
}

func (f *fakeDoer) Do(r *http.Request) (*http.Response, error) {
	f.doCallCount++
	return f.doFn(r)
}

func TestBackoff(t *testing.T) {
    t.Run("applies backoff on retry", func(t *testing.T) {
        doer := new(fakeDoer)
        doer.doFn = func(*http.Request) (*http.Response, error) {
            return stubResp(http.StatusNotFound), nil
        }

        boffer := httpc.NewConstantBackoff(time.Nanosecond, 3)
        client := httpc.New(doer, httpc.Backoff(boffer))

        err := client.
            DELETE("/foo").
            Success(httpc.StatusNoContent()).
            Retry(httpc.StatusNotFound()).
            Do(context.TODO())
        mustError(t, err)

        equals(t, true, retryErr(err))
        equals(t, 3, doer.doCallCount)
    })

    t.Run("does not backoff on non retry error", func(t *testing.T) {
        doer := new(fakeDoer)
        doer.doFn = func(*http.Request) (*http.Response, error) {
            return stubResp(http.StatusInternalServerError), nil
        }

        boffer := httpc.NewConstantBackoff(time.Nanosecond, 10)
        client := httpc.New(doer, httpc.Backoff(boffer))

        err := client.
            GET("/foo").
            Success(httpc.StatusOK()).
            Retry(httpc.StatusNotFound()).
            Do(context.TODO())
        mustError(t, err)

        equals(t, false, retryErr(err))
        equals(t, 1, doer.doCallCount)
    })
}

func retryErr(err error) bool {
	type retrier interface {
		Retry() bool
	}
	r, ok := err.(retrier)
	return ok && r.Retry()
}

func equals(t *testing.T, expected interface{}, actual interface{}) {
	t.Helper()
	if expected == actual {
		return
	}
	t.Errorf("expected: %v\tgot: %v", expected, actual)
}
```
