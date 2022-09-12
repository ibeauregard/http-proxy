package cache

import (
	"bufio"
	"io"
	"my_proxy/internal/errors"
	"my_proxy/internal/http_"
	"net/http"
	"regexp"
	"strconv"
)

type cacheResponseBuilder struct {
	response *http_.Response
	reader   *cacheEntryReader
	error    bool
}

func newCacheResponseBuilder(readCloser io.ReadCloser) *cacheResponseBuilder {
	return &cacheResponseBuilder{response: &http_.Response{
		Response: &http.Response{},
	}, reader: &cacheEntryReader{
		Reader: bufio.NewReader(readCloser),
		Closer: readCloser,
	}}
}

func (b *cacheResponseBuilder) setStatusCode() *cacheResponseBuilder {
	firstLine, err := getFirstLine(b.reader)
	if err != nil {
		b.error = true
		return b
	}
	b.response.StatusCode, err = getStatusCode(firstLine)
	b.error = err != nil
	return b
}

// TODO: refactor
func (b *cacheResponseBuilder) setHeaders() *cacheResponseBuilder {
	if b.error {
		return b
	}
	line, err := b.reader.ReadString('\n')
	if err != nil {
		errors.Log(b.setHeaders, err)
		b.error = true
		return b
	}
	b.response.Header = make(map[string][]string)
	for line != "\r\n" {
		headerParts := regexp.MustCompile(`([-\w]+)\s*:\s*(.*\S)`).FindStringSubmatch(line)
		if headerParts == nil {
			errors.Log(b.setHeaders, errors.New("malformed header in cache entry"))
			b.error = true
			return b
		}
		key, value := headerParts[1], headerParts[2]
		canonicalKey := http.CanonicalHeaderKey(key)
		b.response.Header[canonicalKey] = append(b.response.Header[canonicalKey], value)
		line, err = b.reader.ReadString('\n')
		if err != nil {
			errors.Log(b.setHeaders, err)
		}
	}
	dates, ok := b.response.Header[http.CanonicalHeaderKey("Date")]
	if !ok {
		errors.Log(b.setHeaders, errors.New("Date header missing from cache entry"))
		return b
	}
	if len(dates) > 1 {
		errors.Log(b.setHeaders, errors.New("multiple Date headers in cache entry"))
	}
	canonicalAgeKey := http.CanonicalHeaderKey("Age")
	b.response.Header[canonicalAgeKey] = append(b.response.Header[canonicalAgeKey], getEntryAge(dates[0]))
	return b
}

func (b *cacheResponseBuilder) setBody() *cacheResponseBuilder {
	if b.error {
		return b
	}
	b.response.Body = &http_.Body{ReadCloser: b.reader}
	return b
}

func (b *cacheResponseBuilder) build() *http_.Response {
	if b.error {
		return nil
	}
	return b.response
}

type byteSliceReader interface {
	ReadBytes(delim byte) ([]byte, error)
}

func getFirstLine(reader byteSliceReader) ([]byte, error) {
	firstLine, err := reader.ReadBytes('\n')
	if err != nil {
		errors.Log(getFirstLine, errors.New("unexpected end of cache entry"))
		return nil, err
	}
	return firstLine, nil
}

func getStatusCode(firstLine []byte) (int, error) {
	decimalStatusCode := regexp.MustCompile(`\b\d{3}\b`).Find(firstLine)
	statusCode, err := strconv.Atoi(string(decimalStatusCode))
	if err != nil {
		errors.Log(getStatusCode, errors.New(
			"first line of the cache entry does not contain a valid HTTP response status code"))
		return 0, err
	}
	return statusCode, nil
}

func getEntryAge(dateTimestamp string) string {
	age := GetDurationSinceTimestamp(dateTimestamp)
	return strconv.Itoa(int(age.Seconds()))
}
