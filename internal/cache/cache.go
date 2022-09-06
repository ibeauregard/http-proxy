package cache

import (
	"io"
	"my_proxy/internal/errors"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type cacheableResponse interface {
	getHeaders() http.Header
	writeToCache(io.Writer) error
}

func Store(r cacheableResponse, cacheKey string) {
	cacheLifespan := getCacheLifespan(r.getHeaders())
	if cacheLifespan == 0 {
		return
	}
	cacheFile := cacheFile{filepath.Join(os.Getenv("CACHE_DIR_NAME"), cacheKey)}
	openCacheFile, err := cacheFile.open()
	if err != nil {
		return
	}
	defer closeFile(openCacheFile)
	if err = r.writeToCache(openCacheFile); err != nil {
		errors.Log(Store, err)
		cacheFile.delete()
		return
	}
	cacheFile.scheduleDeletion(cacheLifespan)
}

type cacheFile struct {
	path string
}

func (f *cacheFile) open() (*os.File, error) {
	osFile, err := os.Create(f.path)
	if err != nil {
		errors.Log(f.open, err)
	}
	return osFile, err
}

func (f *cacheFile) delete() {
	if err := os.Remove(f.path); err != nil {
		errors.Log(f.delete, err)
	}
}

func (f *cacheFile) scheduleDeletion(lifespan time.Duration) {
	time.AfterFunc(lifespan, func() {
		f.delete()
	})
}

func closeFile(f io.Closer) {
	if err := f.Close(); err != nil {
		errors.Log(closeFile, err)
	}
}
