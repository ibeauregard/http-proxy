package http

import (
	"my_proxy/internal/errors"
	"net/http"
)

type Response struct {
	Proto      string
	StatusCode int
	Headers    http.Header
	Body       []byte
}

func (r *Response) Serve(writer http.ResponseWriter) {
	writeHeaders(writer, r.Headers)
	writer.WriteHeader(r.StatusCode)

	_, err := writer.Write(r.Body)
	if err != nil {
		errors.Log(r.Serve, err)
	}
}

func writeHeaders(writer http.ResponseWriter, headers http.Header) {
	for name, values := range headers {
		writer.Header()[name] = values
	}
}
