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
	err      error
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
	firstLine, err := getLine(b.reader)
	if err != nil {
		return b.withError(err)
	}
	b.response.StatusCode, err = getStatusCode(firstLine)
	return b.withError(err)
}

var headerMatchingRegexp = regexp.MustCompile(`([-\w]+)\s*:\s*(.*\S)`)

func (b *cacheResponseBuilder) setHeaders() *cacheResponseBuilder {
	if b.err != nil {
		return b
	}
	if err := b.setCachedHeaders(); err != nil {
		return b.withError(err)
	}
	return b.withError(b.setAgeHeader())
}

func (b *cacheResponseBuilder) setBody() *cacheResponseBuilder {
	if b.err != nil {
		return b
	}
	b.response.Body = &http_.Body{ReadCloser: b.reader}
	return b
}

func (b *cacheResponseBuilder) build() *http_.Response {
	if b.err != nil {
		return nil
	}
	return b.response
}

func (b *cacheResponseBuilder) setCachedHeaders() error {
	line, err := getLine(b.reader)
	if err != nil {
		return err
	}
	b.response.Header = make(map[string][]string)
	for line != crlf {
		if err = setHeader(b.response.Header, line); err != nil {
			errors.Log(b.setHeaders, err)
			return err
		}
		if line, err = getLine(b.reader); err != nil {
			return err
		}
	}
	return nil
}

func (b *cacheResponseBuilder) setAgeHeader() error {
	if err := addAgeHeader(b.response.Header); err != nil {
		errors.Log(b.setHeaders, err)
		return err
	}
	return nil
}

func (b *cacheResponseBuilder) withError(err error) *cacheResponseBuilder {
	b.err = err
	return b
}

type stringReader interface {
	ReadString(byte) (string, error)
}

func getLine(reader stringReader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		errors.Log(getLine, errors.New("unexpected end of cache entry"))
		return "", err
	}
	return line, nil
}

var statusCodeRegexp = regexp.MustCompile(`\b\d{3}\b`)

func getStatusCode(firstLine string) (int, error) {
	decimalStatusCode := statusCodeRegexp.FindString(firstLine)
	statusCode, err := strconv.Atoi(decimalStatusCode)
	if err != nil {
		errors.Log(getStatusCode, errors.New(
			"first line of the cache entry does not contain a valid HTTP response status code"))
		return 0, err
	}
	return statusCode, nil
}

func setHeader(headers http.Header, line string) error {
	headerParts := headerMatchingRegexp.FindStringSubmatch(line)
	if headerParts == nil {
		return errors.Format(setHeader, errors.New("malformed header in cache entry"))
	}
	key, value := headerParts[1], headerParts[2]
	canonicalKey := http.CanonicalHeaderKey(key)
	headers[canonicalKey] = append(headers[canonicalKey], value)
	return nil
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
