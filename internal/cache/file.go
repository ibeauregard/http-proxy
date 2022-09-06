package cache

import (
	"io"
	"my_proxy/internal/errors"
	"os"
	"time"
)

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
