package cache

import (
	"bufio"
	"fmt"
	"io"
	"my_proxy/internal/errors"
	"net/http"
)

type CacheableResponse struct {
	proto   string
	status  string
	headers http.Header
	body    []byte
}

func NewCacheableResponse(proto string, status string, headers http.Header, body []byte) *CacheableResponse {
	return &CacheableResponse{
		proto:   proto,
		status:  status,
		headers: headers,
		body:    body,
	}
}

func (r *CacheableResponse) getHeaders() http.Header {
	return r.headers
}

func (r *CacheableResponse) write(f io.Writer) error {
	w := cacheEntryWriter{bufio.NewWriter(f)}
	if err := w.writeStatusLine(r.proto, r.status); err != nil {
		return errors.Format(r.write, err)
	}
	if err := w.writeHeaders(r.headers); err != nil {
		return errors.Format(r.write, err)
	}
	if err := w.writeBody(r.body); err != nil {
		return errors.Format(r.write, err)
	}
	if err := w.Flush(); err != nil {
		return errors.Format(r.write, err)
	}
	return nil
}

type cacheEntryWriter struct {
	*bufio.Writer
}

var crlf = "\r\n"

func (w *cacheEntryWriter) writeStatusLine(proto, status string) error {
	if _, err := w.WriteString(fmt.Sprintf("%s %s %s", proto, status, crlf)); err != nil {
		return errors.Format(w.writeStatusLine, err)
	}
	return nil
}

func (w *cacheEntryWriter) writeHeaders(headers http.Header) error {
	colonSpace := ": "
	for headerKey, headerValues := range headers {
		for _, headerValue := range headerValues {
			if _, err := w.WriteString(fmt.Sprint(headerKey, colonSpace, headerValue, crlf)); err != nil {
				return errors.Format(w.writeHeaders, err)
			}
		}
	}
	if _, err := w.WriteString(fmt.Sprint("X-CACHE", colonSpace, "HIT", crlf)); err != nil {
		return errors.Format(w.writeHeaders, err)
	}
	if _, err := w.WriteString(crlf); err != nil {
		return errors.Format(w.writeHeaders, err)
	}
	return nil
}

func (w *cacheEntryWriter) writeBody(body []byte) error {
	if _, err := w.Write(body); err != nil {
		return errors.Format(w.writeBody, err)
	}
	return nil
}
