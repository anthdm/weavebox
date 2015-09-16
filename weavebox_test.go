package weavebox

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/net/context"
)

var noopHandler = func(ctx *Context) error { return nil }

func TestHandle(t *testing.T) {
	w := New()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	for _, method := range []string{"GET", "PUT", "POST", "DELETE"} {
		w.Handle(method, "/", handler)
		code, _ := doRequest(t, method, "/", nil, w)
		isHTTPStatusOK(t, code)
	}
}

func TestMethodGet(t *testing.T) {
	w := New()
	w.Get("/", noopHandler)
	code, _ := doRequest(t, "GET", "/", nil, w)
	isHTTPStatusOK(t, code)
}

func TestMethodPost(t *testing.T) {
	w := New()
	w.Post("/", noopHandler)
	code, _ := doRequest(t, "POST", "/", nil, w)
	isHTTPStatusOK(t, code)
}

func TestMethodPut(t *testing.T) {
	w := New()
	w.Put("/", noopHandler)
	code, _ := doRequest(t, "PUT", "/", nil, w)
	isHTTPStatusOK(t, code)
}

func TestMethodDelete(t *testing.T) {
	w := New()
	w.Delete("/", noopHandler)
	code, _ := doRequest(t, "DELETE", "/", nil, w)
	isHTTPStatusOK(t, code)
}

func TestMethodHead(t *testing.T) {
	w := New()
	w.Head("/", noopHandler)
	code, _ := doRequest(t, "HEAD", "/", nil, w)
	isHTTPStatusOK(t, code)
}

func TestMethodOptions(t *testing.T) {
	w := New()
	w.Options("/", noopHandler)
	code, _ := doRequest(t, "OPTIONS", "/", nil, w)
	isHTTPStatusOK(t, code)
}

func TestBox(t *testing.T) {
	w := New()
	sr := w.Box("/foo")
	sr.Get("/bar", noopHandler)
	code, _ := doRequest(t, "GET", "/foo/bar", nil, w)
	isHTTPStatusOK(t, code)
}

func TestStatic(t *testing.T) {
	w := New()
	w.Static("/public", "./")
	code, body := doRequest(t, "GET", "/public/README.md", nil, w)
	isHTTPStatusOK(t, code)
	if len(body) == 0 {
		t.Error("body cannot be empty")
	}
	if !strings.Contains(body, "weavebox") {
		t.Error("expecting body containing string (weavebox)")
	}

	code, body = doRequest(t, "GET", "/public/nofile", nil, w)
	if code != http.StatusNotFound {
		t.Error("expecting status 404 got %d", code)
	}
}

func TestContext(t *testing.T) {
	w := New()
	w.Get("/", checkContext(t, "m1", "m1"))
	w.Use(func(next Handler) Handler {
		return func(c *Context) error {
			c.Context = context.WithValue(c.Context, "m1", "m1")
			return next(c)
		}
	})
	code, _ := doRequest(t, "GET", "/", nil, w)
	isHTTPStatusOK(t, code)

	w.Get("/some", checkContext(t, "m1", "m2"))
	w.Use(func(next Handler) Handler {
		return func(c *Context) error {
			c.Context = context.WithValue(c.Context, "m1", "m2")
			c.Response().WriteHeader(http.StatusBadRequest)
			return next(c)
		}
	})
	code, _ = doRequest(t, "GET", "/some", nil, w)
	if code != http.StatusBadRequest {
		t.Error("expecting %d, got %d", http.StatusBadRequest, code)
	}
}

func TestContextWithSubrouter(t *testing.T) {
	w := New()
	sub := w.Box("/test")
	sub.Get("/", checkContext(t, "a", "b"))
	sub.Use(func(next Handler) Handler {
		return func(c *Context) error {
			c.Context = context.WithValue(c.Context, "a", "b")
			return next(c)
		}
	})
	code, _ := doRequest(t, "GET", "/test", nil, w)
	if code != http.StatusOK {
		t.Errorf("expected status code 200 got %d", code)
	}
}

func TestBindContext(t *testing.T) {
	w := New()
	w.BindContext(context.WithValue(context.Background(), "a", "b"))

	w.Get("/", checkContext(t, "a", "b"))

	sub := w.Box("/foo")
	sub.Get("/", checkContext(t, "a", "b"))

	code, _ := doRequest(t, "GET", "/", nil, w)
	isHTTPStatusOK(t, code)
	code, _ = doRequest(t, "GET", "/foo", nil, w)
	isHTTPStatusOK(t, code)
}

func TestBindContextSubrouter(t *testing.T) {
	w := New()
	sub := w.Box("/foo")
	sub.Get("/", checkContext(t, "foo", "bar"))
	sub.BindContext(context.WithValue(context.Background(), "foo", "bar"))

	code, _ := doRequest(t, "GET", "/foo", nil, w)
	isHTTPStatusOK(t, code)
}

