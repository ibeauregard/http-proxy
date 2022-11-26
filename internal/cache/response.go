package cache

import (
	"bufio"
	"io"
	"my_proxy/internal/errors_"
	"my_proxy/internal/http_"
)

type CacheableResponse struct {
	*http_.Response
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
	index.add(cacheFile.key)
	cacheFile.scheduleDeletion(cacheLifespan)
}

func Retrieve(cacheKey string) *http_.Response {
	cacheFile := newCacheFile(cacheKey)
	openCacheFile := cacheFile.open()
	if openCacheFile == nil {
		return nil
	}
	return newCacheResponseBuilder(openCacheFile).
		setStatusCode().
		setHeaders().
		setBody().
		build()
}

func (r *CacheableResponse) writeToCache(f io.Writer) error {
	w := cacheEntryWriter{bufio.NewWriter(f)}
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
