package cache

import (
	"bufio"
	"fmt"
	"io"
	"my_proxy/internal/errors"
	r "my_proxy/internal/response"
	"net/http"
)

type cacheableResponseImpl struct {
	proto string
	*r.Response
}

func NewCacheableResponse(proto string, r *r.Response) *cacheableResponseImpl {
	return &cacheableResponseImpl{
		proto:    proto,
		Response: r,
	}
}

func (r *cacheableResponseImpl) getHeaders() http.Header {
	return r.GetHeaders()
}

func (r *cacheableResponseImpl) writeToCache(f io.Writer) error {
	w := cacheEntryWriter{bufio.NewWriter(f)}
	if err := w.writeStatusLine(r.proto, r.GetStatusCode()); err != nil {
		return errors.Format(r.writeToCache, err)
	}
	if err := w.writeHeaders(r.GetHeaders()); err != nil {
		return errors.Format(r.writeToCache, err)
	}
	if err := w.writeBody(r.GetBody()); err != nil {
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
