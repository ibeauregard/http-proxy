package cache

import (
	"my_proxy/internal/errors_"
	"path/filepath"
	"sync"
	"time"
)

var index = newIndex()

type mapp[K comparable, V any] struct {
	m mapInterface
}

func newIndex() *mapp[string, time.Time] {
	return &mapp[string, time.Time]{&sync.Map{}}
}

type mapInterface interface {
	Load(any) (any, bool)
	Store(key, value any)
	Delete(any)
	Range(func(key, value any) (shouldContinue bool))
}

func (m *mapp[K, _]) contains(k K) bool {
	_, ok := m.m.Load(k)
	return ok
}

func (m *mapp[K, V]) store(k K, v V) {
	m.m.Store(k, v)
}

func (m *mapp[K, _]) remove(k K) {
	m.m.Delete(k)
}

func (m *mapp[K, V]) getMap() map[K]V {
	mm := map[K]V{}
	m.m.Range(func(key, value any) bool {
		mm[key.(K)] = value.(V)
		return true
	})
	return mm
}

var cacheIndexPath = filepath.Join(cacheDirName, "index.gob")

func Persist() {
	file, err := sysCreate(cacheIndexPath)
	if err != nil {
		errors_.Log(Persist, err)
		return
	}
	// TODO: log this error
	defer file.Close()
	err = newEncoder(file).Encode(index.getMap())
	if err != nil {
		errors_.Log(Persist, err)
	}
}

func Load() {
	file, err := sysOpen(cacheIndexPath)
	if err != nil {
		errors_.Log(Load, err)
		return
	}
	// TODO: log this error
	defer file.Close()
	m := map[string]time.Time{}
	if err = newDecoder(file).Decode(&m); err != nil {
		errors_.Log(Load, err)
		return
	}
	updateCache(m)
}

type cacheFileDeleter interface {
	delete()
	scheduleDeletion(time.Duration)
}

var cacheFileFactory = func(cacheKey string) cacheFileDeleter {
	return newCacheFile(cacheKey)
}

var updateCache = func(m map[string]time.Time) {
	for key, deletionTime := range m {
		now := timeDotNow()
		if deletionTime.Before(now) {
			cacheFileFactory(key).delete()
		} else {
			index.store(key, deletionTime)
			cacheFileFactory(key).scheduleDeletion(deletionTime.Sub(now))
		}
	}
}
