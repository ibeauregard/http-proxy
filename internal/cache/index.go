package cache

import (
	"encoding/gob"
	"my_proxy/internal/errors_"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var index = syncMap[string, time.Time]{map_: &sync.Map{}}

type syncMap[K comparable, V any] struct {
	map_ *sync.Map
}

func (sm *syncMap[K, _]) contains(k K) bool {
	_, ok := sm.map_.Load(k)
	return ok
}

func (sm *syncMap[K, V]) store(k K, v V) {
	sm.map_.Store(k, v)
}

func (sm *syncMap[K, _]) remove(k K) {
	sm.map_.Delete(k)
}

func (sm *syncMap[K, V]) getMap() map[K]V {
	m := map[K]V{}
	sm.map_.Range(func(key, value any) bool {
		m[key.(K)] = value.(V)
		return true
	})
	return m
}

var cacheIndexPath = filepath.Join(cacheDirName, "index.gob")

func Persist() {
	file, err := os.Create(cacheIndexPath)
	if err != nil {
		errors_.Log(Persist, err)
		return
	}
	defer file.Close()
	err = gob.NewEncoder(file).Encode(index.getMap())
	if err != nil {
		errors_.Log(Persist, err)
	}
}

func Load() {
	file, err := os.Open(cacheIndexPath)
	if err != nil {
		errors_.Log(Load, err)
		return
	}
	defer file.Close()
	m := map[string]time.Time{}
	if err = gob.NewDecoder(file).Decode(&m); err != nil {
		errors_.Log(Load, err)
		return
	}
	updateCache(m)
}

func updateCache(m map[string]time.Time) {
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
