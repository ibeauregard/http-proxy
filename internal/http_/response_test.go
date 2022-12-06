package http_

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"my_proxy/internal/tests"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var ioCopyBackup = ioCopy

type bodyMock struct {
	io.Reader
	closed bool
}

func (b *bodyMock) Close() error {
	b.closed = true
	return nil
}

func TestNewResponse(t *testing.T) {
	statusCode := 402
	headers := http.Header{
		"Foo":           {""},
		"Bar":           {"", ""},
		"Content-Type":  {"1"},
		"Cache-Control": {"2"},
		"Date":          {"3"},
		"Expires":       {"4"},
		"Set-Cookie":    {"5"},
	}
	filteredHeaders := http.Header{
		"Content-Type":  {"1"},
		"Cache-Control": {"2"},
		"Date":          {"3"},
		"Expires":       {"4"},
		"Set-Cookie":    {"5"},
		"Server":        {"Ian's Proxy"},
	}
	bodyContent := "my body content"
	body := io.NopCloser(strings.NewReader(bodyContent))
	resp := &http.Response{
		StatusCode: statusCode,
		Header:     headers,
		Body:       body,
	}
	output := NewResponse(resp)
	assert.Equal(t, statusCode, output.StatusCode)
	assert.Equal(t, filteredHeaders, output.Header)
	writer := &strings.Builder{}
	_, _ = io.Copy(writer, output)
	assert.Equal(t, bodyContent, writer.String())
}

func TestServeSuccess(t *testing.T) {
	bodyContent := "my response body"
	body := &bodyMock{Reader: strings.NewReader(bodyContent)}
	statusCode := 303
	headers := http.Header{"key1": {"value1"}, "key2": {"value2"}}
	resp := &Response{
		Response: &http.Response{
			StatusCode: statusCode,
			Header:     headers,
		},
		Body: &Body{body},
	}
	writer := httptest.NewRecorder()
	assert.Empty(t, tests.CaptureLog(func() { resp.Serve(writer) }))
	assert.Equal(t, statusCode, writer.Code)
	assert.Equal(t, headers, writer.Header())
	assert.Equal(t, bodyContent, writer.Body.String())
	assert.True(t, body.closed)
}

func TestServeBodyCopyFailure(t *testing.T) {
	body := &bodyMock{}
	resp := &Response{
		Response: &http.Response{
			StatusCode: 200,
		},
		Body: &Body{body},
	}
	ioCopy = func(_ io.Writer, _ io.Reader) (int64, error) {
		return 0, errors.New("error")
	}
	defer func() { ioCopy = ioCopyBackup }()
	assert.NotEmpty(t, tests.CaptureLog(func() { resp.Serve(httptest.NewRecorder()) }))
	assert.True(t, body.closed)
}

func TestWithBody(t *testing.T) {
	content := "my content"
	writer := &strings.Builder{}
	resp := &Response{}

	t.Run("body not an io.Closer", func(t *testing.T) {
		_, _ = io.Copy(writer, resp.WithBody(strings.NewReader(content)))
		assert.Equal(t, content, writer.String())
	})

	t.Run("body already an io.ReadCloser", func(t *testing.T) {
		writer.Reset()
		_, _ = io.Copy(writer, resp.WithBody(io.NopCloser(strings.NewReader(content))))
		assert.Equal(t, content, writer.String())
	})
}

type readCloserMock struct {
	io.Reader
	closeError error
	closed     bool
}

func (m *readCloserMock) Close() error {
	if m.closeError != nil {
		return m.closeError
	}
	m.closed = true
	return nil
}

func TestCloseSuccess(t *testing.T) {
	mock := &readCloserMock{}
	assert.Empty(t, tests.CaptureLog(func() {
		(&Body{mock}).Close()
	}))
	assert.True(t, mock.closed)
}

func TestCloseError(t *testing.T) {
	assert.NotEmpty(t, tests.CaptureLog(func() {
		(&Body{
			&readCloserMock{
				closeError: errors.New("error")}}).Close()
	}))
}

func TestWriteHeaders(t *testing.T) {
	headers := http.Header{
		"key1": {"value1"},
		"key2": {"value2"},
	}
	writer := httptest.NewRecorder()
	writeHeaders(writer, headers)
	assert.Equal(t, headers, writer.Header())
}

func TestGetFilteredHeaders(t *testing.T) {
	for _, test := range []struct {
		input          http.Header
		expectedOutput http.Header
	}{
		{
			input:          http.Header{},
			expectedOutput: http.Header{},
		},
		{
			input: http.Header{
				"cONTENT-tYPE":  {"1"},
				"cACHE-cONTROL": {"2"},
				"DATE":          {"3"},
				"eXPIRES":       {"4"},
				"sET-cOOKIE":    {"5"},
			},
			expectedOutput: http.Header{
				"Content-Type":  {"1"},
				"Cache-Control": {"2"},
				"Date":          {"3"},
				"Expires":       {"4"},
				"Set-Cookie":    {"5"},
			},
		},
		{
			input: http.Header{
				"Foo":           {""},
				"Bar":           {"", ""},
				"Content-Type":  {"1"},
				"Cache-Control": {"2"},
				"Date":          {"3"},
				"Expires":       {"4"},
				"Set-Cookie":    {"5"},
			},
			expectedOutput: http.Header{
				"Content-Type":  {"1"},
				"Cache-Control": {"2"},
				"Date":          {"3"},
				"Expires":       {"4"},
				"Set-Cookie":    {"5"},
			},
		},
	} {
		testName := fmt.Sprintf("getFilteredHeaders(h=%v)", test.input)
		t.Run(testName, func(t *testing.T) {
			test.expectedOutput["Server"] = []string{"Ian's Proxy"}
			assert.Equal(t, test.expectedOutput, getFilteredHeaders(test.input))
		})
	}
}
