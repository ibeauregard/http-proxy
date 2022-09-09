package cache

import (
	"bufio"
	"fmt"
	"io"
	"my_proxy/internal/errors"
	h "my_proxy/internal/http"
	"net/http"
	"regexp"
	"strconv"
)

type CacheableResponse struct {
	*h.Response
}

func (r *CacheableResponse) Store(cacheKey string) {
	cacheLifespan := getCacheLifespan(r.Header)
	if cacheLifespan == 0 {
		return
	}
	cacheFile := newCacheFile(cacheKey)
	openCacheFile, err := cacheFile.create()
	if err != nil {
		errors.Log(r.Store, err)
		return
	}
	defer openCacheFile.close()
	if err = r.writeToCache(openCacheFile); err != nil {
		errors.Log(r.Store, err)
		cacheFile.delete()
		return
	}
	cacheFile.scheduleDeletion(cacheLifespan)
}

// Retrieve TODO: might need to acquire locks here
func Retrieve(cacheKey string) *h.Response {
	cacheFile := newCacheFile(cacheKey)
	openCacheFile, err := cacheFile.open()
	if err != nil {
		return nil
	}
	return newCacheResponseBuilder(openCacheFile).
		setStatusCode().
		setHeaders().
		setBody().
		build()
}

func (r *CacheableResponse) writeToCache(f io.Writer) error {
	w := cacheEntryWriter{bufio.NewWriter(f)}
	if err := w.writeStatusLine(r.Proto, r.StatusCode); err != nil {
		return errors.Format(r.writeToCache, err)
	}
	if err := w.writeHeaders(r.Header); err != nil {
		return errors.Format(r.writeToCache, err)
	}
	if err := w.writeBody(r.Body); err != nil {
		return errors.Format(r.writeToCache, err)
	}
	if err := w.Flush(); err != nil {
		return errors.Format(r.writeToCache, err)
	}
	return nil
}

type cacheEntryWriter struct {
	*bufio.Writer
}

var crlf = "\r\n"

func (w *cacheEntryWriter) writeStatusLine(proto string, statusCode int) error {
	if _, err := w.WriteString(
		fmt.Sprintf("%s %d %s %s", proto, statusCode, http.StatusText(statusCode), crlf)); err != nil {
		return errors.Format(w.writeStatusLine, err)
	}
	return nil
}

func (w *cacheEntryWriter) writeHeaders(headers http.Header) error {
	colonSpace := ": "
	for headerKey, headerValues := range headers {
		for _, headerValue := range headerValues {
			if _, err := w.WriteString(fmt.Sprint(headerKey, colonSpace, headerValue, crlf)); err != nil {
				return errors.Format(w.writeHeaders, err)
			}
		}
	}
	if _, err := w.WriteString(fmt.Sprint("X-Cache", colonSpace, "HIT", crlf)); err != nil {
		return errors.Format(w.writeHeaders, err)
	}
	if _, err := w.WriteString(crlf); err != nil {
		return errors.Format(w.writeHeaders, err)
	}
	return nil
}

func (w *cacheEntryWriter) writeBody(body io.Reader) error {
	if _, err := io.Copy(w, body); err != nil {
		return errors.Format(w.writeBody, err)
	}
	return nil
}

type cacheResponseBuilder struct {
	response *h.Response
	reader   *bufio.Reader
	error    bool
}

func newCacheResponseBuilder(reader io.Reader) *cacheResponseBuilder {
	return &cacheResponseBuilder{response: &h.Response{
		Response: &http.Response{},
	}, reader: bufio.NewReader(reader)}
}

// TODO: refactor
func (b *cacheResponseBuilder) setStatusCode() *cacheResponseBuilder {
	firstLine, err := b.reader.ReadBytes('\n')
	if err != nil {
		errors.Log(b.setStatusCode, errors.New("unexpected end of cache entry"))
		b.error = true
		return b
	}
	decimalStatusCode := regexp.MustCompile(`\b\d{3}\b`).Find(firstLine)
	statusCode, err := strconv.Atoi(string(decimalStatusCode))
	if err != nil {
		errors.Log(b.setStatusCode, errors.New(
			"first line of the cache entry does not contain a valid HTTP response status code"))
		b.error = true
		return b
	}
	b.response.StatusCode = statusCode
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

func getEntryAge(dateTimestamp string) string {
	age := GetDurationSinceTimestamp(dateTimestamp)
	return strconv.Itoa(int(age.Seconds()))
}

func (b *cacheResponseBuilder) setBody() *cacheResponseBuilder {
	if b.error {
		return b
	}
	b.response.Body = &h.Body{ReadCloser: io.NopCloser(b.reader)}
	return b
}

func (b *cacheResponseBuilder) build() *h.Response {
	if b.error {
		return nil
	}
	return b.response
}
