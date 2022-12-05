package cache

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"my_proxy/internal/http_"
	"my_proxy/internal/tests"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestNewCacheFile(t *testing.T) {
	assert.NotNil(t, newCacheFile("key"))
}

type readWriteCloserMock struct {
	io.ReadWriter
}

func (_ *readWriteCloserMock) Close() error {
	return nil
}

type cacheFileMock struct {
	openFile             *file
	deleted              bool
	scheduledForDeletion bool
}

func (c *cacheFileMock) open() *file {
	return c.openFile
}

func (c *cacheFileMock) delete() {
	c.deleted = true
}

func (c *cacheFileMock) create() *file {
	return c.openFile
}

func (c *cacheFileMock) scheduleDeletion(_ time.Duration) {
	c.scheduledForDeletion = true
}

func TestStoreSuccess(t *testing.T) {
	defer func() { index = newIndex() }()
	key := "my_key"
	statusCode := 301
	proto := "HTTP/1.0"
	maxAge := 31
	body := "Response body"
	resp := &CacheableResponse{
		Response: &http_.Response{
			Response: &http.Response{
				StatusCode: statusCode,
				Proto:      proto,
				Header:     http.Header{"Cache-Control": {fmt.Sprintf("public, max-age=%d", maxAge)}},
			},
			Body: &http_.Body{ReadCloser: io.NopCloser(strings.NewReader(body))},
		},
	}
	buffer := &bytes.Buffer{}
	cacheFileMock := &cacheFileMock{openFile: &file{
		&readWriteCloserMock{
			buffer,
		},
	}}
	newCacheFile = func(_ string) cacheFileInterface {
		return cacheFileMock
	}
	timeDotNow = func() time.Time {
		return nowMock
	}
	expectedCacheFileContent := strings.Join([]string{
		"HTTP/1.0 301 " + http.StatusText(statusCode),
		fmt.Sprintf("Cache-Control: public, max-age=%d", maxAge),
		"X-Cache: HIT",
		"",
		body,
	}, crlf)
	expectedDeletionTime := nowMock.Add(time.Duration(maxAge) * time.Second)
	assert.Empty(t, tests.CaptureLog(func() { resp.Store(key) }))
	assert.Equal(t, expectedCacheFileContent, buffer.String())
	assert.Equal(t, expectedDeletionTime, index.getMap()[key])
	assert.True(t, cacheFileMock.scheduledForDeletion)
}

func TestStoreNonCacheableResponse(t *testing.T) {
	key := "my_key"
	resp := &CacheableResponse{
		Response: &http_.Response{
			Response: &http.Response{
				Header: http.Header{"Cache-Control": {"no-cache"}},
			},
		},
	}
	newCacheFile = func(_ string) cacheFileInterface {
		assert.Fail(t, "newCacheFile() should not be called in this scenario; no cache file to create")
		return nil
	}
	assert.Empty(t, tests.CaptureLog(func() { resp.Store(key) }))
	assert.False(t, index.contains(key))
}

func TestStoreCacheFileCreationError(t *testing.T) {
	key := "my_key"
	resp := &CacheableResponse{
		Response: &http_.Response{
			Response: &http.Response{
				Header: http.Header{"Cache-Control": {"public, max-age=33"}},
			},
		},
	}
	cacheFileMock := &cacheFileMock{}
	newCacheFile = func(_ string) cacheFileInterface {
		return cacheFileMock
	}
	assert.Empty(t, tests.CaptureLog(func() { resp.Store(key) }))
	assert.False(t, index.contains(key))
	assert.False(t, cacheFileMock.scheduledForDeletion)
}

func TestStoreWriteToCacheError(t *testing.T) {
	key := "my_key"
	resp := &CacheableResponse{
		Response: &http_.Response{
			Response: &http.Response{
				Header: http.Header{"Cache-Control": {"public, max-age=33"}}}}}
	cacheFileMock := &cacheFileMock{openFile: &file{
		&readWriteCloserMock{},
	}}
	newCacheFile = func(_ string) cacheFileInterface {
		return cacheFileMock
	}
	newCacheEntryWriter = func(f io.Writer) cacheEntryWriterInterface {
		return &cacheEntryWriterMock{
			writeStatusLineError: errors.New("error"),
		}
	}
	defer func() { newCacheEntryWriter = newCacheEntryWriterBackup }()
	assert.NotEmpty(t, tests.CaptureLog(func() { resp.Store(key) }))
	assert.False(t, index.contains(key))
	assert.False(t, cacheFileMock.scheduledForDeletion)
	assert.True(t, cacheFileMock.deleted)
}

