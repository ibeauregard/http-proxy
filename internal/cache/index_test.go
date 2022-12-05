package cache

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"sync"
	"testing"
	"time"
)

func TestContainsAndStore(t *testing.T) {
	myMap := mapp[int, any]{&sync.Map{}}
	myMap.store(0, "0")
	myMap.store(2, "2")
	myMap.store(4, "4")
	for i := 0; i < 6; i++ {
		if i%2 == 0 {
			assert.True(t, myMap.contains(i))
		} else {
			assert.False(t, myMap.contains(i))
		}
	}
}

func TestRemove(t *testing.T) {
	myMap := mapp[int, any]{&sync.Map{}}
	myMap.store(42, "Foobar")
	assert.True(t, myMap.contains(42))
	myMap.remove(42)
	assert.False(t, myMap.contains(42))
}

func TestGetMap(t *testing.T) {
	for _, myMap := range []map[string]any{{}, {
		"42":  "24",
		"Foo": 123,
		"Bar": 456,
	}} {
		testName := fmt.Sprintf("mapp[K, V].getMap(), from %v", myMap)
		t.Run(testName, func(t *testing.T) {
			myMap2 := mapp[string, any]{&sync.Map{}}
			for k, v := range myMap {
				myMap2.store(k, v)
			}
			assert.EqualValues(t, myMap, myMap2.getMap())
		})
	}
}

type persistFileMock struct {
	*bytes.Buffer
	fileCloseError error
}

func (m *persistFileMock) Close() error {
	return m.fileCloseError
}

func TestPersistSuccess(t *testing.T) {
	mockFile := &persistFileMock{Buffer: &bytes.Buffer{}}
	sysCreate = func(_ string) (io.WriteCloser, error) {
		return mockFile, nil
	}
	index.store("0", nowMock)
	defer func() { index = newIndex() }()
	assert.Empty(t, captureLog(func() { Persist() }))
	m := map[string]time.Time{}
	gob.NewDecoder(mockFile).Decode(&m)
	assert.EqualValues(t, index.getMap(), m)
}

func TestPersistSuccessButFileCloseError(t *testing.T) {
	mockFile := &persistFileMock{&bytes.Buffer{}, errors.New("error")}
	sysCreate = func(_ string) (io.WriteCloser, error) {
		return mockFile, nil
	}
	assert.NotEmpty(t, captureLog(func() { Persist() }))
	m := map[string]time.Time{}
	gob.NewDecoder(mockFile).Decode(&m)
	assert.EqualValues(t, index.getMap(), m)
}

func TestPersistFileCreationError(t *testing.T) {
	sysCreate = func(_ string) (io.WriteCloser, error) {
		return nil, errors.New("error")
	}
	assert.NotEmpty(t, captureLog(func() { Persist() }))
}

type encodeDecodeErrorMock struct {
}

func (m *encodeDecodeErrorMock) Encode(_ any) error {
	return errors.New("error")
}

func (m *encodeDecodeErrorMock) Decode(_ any) error {
	return errors.New("error")
}

func TestPersistEncodingError(t *testing.T) {
	readWriter := &persistFileMock{}
	sysCreate = func(_ string) (io.WriteCloser, error) {
		return readWriter, nil
	}
	newEncoder = func(_ io.Writer) interface{ Encode(any) error } {
		return &encodeDecodeErrorMock{}
	}
	assert.NotEmpty(t, captureLog(func() { Persist() }))
}

func TestLoadSuccess(t *testing.T) {
	updateCache = func(m map[string]time.Time) {
		for key, deletionTime := range m {
			if deletionTime.After(time.Now()) {
				index.store(key, deletionTime)
			}
		}
	}
	defer func() {
		updateCache = updateCacheBackup
		index = newIndex()
	}()
	pastDate := time.Date(1776, 7, 4, 12, 0, 0, 0, time.UTC)
	futureDate := time.Date(6771, 7, 4, 12, 0, 0, 0, time.UTC)
	m := map[string]time.Time{
		"past":   pastDate,
		"future": futureDate,
	}
	mockFile := &persistFileMock{Buffer: &bytes.Buffer{}}
	gob.NewEncoder(mockFile).Encode(&m)
	sysOpen = func(_ string) (io.ReadWriteCloser, error) {
		return mockFile, nil
	}
	assert.Empty(t, captureLog(func() { Load() }))
	assert.EqualValues(t, map[string]time.Time{"future": futureDate}, index.getMap())
}

func TestLoadSuccessButFileCloseError(t *testing.T) {
	m := map[string]time.Time{}
	mockFile := &persistFileMock{&bytes.Buffer{}, errors.New("error")}
	gob.NewEncoder(mockFile).Encode(&m)
	sysOpen = func(_ string) (io.ReadWriteCloser, error) {
		return mockFile, nil
	}
	assert.NotEmpty(t, captureLog(func() { Load() }))
	assert.EqualValues(t, m, index.getMap())
}

