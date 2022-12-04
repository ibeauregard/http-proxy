package cache

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"math/rand"
	"my_proxy/internal/http_"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestNewCacheResponseBuilder(t *testing.T) {
	contentSize := 10
	randomContent := make([]byte, contentSize)
	rand.Seed(time.Now().Unix())
	rand.Read(randomContent)
	readCloser := io.NopCloser(bytes.NewReader(randomContent))
	builder := newCacheResponseBuilder(readCloser)
	assert.NotNil(t, builder.response.Response)
	assert.Equal(t, readCloser, builder.reader.Closer)
	writer := make([]byte, contentSize)
	_, _ = io.ReadFull(readCloser, writer)
	assert.Equal(t, randomContent, writer)
}

func TestSetStatusCodeSuccess(t *testing.T) {
	entry := strings.Join([]string{
		"HTTP/1.1 200 OK",
		"Content-Type: application/json; charset=utf-8",
		"Date: Sat, 03 Dec 2022 23:25:26 GMT",
		"",
		"Response Body",
	}, crlf)
	expectedStatusCode := 200
	builder := &cacheResponseBuilder{
		response: &http_.Response{
			Response: &http.Response{},
		},
		reader: &cacheEntryReader{
			Reader: bufio.NewReader(strings.NewReader(entry)),
		},
	}
	assert.Same(t, builder, builder.setStatusCode())
	assert.Nil(t, builder.err)
	assert.EqualValues(t, expectedStatusCode, builder.response.StatusCode)
}

func TestSetStatusCodeError(t *testing.T) {
	for _, entry := range []string{
		"",
		"no new line character",
		"http/1.1 42 invalid\r\nCache-Control: public\r\n\r\nBody",
	} {
		testName := "setStatusCode, entry=" + entry
		t.Run(testName, func(t *testing.T) {
			builder := &cacheResponseBuilder{
				response: &http_.Response{
					Response: &http.Response{},
				},
				reader: &cacheEntryReader{
					Reader: bufio.NewReader(strings.NewReader(entry)),
				},
			}
			assert.Same(t, builder, builder.setStatusCode())
			assert.NotNil(t, builder.err)
		})
	}
}

func TestSetHeadersSuccess(t *testing.T) {
	entry := strings.Join([]string{
		"Content-Type: application/json; charset=utf-8",
		"Date: Sat, 03 Dec 2022 23:25:26 GMT",
		"",
		"Response Body",
	}, crlf)
	expectedHeaders := http.Header{
		"Content-Type": {"application/json; charset=utf-8"},
		"Date":         {"Sat, 03 Dec 2022 23:25:26 GMT"},
		"Age":          {"30"},
	}
	now, _ := time.Parse(time.RFC1123, "Sat, 03 Dec 2022 23:25:56 GMT")
	timeSince = func(t time.Time) time.Duration {
		return now.Sub(t)
	}
	builder := &cacheResponseBuilder{
		response: &http_.Response{
			Response: &http.Response{},
		},
		reader: &cacheEntryReader{
			Reader: bufio.NewReader(strings.NewReader(entry)),
		},
	}
	assert.Same(t, builder, builder.setHeaders())
	assert.Nil(t, builder.err)
	assert.EqualValues(t, expectedHeaders, builder.response.Header)
}

func TestSetHeadersBuilderErrorIsNotNil(t *testing.T) {
	builder := &cacheResponseBuilder{
		err: errors.New(""),
	}
	assert.Same(t, builder, builder.setHeaders())
	assert.NotNil(t, builder.err)
}

func TestSetHeadersError(t *testing.T) {
	for _, entry := range []string{
		"Invalid header\r\nCache-Control: public\r\n\r\nResponse Body",
		"Problem: date missing\r\n\r\nResponse Body",
	} {
		testName := "setHeaders(), entry=" + entry
		t.Run(testName, func(t *testing.T) {
			builder := &cacheResponseBuilder{
				response: &http_.Response{
					Response: &http.Response{},
				},
				reader: &cacheEntryReader{
					Reader: bufio.NewReader(strings.NewReader(entry)),
				},
			}
			assert.Same(t, builder, builder.setHeaders())
			assert.NotNil(t, builder.err)
		})
	}
}

func TestSetBodyBuilderErrorIsNil(t *testing.T) {
	reader := &cacheEntryReader{}
	builder := &cacheResponseBuilder{
		response: &http_.Response{},
		reader:   reader,
		err:      nil,
	}
	assert.Same(t, builder, builder.setBody())
	assert.Nil(t, builder.err)
	assert.NotNil(t, builder.response.Body)
	assert.Same(t, reader, builder.response.Body.ReadCloser)
}