func checkContext(t *testing.T, key, expect string) Handler {
	return func(ctx *Context) error {
		value := ctx.Context.Value(key).(string)
		if value != expect {
			t.Errorf("expected %s got %s", expect, value)
		}
		return nil
	}
}

func TestMiddleware(t *testing.T) {
	buf := &bytes.Buffer{}
	w := New()
	w.Use(func(next Handler) Handler {
		return func(c *Context) error {
			buf.WriteString("a")
			return next(c)
		}
	})
	w.Use(func(next Handler) Handler {
		return func(c *Context) error {
			buf.WriteString("b")
			return next(c)
		}
	})
	w.Use(func(next Handler) Handler {
		return func(c *Context) error {
			buf.WriteString("c")
			return next(c)
		}
	})
	w.Use(func(next Handler) Handler {
		return func(c *Context) error {
			buf.WriteString("d")
			return next(c)
		}
	})

	w.Get("/", noopHandler)
	code, _ := doRequest(t, "GET", "/", nil, w)
	isHTTPStatusOK(t, code)
	if buf.String() != "abcd" {
		t.Error("expecting abcd got %s", buf.String())
	}
}

func TestBoxMiddlewareReset(t *testing.T) {
	buf := &bytes.Buffer{}
	w := New()

	w.Use(func(next Handler) Handler {
		return func(c *Context) error {
			buf.WriteString("a")
			return next(c)
		}
	})
	w.Use(func(next Handler) Handler {
		return func(c *Context) error {
			buf.WriteString("b")
			return next(c)
		}
	})

	sub := w.Box("/sub").ResetMiddleware()
	sub.Get("/", noopHandler)
	code, _ := doRequest(t, "GET", "/sub", nil, w)
	isHTTPStatusOK(t, code)
	if buf.String() != "" {
		t.Error("expecting empty buffer got %s", buf.String())
	}
}

func TestBoxMiddlewareInheritsParent(t *testing.T) {
	buf := &bytes.Buffer{}
	w := New()

	w.Use(func(next Handler) Handler {
		return func(c *Context) error {
			buf.WriteString("a")
			return next(c)
		}
	})
	w.Use(func(next Handler) Handler {
		return func(c *Context) error {
			buf.WriteString("b")
			return next(c)
		}
	})

	sub := w.Box("/sub")
	sub.Get("/", noopHandler)
	code, _ := doRequest(t, "GET", "/sub", nil, w)
	isHTTPStatusOK(t, code)
	if buf.String() != "ab" {
		t.Error("expecting ab got %s", buf.String())
	}
}

func TestParentNotInheritBoxMiddleware(t *testing.T) {
	buf := &bytes.Buffer{}
	w := New()
	w.Get("/foo", noopHandler)
	sub := w.Box("/sub")

	sub.Use(func(next Handler) Handler {
		return func(c *Context) error {
			buf.WriteString("a")
			return next(c)
		}
	})

	doRequest(t, "GET", "/foo", nil, w)
	if buf.String() == "a" {
		t.Error("parent cannot inherit box middleware")
	}
}

func TestErrorHandler(t *testing.T) {
	w := New()
	errorMsg := "oops! something went wrong"
	w.SetErrorHandler(func(ctx *Context, err error) {
		ctx.Response().WriteHeader(http.StatusInternalServerError)
		if err.Error() != errorMsg {
			t.Error("expecting %s, got %s", errorMsg, err.Error())
		}
	})

	w.Use(func(next Handler) Handler {
		return func(c *Context) error {
			return errors.New(errorMsg)
		}
	})

	w.Get("/", noopHandler)
	code, _ := doRequest(t, "GET", "/", nil, w)
	if code != http.StatusInternalServerError {
		t.Error("expecting code 500 got %s", code)
	}
}

func TestWeaveboxHandler(t *testing.T) {
	w := New()
	handle := func(respStr string) Handler {
		return func(ctx *Context) error {
			return ctx.Text(http.StatusOK, respStr)
		}
	}
	w.Get("/a", handle("a"))
	w.Get("/b", handle("b"))
	w.Get("/c", handle("c"))

	for _, r := range []string{"a", "b", "c"} {
		code, body := doRequest(t, "GET", "/"+r, nil, w)
		isHTTPStatusOK(t, code)
		if body != r {
			t.Errorf("expecting %s got %s", r, body)
		}
	}
}

func TestNotFoundHandler(t *testing.T) {
	w := New()
	code, body := doRequest(t, "GET", "/", nil, w)
	if code != http.StatusNotFound {
		t.Errorf("expecting code 404 got %d", code)
	}
	if !strings.Contains(body, "404 page not found") {
		t.Errorf("expecting body: 404 page not found got %s", body)
	}
}

