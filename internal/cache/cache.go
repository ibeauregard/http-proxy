package cache

import (
	"bufio"
	"fmt"
	"my_proxy/internal/errors"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// reorganize OOP?
// implement ResponseWriter?

func CacheResponse(proto string, status string, headers http.Header, bodyBytes []byte, cacheKey string) {
	cacheLifespan := getCacheLifespan(headers)
	if cacheLifespan == 0 {
		return
	}
	filePath := filepath.Join(os.Getenv("CACHE_DIR_NAME"), cacheKey)
	f, err := openFile(filePath)
	if err != nil {
		return
	}
	defer closeFile(f)
	if err = writeToFile(f, proto, status, headers, bodyBytes); err != nil {
		errors.Log(CacheResponse, err)
		deleteFile(filePath)
		return
	}
	scheduleDeletion(cacheLifespan, filePath)
}

func openFile(path string) (*os.File, error) {
	f, err := os.Create(path)
	if err != nil {
		errors.Log(openFile, err)
	}
	return f, err
}

func writeToFile(f *os.File, proto string, status string, headers http.Header, bodyBytes []byte) error {
	w := cacheEntryWriter{bufio.NewWriter(f)}
	if err := w.writeStatusLine(proto, status); err != nil {
		return errors.Format(writeToFile, err)
	}
	if err := w.writeHeaders(headers); err != nil {
		return errors.Format(writeToFile, err)
	}
	if err := w.writeBody(bodyBytes); err != nil {
		return errors.Format(writeToFile, err)
	}
	if err := w.Flush(); err != nil {
		return errors.Format(writeToFile, err)
	}
	return nil
}

func closeFile(f *os.File) {
	if err := f.Close(); err != nil {
		errors.Log(closeFile, err)
	}
}

func deleteFile(path string) {
	if err := os.Remove(path); err != nil {
		errors.Log(deleteFile, err)
	}
}

func scheduleDeletion(lifespan time.Duration, filePath string) {
	time.AfterFunc(lifespan, func() {
		deleteFile(filePath)
	})
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
