package cache

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"testing"
	"time"
)

func TestNewCacheFile(t *testing.T) {
	for _, key := range []string{"", "key1", "key2"} {
		assert.EqualValues(t, &cacheFile{key}, newCacheFile(key))
	}
}

func TestPath(t *testing.T) {
	tests := []struct {
		dirName  string
		key      string
		expected string
	}{
		{dirName: "cache/dir/name", key: "", expected: "cache/dir/name"},
		{dirName: "cache/dir/name", key: "key1", expected: "cache/dir/name/key1"},
		{dirName: "cache/dir/name", key: "key2", expected: "cache/dir/name/key2"},
		{dirName: "cache/dir/name/", key: "", expected: "cache/dir/name"},
		{dirName: "cache/dir/name/", key: "key1", expected: "cache/dir/name/key1"},
		{dirName: "cache/dir/name/", key: "key2", expected: "cache/dir/name/key2"},
		{dirName: "cache/dir/name//", key: "", expected: "cache/dir/name"},
		{dirName: "cache/dir/name//", key: "key1", expected: "cache/dir/name/key1"},
		{dirName: "cache/dir/name//", key: "key2", expected: "cache/dir/name/key2"},
		{dirName: "cache/dir//name/", key: "", expected: "cache/dir/name"},
		{dirName: "cache/dir//name/", key: "key1", expected: "cache/dir/name/key1"},
		{dirName: "cache/dir//name/", key: "key2", expected: "cache/dir/name/key2"},
	}
	for _, test := range tests {
		testName := fmt.Sprintf("cacheFile.path(), dirName=%q, key=%q", test.dirName, test.key)
		t.Run(testName, func(t *testing.T) {
			cacheDirName = test.dirName
			defer func() { cacheDirName = cacheDirNameBackup }()
			assert.EqualValues(t, test.expected, (&cacheFile{test.key}).path())
		})
	}
}

func TestCreateNoError(t *testing.T) {
	osFile := &os.File{}
	sysOpenFile = func(name string, flag int, perm os.FileMode) (*os.File, error) {
		return osFile, nil
	}
	var output *file
	assert.Empty(t, captureLog(func() {
		output = (&cacheFile{}).create()
	}))
	assert.EqualValues(t, &file{osFile}, output)
}

func TestCreateError(t *testing.T) {
	sysOpenFile = func(name string, flag int, perm os.FileMode) (*os.File, error) {
		return nil, errors.New("error")
	}
	var output *file
	assert.NotEmpty(t, captureLog(func() {
		output = (&cacheFile{}).create()
	}))
	assert.Nil(t, output)
}

func TestOpenKeyNotInIndex(t *testing.T) {
	var output *file
	assert.Empty(t, captureLog(func() {
		output = (&cacheFile{"non-indexed key"}).open()
	}))
	assert.Nil(t, output)
}

func TestOpenOsOpenFails(t *testing.T) {
	key := "key"
	index.store(key, time.Time{})
	sysOpen = func(name string) (io.ReadWriteCloser, error) {
		return nil, errors.New("error")
	}
	var output *file
	assert.NotEmpty(t, captureLog(func() {
		output = (&cacheFile{key}).open()
	}))
	assert.Nil(t, output)
}

func TestOpenSuccess(t *testing.T) {
	key, osFile := "key", &os.File{}
	index.store(key, time.Time{})
	sysOpen = func(name string) (io.ReadWriteCloser, error) {
		return osFile, nil
	}
	var output *file
	assert.Empty(t, captureLog(func() {
		output = (&cacheFile{key}).open()
	}))
	assert.EqualValues(t, &file{osFile}, output)
}

func TestDeleteSysRemoveFails(t *testing.T) {
	sysRemove = func(name string) error {
		return errors.New("error")
	}
	assert.NotEmpty(t, captureLog(func() {
		(&cacheFile{}).delete()
	}))
}

func TestDeleteSysRemoveSucceeds(t *testing.T) {
	sysRemove = func(name string) error {
		return nil
	}
	assert.Empty(t, captureLog(func() {
		(&cacheFile{}).delete()
	}))
}

func TestScheduleDeletion(t *testing.T) {
	sysRemove = func(name string) error {
		return nil
	}
	scheduledFunctions := map[time.Time][]func(){}
	lifespan := 10 * time.Second
	afterFunc = func(d time.Duration, f func()) *time.Timer {
		deletionTime := nowMock.Add(d)
		scheduledFunctions[deletionTime] = append(scheduledFunctions[deletionTime], f)
		f()
		return nil
	}
	(&cacheFile{}).scheduleDeletion(lifespan)
	assert.Equal(t, 1, len(scheduledFunctions[nowMock.Add(lifespan)]))
}

type osFileMock struct {
	fileInterface
	err error
}

func (m *osFileMock) Close() error {
	return m.err
}

func TestCloseSuccess(t *testing.T) {
	assert.Empty(t, captureLog(func() {
		(&file{&osFileMock{err: nil}}).close()
	}))
}

func TestCloseError(t *testing.T) {
	assert.NotEmpty(t, captureLog(func() {
		(&file{&osFileMock{err: errors.New("error")}}).close()
	}))
}
