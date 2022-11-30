package cache

import (
	"io"
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

var sysOpenFile = os.OpenFile

func (f *cacheFile) create() *file {
	// If O_CREAT and O_EXCL are set, open() shall fail if the file exists
	// If we don't set O_EXCL, a succession of requests very close in time
	// could prevent the cache entry from ever being completed.
	osFile, err := sysOpenFile(f.path(), os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		errors_.Log(f.create, err)
		return nil
	}
	return &file{osFile}
}

var sysOpen = os.Open

func (f *cacheFile) open() *file {
	if !index.contains(f.key) {
		return nil
	}
	osFile, err := sysOpen(f.path())
	if err != nil {
		errors_.Log(f.open, err)
		return nil
	}
	return &file{osFile}
}

var sysRemove = os.Remove

func (f *cacheFile) delete() {
	if err := sysRemove(f.path()); err != nil {
		errors_.Log(f.delete, err)
	}
}

var afterFunc = time.AfterFunc

func (f *cacheFile) scheduleDeletion(lifespan time.Duration) {
	afterFunc(lifespan, func() {
		index.remove(f.key)
		f.delete()
	})
}

type file struct {
	fileInterface
}

type fileInterface interface {
	io.ReadCloser
	io.Writer
}

func (f *file) close() {
	if err := f.Close(); err != nil {
		errors_.Log(f.close, err)
	}
}
