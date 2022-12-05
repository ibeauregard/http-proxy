package cache

import (
	"bufio"
	"io"
	"my_proxy/internal/errors_"
	"my_proxy/internal/http_"
	"net/http"
	"time"
)

type CacheableResponse struct {
	*http_.Response
}

type cacheFileInterface interface {
	open() *file
	delete()
	create() *file
	scheduleDeletion(time.Duration)
}

var newCacheFile = func(key string) cacheFileInterface {
	return &cacheFile{key}
}

func (r *CacheableResponse) Store(cacheKey string) {
	cacheLifespan := getCacheLifespan(r.Header)
	if cacheLifespan == 0 {
		return
	}
	cacheFile := newCacheFile(cacheKey)
	openCacheFile := cacheFile.create()
	if openCacheFile == nil {
		return
	}
	defer openCacheFile.close()
	if err := r.writeToCache(openCacheFile); err != nil {
		errors_.Log(r.Store, err)
		cacheFile.delete()
		return
	}
	index.store(cacheKey, timeDotNow().Add(cacheLifespan))
	cacheFile.scheduleDeletion(cacheLifespan)
}

func Retrieve(cacheKey string) *http_.Response {
	cacheFile := newCacheFile(cacheKey)
	openCacheFile := cacheFile.open()
	if openCacheFile == nil {
		return nil
	}
	response, err := newCacheResponseBuilder(openCacheFile).
		setStatusCode().
		setHeaders().
		setBody().
		build()
	if err != nil {
		index.remove(cacheKey)
		cacheFile.delete()
	}
	return response
}

type cacheEntryWriterInterface interface {
	writeStatusLine(proto string, statusCode int) error
	writeHeaders(headers http.Header) error
	writeBody(body io.Reader) error
	Flush() error
}

var newCacheEntryWriter = func(f io.Writer) cacheEntryWriterInterface {
	return &cacheEntryWriter{bufio.NewWriter(f)}
}

func (r *CacheableResponse) writeToCache(f io.Writer) error {
	w := newCacheEntryWriter(f)
	if err := w.writeStatusLine(r.Proto, r.StatusCode); err != nil {
		return errors_.Format(r.writeToCache, err)
	}
	if err := w.writeHeaders(r.Header); err != nil {
		return errors_.Format(r.writeToCache, err)
	}
	if err := w.writeBody(r.Body); err != nil {
		return errors_.Format(r.writeToCache, err)
	}
	if err := w.Flush(); err != nil {
		return errors_.Format(r.writeToCache, err)
	}
	return nil
}
