package cache

import (
	"bufio"
	"fmt"
	"io"
	"my_proxy/internal/errors"
	"net/http"
)

type CacheableResponse struct {
	proto      string
	statusCode int
	headers    http.Header
	body       []byte
}

func NewCacheableResponse(proto string, statusCode int, headers http.Header, body []byte) *CacheableResponse {
	return &CacheableResponse{
		proto:      proto,
		statusCode: statusCode,
		headers:    headers,
		body:       body,
	}
}

func (r *CacheableResponse) getHeaders() http.Header {
	return r.headers
}

func (r *CacheableResponse) writeToCache(f io.Writer) error {
	w := cacheEntryWriter{bufio.NewWriter(f)}
	if err := w.writeStatusLine(r.proto, r.statusCode); err != nil {
		return errors.Format(r.writeToCache, err)
	}
	if err := w.writeHeaders(r.headers); err != nil {
		return errors.Format(r.writeToCache, err)
	}
	if err := w.writeBody(r.body); err != nil {
		return errors.Format(r.writeToCache, err)
	}
	if err := w.Flush(); err != nil {
		return errors.Format(r.writeToCache, err)
	}
	return nil
}

type cacheEntryWriter struct {
	*bufio.Writer
}

var crlf = "\r\n"

func (w *cacheEntryWriter) writeStatusLine(proto string, statusCode int) error {
	if _, err := w.WriteString(
		fmt.Sprintf("%s %d %s %s", proto, statusCode, http.StatusText(statusCode), crlf)); err != nil {
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