func TestSetBodyBuilderErrorIsNotNil(t *testing.T) {
	builder := &cacheResponseBuilder{
		response: &http_.Response{},
		err:      errors.New(""),
	}
	assert.Same(t, builder, builder.setBody())
	assert.NotNil(t, builder.err)
	assert.Nil(t, builder.response.Body)
}

func TestBuildBuilderErrorIsNil(t *testing.T) {
	response := &http_.Response{}
	builder := &cacheResponseBuilder{
		response: response,
		err:      nil,
	}
	r, err := builder.build()
	assert.Same(t, response, r)
	assert.Nil(t, err)
}

func TestBuildBuilderErrorIsNotNil(t *testing.T) {
	builder := &cacheResponseBuilder{
		response: &http_.Response{},
		err:      errors.New("error"),
	}
	r, err := builder.build()
	assert.Nil(t, r)
	assert.NotNil(t, err)
}

func TestSetCachedHeadersSuccess(t *testing.T) {
	entry := strings.Join([]string{
		"Age: 2",
		"Cache-Control: public, max-age=86400, s-maxage=86400",
		"Content-Type: application/json; charset=utf-8",
		"Date: Sat, 03 Dec 2022 23:25:26 GMT",
		"Server: Ian's Proxy",
		"X-Cache: HIT",
		"Transfer-Encoding: chunked",
		"",
		"Response Body",
	}, crlf)
	expectedHeaders := http.Header{
		"Age":               {"2"},
		"Cache-Control":     {"public, max-age=86400, s-maxage=86400"},
		"Content-Type":      {"application/json; charset=utf-8"},
		"Date":              {"Sat, 03 Dec 2022 23:25:26 GMT"},
		"Server":            {"Ian's Proxy"},
		"X-Cache":           {"HIT"},
		"Transfer-Encoding": {"chunked"},
	}
	builder := &cacheResponseBuilder{
		response: &http_.Response{
			Response: &http.Response{},
		},
		reader: &cacheEntryReader{
			Reader: bufio.NewReader(strings.NewReader(entry)),
		},
	}
	assert.Nil(t, builder.setCachedHeaders())
	assert.EqualValues(t, expectedHeaders, builder.response.Header)
}

func TestSetCachedHeadersError(t *testing.T) {
	for _, entry := range []string{
		"",
		"Invalid header\r\nAge: 2\r\n\r\nResponse Body",
		"Age: 2\r\nCache-Control: public\r\n",
		"Age: 2\r\nCache-Control: public",
	} {
		testName := fmt.Sprintf("setCachedHeaders(), entry=%s", entry)
		t.Run(testName, func(t *testing.T) {
			builder := &cacheResponseBuilder{
				response: &http_.Response{
					Response: &http.Response{},
				},
				reader: &cacheEntryReader{
					Reader: bufio.NewReader(strings.NewReader(entry)),
				},
			}
			var err error
			assert.NotEmpty(t, captureLog(func() { err = builder.setCachedHeaders() }))
			assert.NotNil(t, err)
		})
	}
}

func TestSetAgeHeaderSuccess(t *testing.T) {
	tests := []struct {
		headers     http.Header
		expectedAge string
	}{
		{
			headers:     http.Header{"Date": {"Mon, 02 Jan 2006 15:04:05 UTC"}},
			expectedAge: "0",
		},
		{
			headers:     http.Header{"Date": {"Mon, 02 Jan 2006 15:04:35 UTC"}},
			expectedAge: "-30",
		},
		{
			headers:     http.Header{"Date": {"Mon, 02 Jan 2006 15:03:35 UTC"}},
			expectedAge: "30",
		},
		{
			headers: http.Header{
				"Date": {"Mon, 02 Jan 2006 15:03:35 UTC"},
				"Age":  {"foobar"},
			},
			expectedAge: "30",
		},
		{
			headers:     http.Header{"Date": {"Monday, 02-Jan-06 15:03:35 UTC"}},
			expectedAge: "30",
		},
		{
			headers:     http.Header{"Date": {"Mon Jan  2 15:03:35 2006"}},
			expectedAge: "30",
		},
	}
	now, _ := time.Parse(time.RFC1123, "Mon, 02 Jan 2006 15:04:05 UTC")
	timeSince = func(t time.Time) time.Duration {
		return now.Sub(t)
	}
	for _, test := range tests {
		testName := fmt.Sprintf("setAgeHeader(), h=%v", test.headers)
		t.Run(testName, func(t *testing.T) {
			builder := &cacheResponseBuilder{
				response: &http_.Response{
					Response: &http.Response{
						Header: test.headers,
					},
				},
			}
			assert.Nil(t, builder.setAgeHeader())
			assert.EqualValues(t, []string{test.expectedAge}, builder.response.Header["Age"])
		})
	}
}

