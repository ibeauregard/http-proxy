package response

import "net/http"

type Response struct {
	statusCode int
	headers    http.Header
	body       []byte
}

func NewResponse(statusCode int, headers http.Header, body []byte) *Response {
	return &Response{
		statusCode: statusCode,
		headers:    headers,
		body:       body,
	}
}

func (r *Response) GetStatusCode() int {
	return r.statusCode
}

func (r *Response) GetHeaders() http.Header {
	return r.headers
}

func (r *Response) GetBody() []byte {
	return r.body
}
