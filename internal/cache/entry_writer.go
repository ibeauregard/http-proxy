package cache

import (
	"fmt"
	"io"
	"my_proxy/internal/errors_"
	"net/http"
)

type cacheEntryWriter struct {
	i
}

type i interface {
	WriteString(string) (int, error)
	Write(p []byte) (n int, err error)
	Flush() error
}

const crlf = "\r\n"

func (w *cacheEntryWriter) writeStatusLine(proto string, statusCode int) error {
	if _, err := w.WriteString(
		fmt.Sprintf("%s %d %s %s", proto, statusCode, http.StatusText(statusCode), crlf)); err != nil {
		return errors_.Format(w.writeStatusLine, err)
	}
	return nil
}

func (w *cacheEntryWriter) writeHeaders(headers http.Header) error {
	colonSpace := ": "
	for headerKey, headerValues := range headers {
		for _, headerValue := range headerValues {
			if _, err := w.WriteString(fmt.Sprint(headerKey, colonSpace, headerValue, crlf)); err != nil {
				return errors_.Format(w.writeHeaders, err)
			}
		}
	}
	if _, err := w.WriteString(fmt.Sprint("X-Cache", colonSpace, "HIT", crlf)); err != nil {
		return errors_.Format(w.writeHeaders, err)
	}
	if _, err := w.WriteString(crlf); err != nil {
		return errors_.Format(w.writeHeaders, err)
	}
	return nil
}

type copyFunc func(io.Writer, io.Reader) (int64, error)

func (w *cacheEntryWriter) writeBody(body io.Reader, copy copyFunc) error {
	if _, err := copy(w, body); err != nil {
		return errors_.Format(w.writeBody, err)
	}
	return nil
}