func TestSetAgeHeaderError(t *testing.T) {
	for _, headers := range []http.Header{
		{},
		{"Foo": {"Bar"}},
		{"date": {"Mon, 02 Jan 2006 15:03:35 UTC"}},
		{"Date": {"Mon, 02 Jan 2006 15:03:35 UTC", "Mon, 02 Jan 2006 15:03:35 UTC"}},
	} {
		testName := fmt.Sprintf("overwriteAgeHeader(h=%v)", headers)
		t.Run(testName, func(t *testing.T) {
			builder := &cacheResponseBuilder{response: &http_.Response{Response: &http.Response{}}}
			var err error
			assert.NotEmpty(t, captureLog(func() { err = builder.setAgeHeader() }))
			assert.NotNil(t, err)
		})
	}
}

func TestWithError(t *testing.T) {
	err := errors.New("error")
	builder := &cacheResponseBuilder{}
	assert.Same(t, builder, builder.withError(err))
	assert.Same(t, err, builder.err)
}

func TestGetLineSuccess(t *testing.T) {
	tests := []struct {
		reader   stringReader
		expected string
	}{
		{
			reader:   bytes.NewBufferString("\n"),
			expected: "\n",
		},
		{
			reader:   bytes.NewBufferString("\n\n"),
			expected: "\n",
		},
		{
			reader:   bytes.NewBufferString("\r\n"),
			expected: "\r\n",
		},
		{
			reader:   bytes.NewBufferString("Hello World\n"),
			expected: "Hello World\n",
		},
		{
			reader:   bytes.NewBufferString("Hello World\n\n"),
			expected: "Hello World\n",
		},
		{
			reader:   bytes.NewBufferString("Hello\nWorld"),
			expected: "Hello\n",
		},
	}
	for _, test := range tests {
		testName := fmt.Sprintf("getLine(r=%v)", test.reader)
		t.Run(testName, func(t *testing.T) {
			line, err := getLine(test.reader)
			assert.EqualValues(t, test.expected, line)
			assert.Nil(t, err)
		})
	}
}

func TestGetLineError(t *testing.T) {
	for _, reader := range []stringReader{
		bytes.NewBufferString(""),
		bytes.NewBufferString("No new line character"),
	} {
		testName := fmt.Sprintf("getLine(r=%v)", reader)
		t.Run(testName, func(t *testing.T) {
			var err error
			assert.NotEmpty(t, captureLog(func() { _, err = getLine(reader) }))
			assert.NotNil(t, err)
		})
	}
}

func TestGetStatusCodeSuccess(t *testing.T) {
	tests := []struct {
		line     string
		expected int
	}{
		{
			line:     "333",
			expected: 333,
		},
		{
			line:     "22   333   ",
			expected: 333,
		},
		{
			line:     "HTTP/1.1 200 OK",
			expected: 200,
		},
		{
			line:     "HTTP/1.1 404 OK",
			expected: 404,
		},
	}
	for _, test := range tests {
		testName := fmt.Sprintf("getStatusCode(line=%s", test.line)
		t.Run(testName, func(t *testing.T) {
			statusCode, err := getStatusCode(test.line)
			assert.EqualValues(t, test.expected, statusCode)
			assert.Nil(t, err)
		})
	}
}

func TestGetStatusCodeError(t *testing.T) {
	for _, line := range []string{
		"",
		"0",
		"20",
		"40",
		"99",
		"20 0",
		"http/1.1 20 OK",
		"http/1.1 20 0 OK",
	} {
		testName := fmt.Sprintf("getStatusCode(line=%s", line)
		t.Run(testName, func(t *testing.T) {
			var err error
			assert.NotEmpty(t, captureLog(func() { _, err = getStatusCode(line) }))
			assert.NotNil(t, err)
		})
	}
}

func TestSetHeaderSuccess(t *testing.T) {
	tests := []struct {
		line            string
		headers         http.Header
		expectedHeaders http.Header
	}{
		{
			line:            "Foo: Bar",
			headers:         http.Header{},
			expectedHeaders: http.Header{"Foo": {"Bar"}},
		},
		{
			line:            "foo_Foo: Bar",
			headers:         http.Header{},
			expectedHeaders: http.Header{"Foo_foo": {"Bar"}},
		},
		{
			line:            "Foo: Bar:Baz",
			headers:         http.Header{},
			expectedHeaders: http.Header{"Foo": {"Bar:Baz"}},
		},
		{
			line:            "Foo:Bar",
			headers:         http.Header{},
			expectedHeaders: http.Header{"Foo": {"Bar"}},
		},
		{
			line:            "        Foo     :      Bar       ",
			headers:         http.Header{},
			expectedHeaders: http.Header{"Foo": {"Bar"}},
		},
		{
			line:            "        Foo     :      Bar  ,  Baz     ",
			headers:         http.Header{},
			expectedHeaders: http.Header{"Foo": {"Bar  ,  Baz"}},
		},
		{
			line:            "Foo: Bar",
			headers:         http.Header{"Foo": {"Baz"}},
			expectedHeaders: http.Header{"Foo": {"Baz", "Bar"}},
		},
	}
	for _, test := range tests {
		testName := fmt.Sprintf("setHeader(line=%s)", test.line)
		t.Run(testName, func(t *testing.T) {
			assert.Nil(t, setHeader(test.headers, test.line))
			assert.EqualValues(t, test.expectedHeaders, test.headers)
		})
	}
}

