package cache

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"io"
	"math"
	"net/http"
	"testing"
)

type writeStatusLineMock struct {
	i
	outputError error
}

func (m *writeStatusLineMock) WriteString(_ string) (int, error) {
	return 0, m.outputError
}

func TestWriteStatusLineNoError(t *testing.T) {
	assert.Nil(
		t,
		(&cacheEntryWriter{&writeStatusLineMock{outputError: nil}}).writeStatusLine("", 0))
}

func TestWriteStatusLineError(t *testing.T) {
	assert.Error(
		t,
		(&cacheEntryWriter{&writeStatusLineMock{outputError: errors.New("err")}}).
			writeStatusLine("", 0))
}

type writeHeadersMock struct {
	i
	numWriteStringCalls int
	failsAfter          int
}

func (m *writeHeadersMock) WriteString(_ string) (int, error) {
	if m.numWriteStringCalls == m.failsAfter {
		return 0, errors.New("error")
	}
	m.numWriteStringCalls++
	return 0, nil
}

var headers = http.Header{"key": []string{"values"}}

func TestWriteHeadersNoError(t *testing.T) {
	assert.Nil(
		t,
		(&cacheEntryWriter{&writeHeadersMock{
			failsAfter: math.MaxInt,
		}}).writeHeaders(headers))
}

func TestWriteHeadersError(t *testing.T) {
	for failsAfter := range [3]int{} {
		assert.Error(
			t,
			(&cacheEntryWriter{&writeHeadersMock{
				failsAfter: failsAfter,
			}}).writeHeaders(headers))
	}
}

func TestWriteBodyNoError(t *testing.T) {
	assert.Nil(t, (&cacheEntryWriter{}).writeBody(
		nil,
		func(_ io.Writer, _ io.Reader) (int64, error) {
			return 0, nil
		}))
}

func TestWriteBodyError(t *testing.T) {
	assert.Error(t, (&cacheEntryWriter{}).writeBody(
		nil,
		func(_ io.Writer, _ io.Reader) (int64, error) {
			return 0, errors.New("error")
		}))
}