func TestRetrieveSuccess(t *testing.T) {
	key := "my_key"
	index.store(key, time.Time{})
	defer func() { index = newIndex() }()
	timestamp := "Sun, 04 Dec 2022 22:59:59 GMT"
	cacheFileContent := strings.Join([]string{
		"HTTP/1.0 301 Moved Permanently",
		"Cache-Control: public",
		fmt.Sprintf("Date: %s", timestamp),
		"X-Cache: HIT",
		"",
		"Response body",
	}, crlf)
	cacheFileMock := &cacheFileMock{openFile: &file{
		&readWriteCloserMock{
			bytes.NewBufferString(cacheFileContent),
		},
	}}
	newCacheFile = func(_ string) cacheFileInterface {
		return cacheFileMock
	}
	now, _ := time.Parse(time.RFC1123, timestamp)
	age := 23
	now = now.Add(time.Duration(age) * time.Second)
	timeSince = func(t time.Time) time.Duration {
		return now.Sub(t)
	}
	response := Retrieve(key)
	assert.True(t, index.contains(key))
	assert.False(t, cacheFileMock.deleted)
	assert.Equal(t, 301, response.StatusCode)
	assert.Equal(t, http.Header{
		"Cache-Control": {"public"},
		"X-Cache":       {"HIT"},
		"Date":          {timestamp},
		"Age":           {strconv.Itoa(age)},
	}, response.Header)
	writer := &strings.Builder{}
	io.Copy(writer, response.Body)
	assert.Equal(t, "Response body", writer.String())
}

func TestRetrieveNoCacheEntry(t *testing.T) {
	newCacheFile = func(_ string) cacheFileInterface {
		return &cacheFileMock{}
	}
	assert.Nil(t, Retrieve("key"))
}

func TestRetrieveResponseBuildingError(t *testing.T) {
	key := "my_key"
	index.store(key, time.Time{})
	defer func() { index = newIndex() }()
	cacheFileContent := "Invalid content"
	mock := &cacheFileMock{openFile: &file{
		&readWriteCloserMock{
			bytes.NewBufferString(cacheFileContent),
		},
	}}
	newCacheFile = func(_ string) cacheFileInterface {
		return mock
	}
	assert.Nil(t, Retrieve(key))
	assert.False(t, index.contains(key))
	assert.True(t, mock.deleted)
}

func TestNewCacheEntryWriter(t *testing.T) {
	assert.NotNil(t, newCacheEntryWriter(nil))
}

type cacheEntryWriterMock struct {
	*cacheEntryWriter
	writeStatusLineError error
	writeHeadersError    error
	writeBodyError       error
	flushError           error
}

func (c *cacheEntryWriterMock) writeStatusLine(proto string, statusCode int) error {
	if c.writeStatusLineError != nil {
		return c.writeStatusLineError
	}
	return c.cacheEntryWriter.writeStatusLine(proto, statusCode)
}

func (c *cacheEntryWriterMock) writeHeaders(headers http.Header) error {
	if c.writeHeadersError != nil {
		return c.writeHeadersError
	}
	return c.cacheEntryWriter.writeHeaders(headers)
}

func (c *cacheEntryWriterMock) writeBody(body io.Reader) error {
	if c.writeBodyError != nil {
		return c.writeBodyError
	}
	return c.cacheEntryWriter.writeBody(body)
}

func (c *cacheEntryWriterMock) Flush() error {
	if c.flushError != nil {
		return c.flushError
	}
	return c.cacheEntryWriter.Flush()
}

func TestWriteToCacheSuccess(t *testing.T) {
	resp := &CacheableResponse{
		Response: &http_.Response{
			Response: &http.Response{
				StatusCode: 301,
				Proto:      "HTTP/1.0",
				Header:     http.Header{"Cache-Control": {"public"}},
			},
			Body: &http_.Body{ReadCloser: io.NopCloser(strings.NewReader("Response body"))}},
	}
	writer := &strings.Builder{}
	newCacheEntryWriter = func(f io.Writer) cacheEntryWriterInterface {
		return &cacheEntryWriterMock{cacheEntryWriter: &cacheEntryWriter{bufio.NewWriter(f)}}
	}
	defer func() { newCacheEntryWriter = newCacheEntryWriterBackup }()
	expectedWriter := strings.Join([]string{
		"HTTP/1.0 301 Moved Permanently",
		"Cache-Control: public",
		"X-Cache: HIT",
		"",
		"Response body",
	}, crlf)
	assert.Nil(t, resp.writeToCache(writer))
	assert.Equal(t, expectedWriter, writer.String())
}

type nopWriter struct {
}

func (_ *nopWriter) Write(_ []byte) (int, error) {
	return 0, nil
}

func TestWriteToCacheError(t *testing.T) {
	resp := &CacheableResponse{
		Response: &http_.Response{
			Response: &http.Response{},
			Body:     &http_.Body{ReadCloser: io.NopCloser(strings.NewReader(""))}},
	}
	writer := &nopWriter{}
	cacheEntryWriter := &cacheEntryWriter{bufio.NewWriter(writer)}
	defer func() { newCacheEntryWriter = newCacheEntryWriterBackup }()
	for _, mock := range []*cacheEntryWriterMock{
		{
			cacheEntryWriter:     cacheEntryWriter,
			writeStatusLineError: errors.New("error"),
		},
		{
			cacheEntryWriter:  cacheEntryWriter,
			writeHeadersError: errors.New("error"),
		},
		{
			cacheEntryWriter: cacheEntryWriter,
			writeBodyError:   errors.New("error"),
		},
		{
			cacheEntryWriter: cacheEntryWriter,
			flushError:       errors.New("error"),
		},
	} {
		newCacheEntryWriter = func(_ io.Writer) cacheEntryWriterInterface {
			return mock
		}
		assert.NotNil(t, resp.writeToCache(writer))
	}
}
