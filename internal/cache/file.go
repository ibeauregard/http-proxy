package cache

import (
	"my_proxy/internal/errors_"
	"os"
	"path/filepath"
	"time"
)

type cacheFile struct {
	key string
}

var cacheDirName = os.Getenv("CACHE_DIR_NAME")

func newCacheFile(cacheKey string) *cacheFile {
	return &cacheFile{cacheKey}
}

func (f *cacheFile) path() string {
	return filepath.Join(cacheDirName, f.key)
}

func (f *cacheFile) create() *file {
	// If O_CREAT and O_EXCL are set, open() shall fail if the file exists
	// If we don't set O_EXCL, a succession of requests very close in time
	// could prevent the cache entry from ever being completed.
	osFile, err := os.OpenFile(f.path(), os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		errors_.Log(f.create, err)
		return nil
	}
	return &file{osFile}
}

func (f *cacheFile) open() *file {
	if !index.contains(f.key) {
		return nil
	}
	osFile, err := os.Open(f.path())
	if err != nil {
		errors_.Log(f.open, err)
		return nil
	}
	return &file{osFile}
}

func (f *cacheFile) delete() {
	if err := os.Remove(f.path()); err != nil {
		errors_.Log(f.delete, err)
	}
}

func (f *cacheFile) scheduleDeletion(lifespan time.Duration) {
	index.add(f.key)
	time.AfterFunc(lifespan, func() {
		index.remove(f.key)
		f.delete()
	})
}

type file struct {
	*os.File
}

func (f *file) close() {
	if err := f.File.Close(); err != nil {
		errors_.Log(f.close, err)
	}
}
