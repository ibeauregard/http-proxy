package http

import (
	"io"
	"my_proxy/internal/errors"
	"net/http"
)

type Response struct {
	Proto      string
	StatusCode int
	Headers    http.Header
	Body       io.Reader
}

func NewResponse(proto string, statusCode int, headers http.Header, body io.Reader) *Response {
	return &Response{
		Proto:      proto,
		StatusCode: statusCode,
		Headers:    headers,
		Body:       body,
	}
}

func (r *Response) Serve(writer http.ResponseWriter) {
	writeHeaders(writer, r.Headers)
	writer.WriteHeader(r.StatusCode)

	_, err := io.Copy(writer, r.Body)
	if err != nil {
		errors.Log(r.Serve, err)
	}
}

func (r *Response) WithNewBody(body io.Reader) *Response {
	r.Body = body
	return r
}

func writeHeaders(writer http.ResponseWriter, headers http.Header) {
	for name, values := range headers {
		writer.Header()[name] = values
	}
}