func TestLoadFileCreationError(t *testing.T) {
	sysOpen = func(_ string) (io.ReadWriteCloser, error) {
		return nil, errors.New("error")
	}
	assert.NotEmpty(t, captureLog(func() { Load() }))
}

func TestLoadDecodingError(t *testing.T) {
	readWriter := &persistFileMock{}
	sysOpen = func(_ string) (io.ReadWriteCloser, error) {
		return readWriter, nil
	}
	newDecoder = func(_ io.Reader) interface{ Decode(any) error } {
		return &encodeDecodeErrorMock{}
	}
	assert.NotEmpty(t, captureLog(func() { Load() }))
}

func TestCacheFileFactory(t *testing.T) {
	key := "my_key"
	assert.EqualValues(t, &cacheFile{key}, newCacheFileForUpdateCache(key))
}

type cacheFileDeleterMock struct {
	key       string
	cache     map[string]struct{}
	callbacks map[time.Time]map[string]struct{}
}

func (mock *cacheFileDeleterMock) delete() {
	delete(mock.cache, mock.key)
}

func (mock *cacheFileDeleterMock) scheduleDeletion(lifetime time.Duration) {
	deletionTime := nowMock.Add(lifetime)
	if callbacks, ok := mock.callbacks[deletionTime]; ok {
		callbacks[mock.key] = struct{}{}
	} else {
		mock.callbacks[deletionTime] = map[string]struct{}{mock.key: {}}
	}
}

func TestUpdateCache(t *testing.T) {
	timeDotNow = func() time.Time {
		return nowMock
	}
	for _, cache := range []map[string]time.Time{
		{},
		{
			"f1": time.Date(1943, 4, 19, 0, 0, 0, 0, time.UTC),
			"f2": time.Date(1943, 4, 19, 0, 0, 0, 0, time.UTC),
		},
		{
			"f1": time.Date(2306, 2, 4, 0, 0, 0, 0, time.UTC),
			"f2": time.Date(2306, 2, 4, 0, 0, 0, 0, time.UTC),
		},
		{
			"f1": time.Date(1943, 4, 19, 0, 0, 0, 0, time.UTC),
			"f2": time.Date(2001, 7, 27, 0, 0, 0, 0, time.UTC),
			"f3": time.Date(2306, 2, 4, 0, 0, 0, 0, time.UTC),
		},
		{
			"f0": time.Date(2043, 4, 19, 0, 0, 0, 0, time.UTC),
			"f1": time.Date(1983, 4, 19, 0, 0, 0, 0, time.UTC),
			"f2": time.Date(2011, 4, 19, 0, 0, 0, 0, time.UTC),
			"f3": time.Date(1776, 4, 19, 0, 0, 0, 0, time.UTC),
			"f4": time.Date(2189, 4, 19, 0, 0, 0, 0, time.UTC),
			"f5": time.Date(2293, 4, 19, 0, 0, 0, 0, time.UTC),
			"f6": time.Date(1918, 4, 19, 0, 0, 0, 0, time.UTC),
			"f7": time.Date(1935, 4, 19, 0, 0, 0, 0, time.UTC),
			"f8": time.Date(2154, 4, 19, 0, 0, 0, 0, time.UTC),
			"f9": time.Date(1943, 4, 19, 0, 0, 0, 0, time.UTC),
		},
	} {
		testName := fmt.Sprintf("updateDate, cache=%v", cache)
		mockCache, mockCallbacks := map[string]struct{}{}, map[time.Time]map[string]struct{}{}
		expectedCache, expectedCallbacks := map[string]struct{}{}, map[time.Time]map[string]struct{}{}
		expectedIndex := map[string]time.Time{}
		for key, deletionTime := range cache {
			mockCache[key] = struct{}{}
			if deletionTime.After(nowMock) {
				expectedCache[key] = struct{}{}
				if callbacks, ok := expectedCallbacks[deletionTime]; ok {
					callbacks[key] = struct{}{}
				} else {
					expectedCallbacks[deletionTime] = map[string]struct{}{key: {}}
				}
				expectedIndex[key] = deletionTime
			}
		}
		newCacheFileForUpdateCache = func(cacheKey string) cacheFileInterfaceForUpdateCache {
			return &cacheFileDeleterMock{
				key:       cacheKey,
				cache:     mockCache,
				callbacks: mockCallbacks,
			}
		}
		t.Run(testName, func(t *testing.T) {
			updateCache(cache)
			defer func() { index = newIndex() }()
			assert.EqualValues(t, expectedCache, mockCache)
			assert.EqualValues(t, expectedCallbacks, mockCallbacks)
			assert.EqualValues(t, expectedIndex, index.getMap())
		})
	}
}
