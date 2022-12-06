package http_

import (
	"io"
	"my_proxy/internal/errors_"
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

var ioCopy = io.Copy

func (r *Response) Serve(writer http.ResponseWriter) {
	defer r.Body.Close()
	writeHeaders(writer, r.Header)
	writer.WriteHeader(r.StatusCode)

	_, err := ioCopy(writer, r.Body)
	if err != nil {
		errors_.Log(r.Serve, err)
	}
}

func (r *Response) WithBody(body io.Reader) *Response {
	readCloserBody, ok := body.(io.ReadCloser)
	if !ok {
		readCloserBody = io.NopCloser(body)
	}
	return &Response{r.Response, &Body{readCloserBody}}
}

func (b *Body) Close() {
	if err := b.ReadCloser.Close(); err != nil {
		errors_.Log(b.Close, err)
	}
}

func writeHeaders(writer http.ResponseWriter, headers http.Header) {
	for name, values := range headers {
		writer.Header()[name] = values
	}
}

var copiedHeaders = map[string]struct{}{
	"Content-Type":  {},
	"Cache-Control": {},
	"Date":          {},
	"Expires":       {},
	"Set-Cookie":    {},
}

func getFilteredHeaders(responseHeaders http.Header) http.Header {
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
