package cache

import (
	"my_proxy/internal/errors_"
	"os"
	"path/filepath"
	"time"
)

type cacheFile struct {
	path string
}

func newCacheFile(cacheKey string) *cacheFile {
	return &cacheFile{filepath.Join(os.Getenv("CACHE_DIR_NAME"), cacheKey)}
}

func (f *cacheFile) create() (*file, error) {
	osFile, err := os.Create(f.path)
	if err != nil {
		errors_.Log(f.create, err)
	}
	return &file{osFile}, err
}

func (f *cacheFile) open() (*file, error) {
	osFile, err := os.Open(f.path)
	if err != nil {
		return nil, err
	}
	return &file{osFile}, err
}

func (f *cacheFile) delete() {
	if err := os.Remove(f.path); err != nil {
		errors_.Log(f.delete, err)
	}
}

func (f *cacheFile) scheduleDeletion(lifespan time.Duration) {
	time.AfterFunc(lifespan, func() {
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
