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
}

func (m *persistFileMock) Close() error {
	return nil
}

func TestPersistSuccess(t *testing.T) {
	mockFile := &persistFileMock{&bytes.Buffer{}}
	sysCreate = func(_ string) (io.WriteCloser, error) {
		return mockFile, nil
	}
	index.store("0", time.Date(2022, time.November, 30, 23, 21, 43, 0, time.UTC))
	defer clearIndex(t, "0")
	assert.Empty(t, captureLog(func() { Persist() }))
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
	pastDate := time.Date(1776, time.July, 4, 12, 0, 0, 0, time.UTC)
	futureDate := time.Date(6771, time.July, 4, 12, 0, 0, 0, time.UTC)
	m := map[string]time.Time{
		"past":   pastDate,
		"future": futureDate,
	}
	mockFile := &persistFileMock{&bytes.Buffer{}}
	gob.NewEncoder(mockFile).Encode(&m)
	sysOpen = func(_ string) (io.ReadWriteCloser, error) {
		return mockFile, nil
	}
	assert.Empty(t, captureLog(func() { Load() }))
	defer clearIndex(t, "future")
	assert.EqualValues(t, map[string]time.Time{"future": futureDate}, index.getMap())
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
