package cache

import (
	"my_proxy/internal/errors_"
	"path/filepath"
	"sync"
	"time"
)

var index = mapp[string, time.Time]{&sync.Map{}}

type mapp[K comparable, V any] struct {
	m mapInterface
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

// TODO: Once this is tested, we probably won't need to make it a package-global var
var updateCache = func(m map[string]time.Time) {
	for key, deletionTime := range m {
		now := time.Now()
		if deletionTime.Before(now) {
			newCacheFile(key).delete()
		} else {
			index.store(key, deletionTime)
			newCacheFile(key).scheduleDeletion(deletionTime.Sub(now))
		}
	}
}
