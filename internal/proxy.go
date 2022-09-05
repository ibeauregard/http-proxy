package main

import (
	"io"
	"my_proxy/internal/cache"
	"my_proxy/internal/errors"
	"net/http"
	"strconv"
)

func myProxy(writer http.ResponseWriter, request *http.Request) {
	if !validateRequestMethod(writer, request.Method) {
		return
	}
	requestUrl := request.URL.Query().Get("request")
	cacheKey := cache.GetKey(requestUrl)
	if serveFromCache(writer, cacheKey) {
		return
	}
	serveFromUpstream(writer, requestUrl, cacheKey)
}

func validateRequestMethod(writer http.ResponseWriter, requestMethod string) bool {
	// Request method names are case-sensitive
	// See https://www.rfc-editor.org/rfc/rfc7230#section-3.1.1
	if requestMethod != "GET" {
		http.Error(writer, "Invalid request method; use GET", http.StatusMethodNotAllowed)
		return false
	}
	return true
}

// have the file fetch return http.Response? most likely
func serveFromCache(writer http.ResponseWriter, cacheKey string) bool {
	return false
}

func serveFromUpstream(writer http.ResponseWriter, requestUrl string, cacheKey string) {
	resp, err := http.Get(requestUrl)
	if err != nil {
		return
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	filteredHeaders := getFilteredHeaders(resp.Header, bodyBytes)

	go cache.CacheResponse(resp.Proto, resp.Status, filteredHeaders, bodyBytes, cacheKey)

	writeHeaders(filteredHeaders, writer, resp.StatusCode)

	_, err = writer.Write(bodyBytes)
	if err != nil {
		errors.Log(serveFromUpstream, err)
	}
}

func writeHeaders(headers http.Header, writer http.ResponseWriter, statusCode int) {
	for name, values := range headers {
		writer.Header()[name] = values
	}
	writer.Header()["X-Cache"] = []string{"MISS"}
	writer.WriteHeader(statusCode)
}

var getFilteredHeaders = func() func(http.Header, []byte) http.Header {
	copiedHeaders := map[string]struct{}{
		"Content-Type":  {},
		"Cache-Control": {},
		"Date":          {},
		"Expires":       {},
		"Set-Cookie":    {},
	}
	return func(responseHeaders http.Header, bodyBytes []byte) http.Header {
		filteredHeaders := make(http.Header)
		for name, values := range responseHeaders {
			canonicalHeaderKey := http.CanonicalHeaderKey(name)
			if _, ok := copiedHeaders[canonicalHeaderKey]; ok {
				filteredHeaders[canonicalHeaderKey] = values
			}
		}
		filteredHeaders["Content-Length"] = []string{strconv.Itoa(len(bodyBytes))}
		filteredHeaders["Server"] = []string{"Ian's Proxy"}
		return filteredHeaders
	}
}()
