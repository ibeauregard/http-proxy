package http_

import (
	"io"
	"my_proxy/internal/errors"
	"net/http"
)

type Response struct {
	*http.Response
	*Body
}

type Body struct {
	io.ReadCloser
}

func NewResponse(r *http.Response) *Response {
	resp := &Response{r, &Body{r.Body}}
	resp.Header = getFilteredHeaders(r.Header)
	return resp
}

func (r *Response) Serve(writer http.ResponseWriter) {
	defer r.Body.Close()
	writeHeaders(writer, r.Header)
	writer.WriteHeader(r.StatusCode)

	_, err := io.Copy(writer, r.Body)
	if err != nil {
		errors.Log(r.Serve, err)
	}
}

func (r *Response) WithNewBody(body io.ReadCloser) *Response {
	return &Response{r.Response, &Body{body}}
}

func (b *Body) Close() {
	if err := b.ReadCloser.Close(); err != nil {
		errors.Log(b.Close, err)
	}
}

func writeHeaders(writer http.ResponseWriter, headers http.Header) {
	for name, values := range headers {
		writer.Header()[name] = values
	}
}

var getFilteredHeaders = func() func(http.Header) http.Header {
	copiedHeaders := map[string]struct{}{
		"Content-Type":  {},
		"Cache-Control": {},
		"Date":          {},
		"Expires":       {},
		"Set-Cookie":    {},
	}
	return func(responseHeaders http.Header) http.Header {
		filteredHeaders := make(http.Header)
		for name, values := range responseHeaders {
			canonicalHeaderKey := http.CanonicalHeaderKey(name)
			if _, ok := copiedHeaders[canonicalHeaderKey]; ok {
				filteredHeaders[canonicalHeaderKey] = values
			}
		}
		filteredHeaders["Server"] = []string{"Ian's Proxy"}
		return filteredHeaders
	}
}()