func TestSetHeaderError(t *testing.T) {
	headers := http.Header{}
	for _, line := range []string{
		"",
		": Bar",
		"Foo Bar:      Baz",
		"Foo     :      ",
		"Foo&Foo: Bar",
		"$Foo: Bar",
	} {
		testName := fmt.Sprintf("setHeader(line=%s)", line)
		t.Run(testName, func(t *testing.T) {
			assert.NotNil(t, setHeader(headers, line))
		})
	}
}

func TestAddOverwriteHeaderSuccess(t *testing.T) {
	tests := []struct {
		headers     http.Header
		expectedAge string
	}{
		{
			headers:     http.Header{"Date": {"Mon, 02 Jan 2006 15:04:05 UTC"}},
			expectedAge: "0",
		},
		{
			headers:     http.Header{"Date": {"Mon, 02 Jan 2006 15:04:35 UTC"}},
			expectedAge: "-30",
		},
		{
			headers:     http.Header{"Date": {"Mon, 02 Jan 2006 15:03:35 UTC"}},
			expectedAge: "30",
		},
		{
			headers: http.Header{
				"Date": {"Mon, 02 Jan 2006 15:03:35 UTC"},
				"Age":  {"foobar"},
			},
			expectedAge: "30",
		},
		{
			headers:     http.Header{"Date": {"Monday, 02-Jan-06 15:03:35 UTC"}},
			expectedAge: "30",
		},
		{
			headers:     http.Header{"Date": {"Mon Jan  2 15:03:35 2006"}},
			expectedAge: "30",
		},
	}
	now, _ := time.Parse(time.RFC1123, "Mon, 02 Jan 2006 15:04:05 UTC")
	timeSince = func(t time.Time) time.Duration {
		return now.Sub(t)
	}
	for _, test := range tests {
		testName := fmt.Sprintf("overwriteAgeHeader(h=%v)", test.headers)
		t.Run(testName, func(t *testing.T) {
			assert.Nil(t, overwriteAgeHeader(test.headers))
			assert.EqualValues(t, []string{test.expectedAge}, test.headers["Age"])
		})
	}
}

func TestOverwriteAgeHeaderError(t *testing.T) {
	for _, headers := range []http.Header{
		{},
		{"Foo": {"Bar"}},
		{"date": {"Mon, 02 Jan 2006 15:03:35 UTC"}},
		{"Date": {"Mon, 02 Jan 2006 15:03:35 UTC", "Mon, 02 Jan 2006 15:03:35 UTC"}},
	} {
		testName := fmt.Sprintf("overwriteAgeHeader(h=%v)", headers)
		t.Run(testName, func(t *testing.T) {
			assert.NotNil(t, overwriteAgeHeader(headers))
		})
	}
}

func TestGetEntryAge(t *testing.T) {
	tests := []struct {
		timestamp string
		expected  string
	}{
		{
			timestamp: "",
			expected:  "0",
		},
		{
			timestamp: "invalid ts 19 avril 2023, 13 h 42",
			expected:  "0",
		},
		{
			timestamp: "Mon, 02 Jan 2006 15:04:05 UTC",
			expected:  "0",
		},
		{
			timestamp: "Mon, 02 Jan 2006 15:03:35 UTC",
			expected:  "30",
		},
		{
			timestamp: "Mon, 02 Jan 2006 15:04:35 UTC",
			expected:  "-30",
		},
		{
			timestamp: "Monday, 02-Jan-06 15:03:35 UTC",
			expected:  "30",
		},
		{
			timestamp: "Mon Jan  2 15:03:35 2006",
			expected:  "30",
		},
	}
	now, _ := time.Parse(time.RFC1123, "Mon, 02 Jan 2006 15:04:05 UTC")
	timeSince = func(t time.Time) time.Duration {
		return now.Sub(t)
	}
	for _, test := range tests {
		testName := fmt.Sprintf("getEntryAge(ts=%s)", test.timestamp)
		t.Run(testName, func(t *testing.T) {
			assert.EqualValues(t, test.expected, getEntryAge(test.timestamp))
		})
	}
}