func TestSetNotFound(t *testing.T) {
	w := New()
	notFoundMsg := "hey! not found"
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(notFoundMsg))
	})
	w.SetNotFoundHandler(h)

	code, body := doRequest(t, "GET", "/", nil, w)
	if code != http.StatusNotFound {
		t.Errorf("expecting code 404 got %d", code)
	}
	if !strings.Contains(body, notFoundMsg) {
		t.Errorf("expecting body: %s got %s", notFoundMsg, body)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	w := New()
	w.Get("/", noopHandler)
	code, body := doRequest(t, "POST", "/", nil, w)
	if code != http.StatusMethodNotAllowed {
		t.Errorf("expecting code 405 got %d", code)
	}
	if !strings.Contains(body, "Method Not Allowed") {
		t.Errorf("expecting body: Method Not Allowed got %s", body)
	}
}

func TestSetMethodNotAllowed(t *testing.T) {
	w := New()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("foo"))
	})
	w.SetMethodNotAllowed(handler)
	w.Get("/", noopHandler)

	code, body := doRequest(t, "POST", "/", nil, w)
	if code != http.StatusMethodNotAllowed {
		t.Errorf("expecting code 405 got %d", code)
	}
	if !strings.Contains(body, "foo") {
		t.Errorf("expecting body: foo got %s", body)
	}
}

func TestContextURLQuery(t *testing.T) {
	req, _ := http.NewRequest("GET", "/?name=anthony", nil)
	ctx := &Context{request: req}
	if ctx.Query("name") != "anthony" {
		t.Errorf("expected anthony got %s", ctx.Query("name"))
	}
}

func TestContextForm(t *testing.T) {
	values := url.Values{}
	values.Set("email", "john@gmail.com")
	req, _ := http.NewRequest("POST", "/", strings.NewReader(values.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	ctx := &Context{request: req}
	if ctx.Form("email") != "john@gmail.com" {
		t.Errorf("expected john@gmail.com got %s", ctx.Form("email"))
	}
}

func TestContextHeader(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add("x-test", "test")
	ctx := &Context{request: req}
	if ctx.Header("x-test") != "test" {
		t.Error("expected header to be (test) got %s", ctx.Header("x-test"))
	}
}

func TestSetHeader(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	resp := httptest.NewRecorder()
	ctx := &Context{request: req, response: resp}

	ctx.SetHeader("X-Test", "foo")
	ctx.SetHeader("X-Test", "bar")
	if want, have := "bar", ctx.response.Header().Get("X-Test"); want != have {
		t.Errorf("expecting %s have %s", want, have)
	}
}

func TestContextSetGet(t *testing.T) {
	w := New()
	w.Use(func(next Handler) Handler {
		return func(c *Context) error {
			c.Set("foo", "bar")
			return nil
		}
	})
	w.Use(func(next Handler) Handler {
		return func(c *Context) error {
			if want, have := "bar", c.Get("foo").(string); want != have {
				t.Error("expected %s but got %s", want, have)
			}
			return nil
		}
	})
	w.Get("/", noopHandler)
	code, _ := doRequest(t, "GET", "/", nil, w)
	isHTTPStatusOK(t, code)
}

func TestHTTPError(t *testing.T) {
	handler := func(code int, desc string) Handler {
		return func(c *Context) error {
			return c.HTTPError(code, desc)
		}
	}

	w := New()
	w.SetErrorHandler(func(c *Context, err error) {
		if httpErr, ok := err.(HTTPError); ok {
			c.JSON(httpErr.Code, httpErr)
		}
	})
	w.Get("/foo", handler(http.StatusBadRequest, "bad request"))

	code, body := doRequest(t, "GET", "/foo", nil, w)
	if code != http.StatusBadRequest {
		t.Error("expected %d but have %s", http.StatusBadRequest, code)
	}

	var httpErr HTTPError
	r := bytes.NewBufferString(body)
	if err := json.NewDecoder(r).Decode(&httpErr); err != nil {
		t.Error(err)
	}
	if want, have := "bad request", httpErr.Description; want != have {
		t.Error("expected %s but have %s", want, have)
	}
}

func isHTTPStatusOK(t *testing.T, code int) {
	if code != http.StatusOK {
		t.Errorf("Expecting status 200 got %d", code)
	}
}

func doRequest(t *testing.T, method, route string, body io.Reader, w *Weavebox) (int, string) {
	r, err := http.NewRequest(method, route, body)
	if err != nil {
		t.Fatal(err)
	}
	rw := httptest.NewRecorder()
	w.ServeHTTP(rw, r)
	return rw.Code, rw.Body.String()
}
