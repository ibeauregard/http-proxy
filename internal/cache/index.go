package cache

import (
	"encoding/gob"
	"my_proxy/internal/errors_"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var index = cacheIndex[string, time.Time]{map_: &sync.Map{}}

type cacheIndex[keyType ~string, valueType time.Time] struct {
	map_ *sync.Map
}

func (ci *cacheIndex[keyType, _]) contains(k keyType) bool {
	_, ok := ci.map_.Load(k)
	return ok
}

func (ci *cacheIndex[keyType, valueType]) store(k keyType, v valueType) {
	ci.map_.Store(k, v)
}

func (ci *cacheIndex[keyType, _]) remove(k keyType) {
	ci.map_.Delete(k)
}

func (ci *cacheIndex[keyType, valueType]) getMap() map[keyType]valueType {
	m := map[keyType]valueType{}
	ci.map_.Range(func(key, value any) bool {
		m[key.(keyType)] = value.(valueType)
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
