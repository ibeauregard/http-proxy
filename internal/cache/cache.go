package cache

import (
	"bufio"
	"fmt"
	"io"
	"my_proxy/internal/errors"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// reorganize OOP?
// implement ResponseWriter?

func CacheResponse(headers http.Header, bodyBytes []byte, cacheKey string) {
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
	if err = writeToFile(f, headers, bodyBytes); err != nil {
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

func writeToFile(f *os.File, headers http.Header, bodyBytes []byte) error {
	w := bufio.NewWriter(f)
	if err := writeHeaders(w, headers); err != nil {
		return errors.Format(writeToFile, err)
	}
	if _, err := w.Write(bodyBytes); err != nil {
		return errors.Format(writeToFile, err)
	}
	if err := w.Flush(); err != nil {
		return errors.Format(writeToFile, err)
	}
	return nil
}

func writeHeaders(w io.StringWriter, headers http.Header) error {
	crlf, colonSpace := "\r\n", ": "
	for headerKey, headerValues := range headers {
		for _, headerValue := range headerValues {
			if _, err := w.WriteString(fmt.Sprint(headerKey, colonSpace, headerValue, crlf)); err != nil {
				return errors.Format(writeHeaders, err)
			}
		}
	}
	if _, err := w.WriteString(fmt.Sprint("X-CACHE", colonSpace, "HIT", crlf)); err != nil {
		return errors.Format(writeHeaders, err)
	}
	if _, err := w.WriteString(crlf); err != nil {
		return errors.Format(writeHeaders, err)
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
