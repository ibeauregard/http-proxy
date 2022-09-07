package http

import (
	"io"
	"my_proxy/internal/errors"
	"net/http"
)

type Response struct {
	*http.Response
}

func NewResponse(r *http.Response) *Response {
	resp := &Response{r}
	resp.Header = getFilteredHeaders(r.Header)
	return resp
}

func (r *Response) Serve(writer http.ResponseWriter) {
	writeHeaders(writer, r.Header)
	writer.WriteHeader(r.StatusCode)

	_, err := io.Copy(writer, r.Body)
	if err != nil {
		errors.Log(r.Serve, err)
	}
}

func (r *Response) WithNewBody(body io.Reader) *Response {
	r.Body = io.NopCloser(body)
	return r
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
