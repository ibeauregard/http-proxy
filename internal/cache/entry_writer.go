package cache

import (
	"fmt"
	"github.com/ibeauregard/http-proxy/internal/errors_"
	"io"
	"net/http"
)

type cacheEntryWriter struct {
	bufferedWriterInterface
}

type bufferedWriterInterface interface {
	WriteString(string) (int, error)
	Write(p []byte) (n int, err error)
	Flush() error
}

const crlf = "\r\n"

func (w *cacheEntryWriter) writeStatusLine(proto string, statusCode int) error {
	if _, err := w.WriteString(
		fmt.Sprintf("%s %d %s%s", proto, statusCode, http.StatusText(statusCode), crlf)); err != nil {
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
	if _, err := w.WriteString(fmt.Sprint("X-Cache", colonSpace, "HIT", crlf, crlf)); err != nil {
		return errors_.Format(w.writeHeaders, err)
	}
	return nil
}

func (w *cacheEntryWriter) writeBody(body io.Reader) error {
	if _, err := ioCopy(w, body); err != nil {
		return errors_.Format(w.writeBody, err)
	}
	return nil
}
