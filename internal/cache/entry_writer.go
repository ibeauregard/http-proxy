package cache

import (
	"bufio"
	"fmt"
	"io"
	"my_proxy/internal/errors"
	"net/http"
)

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
	if _, err := w.WriteString(fmt.Sprint("X-Cache", colonSpace, "HIT", crlf)); err != nil {
		return errors.Format(w.writeHeaders, err)
	}
	if _, err := w.WriteString(crlf); err != nil {
		return errors.Format(w.writeHeaders, err)
	}
	return nil
}

func (w *cacheEntryWriter) writeBody(body io.Reader) error {
	if _, err := io.Copy(w, body); err != nil {
		return errors.Format(w.writeBody, err)
	}
	return nil
}
