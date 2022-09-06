package response

import "net/http"

type Response struct {
	proto      string
	statusCode int
	headers    http.Header
	body       []byte
}

func NewResponse(proto string, statusCode int, headers http.Header, body []byte) *Response {
	return &Response{
		proto:      proto,
		statusCode: statusCode,
		headers:    headers,
		body:       body,
	}
}

func (r *Response) GetProto() string {
	return r.proto
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
