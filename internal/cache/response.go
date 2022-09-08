package cache

import (
	"bufio"
	"fmt"
	"io"
	"my_proxy/internal/errors"
	h "my_proxy/internal/http"
	"net/http"
	"os"
	"path/filepath"
)

type CacheableResponse struct {
	*h.Response
}

func (r *CacheableResponse) Store(cacheKey string) {
	cacheLifespan := getCacheLifespan(r.Header)
	if cacheLifespan == 0 {
		return
	}
	cacheFile := cacheFile{filepath.Join(os.Getenv("CACHE_DIR_NAME"), cacheKey)}
	openCacheFile, err := cacheFile.open()
	if err != nil {
		errors.Log(r.Store, err)
		return
	}
	defer closeFile(openCacheFile)
	if err = r.writeToCache(openCacheFile); err != nil {
		errors.Log(r.Store, err)
		cacheFile.delete()
		return
	}
	cacheFile.scheduleDeletion(cacheLifespan)
}

func (r *CacheableResponse) writeToCache(f io.Writer) error {
	w := cacheEntryWriter{bufio.NewWriter(f)}
	if err := w.writeStatusLine(r.Proto, r.StatusCode); err != nil {
		return errors.Format(r.writeToCache, err)
	}
	if err := w.writeHeaders(r.Header); err != nil {
		return errors.Format(r.writeToCache, err)
	}
	if err := w.writeBody(r.Body); err != nil {
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

func (w *cacheEntryWriter) writeBody(body io.Reader) error {
	if _, err := io.Copy(w, body); err != nil {
		return errors.Format(w.writeBody, err)
	}
	return nil
}
