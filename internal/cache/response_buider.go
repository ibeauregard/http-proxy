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

var headerMatchingRegexp = regexp.MustCompile(`([-\w]+)\s*:\s*(.*\S)`)

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
		headerParts := headerMatchingRegexp.FindStringSubmatch(line)
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
			b.error = true
			return b
		}
	}
	err = addAgeHeader(b.response.Header)
	if err != nil {
		errors.Log(b.setHeaders, err)
		b.error = true
	}
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

var statusCodeRegexp = regexp.MustCompile(`\b\d{3}\b`)

func getStatusCode(firstLine []byte) (int, error) {
	decimalStatusCode := statusCodeRegexp.Find(firstLine)
	statusCode, err := strconv.Atoi(string(decimalStatusCode))
	if err != nil {
		errors.Log(getStatusCode, errors.New(
			"first line of the cache entry does not contain a valid HTTP response status code"))
		return 0, err
	}
	return statusCode, nil
}

func addAgeHeader(headers http.Header) error {
	dates, ok := headers[http.CanonicalHeaderKey("Date")]
	if !ok {
		return errors.Format(addAgeHeader, errors.New("Date header missing from cache entry"))
	}
	if len(dates) > 1 {
		return errors.Format(addAgeHeader, errors.New("multiple Date headers in cache entry"))
	}
	canonicalAgeKey := http.CanonicalHeaderKey("Age")
	headers[canonicalAgeKey] = append(headers[canonicalAgeKey], getEntryAge(dates[0]))
	return nil
}

func getEntryAge(dateTimestamp string) string {
	age := GetDurationSinceTimestamp(dateTimestamp)
	return strconv.Itoa(int(age.Seconds()))
}
