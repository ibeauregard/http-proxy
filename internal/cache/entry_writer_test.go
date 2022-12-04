package cache

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"math"
	"net/http"
	"strings"
	"testing"
)

type writeStatusLineMock struct {
	bufferedWriterInterface
	writtenLine string
	outputError error
}

func (m *writeStatusLineMock) WriteString(writtenLine string) (int, error) {
	m.writtenLine = writtenLine
	return 0, m.outputError
}

func TestWriteStatusLineNoError(t *testing.T) {
	tests := []struct {
		proto               string
		statusCode          int
		expectedWrittenLine string
	}{
		{proto: "", statusCode: 0, expectedWrittenLine: " 0 \r\n"},
		{proto: "P", statusCode: 0, expectedWrittenLine: "P 0 \r\n"},
		{proto: "P", statusCode: 1, expectedWrittenLine: "P 1 \r\n"},
		{proto: "P", statusCode: 100, expectedWrittenLine: "P 100 Continue\r\n"},
		{proto: "P", statusCode: 101, expectedWrittenLine: "P 101 Switching Protocols\r\n"},
		{proto: "P", statusCode: 102, expectedWrittenLine: "P 102 Processing\r\n"},
		{proto: "P", statusCode: 200, expectedWrittenLine: "P 200 OK\r\n"},
		{proto: "P", statusCode: 201, expectedWrittenLine: "P 201 Created\r\n"},
		{proto: "P", statusCode: 202, expectedWrittenLine: "P 202 Accepted\r\n"},
		{proto: "P", statusCode: 400, expectedWrittenLine: "P 400 Bad Request\r\n"},
		{proto: "P", statusCode: 401, expectedWrittenLine: "P 401 Unauthorized\r\n"},
		{proto: "P", statusCode: 402, expectedWrittenLine: "P 402 Payment Required\r\n"},
	}
	for _, test := range tests {
		testName := fmt.Sprintf(
			"cacheEntryWriter.writeStatusLine(proto=%q, statusCode=%d)",
			test.proto, test.statusCode)
		t.Run(testName, func(t *testing.T) {
			mock := writeStatusLineMock{outputError: nil}
			output := (&cacheEntryWriter{&mock}).writeStatusLine(test.proto, test.statusCode)
			assert.EqualValues(t, test.expectedWrittenLine, mock.writtenLine)
			assert.Nil(t, output)
		})
	}
}

func TestWriteStatusLineError(t *testing.T) {
	assert.Error(
		t,
		(&cacheEntryWriter{&writeStatusLineMock{outputError: errors.New("err")}}).
			writeStatusLine("", 0))
}

type writeHeadersMock struct {
	bufferedWriterInterface
	builder             strings.Builder
	numWriteStringCalls int
	failsAfter          int
}

func (m *writeHeadersMock) WriteString(s string) (int, error) {
	m.builder.WriteString(s)
	if m.numWriteStringCalls == m.failsAfter {
		return 0, errors.New("error")
	}
	m.numWriteStringCalls++
	return 0, nil
}

func TestWriteHeadersNoError(t *testing.T) {
	tests := []struct {
		headers                http.Header
		expectedWrittenHeaders []string
	}{
		{
			headers:                http.Header{},
			expectedWrittenHeaders: []string{"X-Cache: HIT\r\n\r\n"},
		},
		{
			headers:                http.Header{"key": []string{"value"}},
			expectedWrittenHeaders: []string{"key: value\r\nX-Cache: HIT\r\n\r\n"},
		},
		{
			headers:                http.Header{"key": []string{"val1", "val2"}},
			expectedWrittenHeaders: []string{"key: val1\r\nkey: val2\r\nX-Cache: HIT\r\n\r\n"},
		},
		{
			headers: http.Header{"key1": []string{"k1v1", "k1v2"}, "key2": []string{"k2v1", "k2v2"}},
			expectedWrittenHeaders: []string{
				"key1: k1v1\r\nkey1: k1v2\r\nkey2: k2v1\r\nkey2: k2v2\r\nX-Cache: HIT\r\n\r\n",
				"key2: k2v1\r\nkey2: k2v2\r\nkey1: k1v1\r\nkey1: k1v2\r\nX-Cache: HIT\r\n\r\n",
			},
		},
	}
	for _, test := range tests {
		testName := fmt.Sprintf("cacheEntryWriter.writeHeaders(headers=%v", test.headers)
		t.Run(testName, func(t *testing.T) {
			mock := writeHeadersMock{failsAfter: math.MaxInt}
			output := (&cacheEntryWriter{&mock}).writeHeaders(test.headers)
			writtenHeader := mock.builder.String()
			matchesAnyExpected := false
			for _, expected := range test.expectedWrittenHeaders {
				if writtenHeader == expected {
					matchesAnyExpected = true
					break
				}
			}
			assert.True(t, matchesAnyExpected)
			assert.Nil(t, output)
		})
	}
}

func TestWriteHeadersError(t *testing.T) {
	headers := http.Header{"key": []string{"values"}}
	for failsAfter, failContext := range []string{
		"while writing input headers",
		"while writing X-Cache header and final CRLF",
	} {
		testName := fmt.Sprintf("cacheEntryWriter.writeHeaders(), will fail %s", failContext)
		t.Run(testName, func(t *testing.T) {
			assert.Error(
				t,
				(&cacheEntryWriter{&writeHeadersMock{
					failsAfter: failsAfter,
				}}).writeHeaders(headers))
		})
	}
}

func TestWriteBodyNoError(t *testing.T) {
	for _, body := range []string{"", "body", "super long body"} {
		testName := fmt.Sprintf("cacheEntryWriter.writeBody(body=%q)", body)
		t.Run(testName, func(t *testing.T) {
			buf := &bytes.Buffer{}
			writer := bufio.NewWriter(buf)
			output := (&cacheEntryWriter{writer}).writeBody(strings.NewReader(body))
			writer.Flush()
			// Will not work with .EqualValues in the case of empty body; []byte{} vs []byte(nil)
			assert.True(t, bytes.Equal([]byte(body), buf.Bytes()))
			assert.Nil(t, output)
		})
	}
}

func TestWriteBodyError(t *testing.T) {
	ioCopy = func(_ io.Writer, _ io.Reader) (int64, error) {
		return 0, errors.New("error")
	}
	defer func() { ioCopy = ioCopyBackup }()
	assert.Error(t, (&cacheEntryWriter{}).writeBody(nil))
}
