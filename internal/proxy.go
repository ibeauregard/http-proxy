package main

import (
	"io"
	"my_proxy/internal/cache"
	"my_proxy/internal/errors"
	r "my_proxy/internal/response"
	"net/http"
	"strconv"
)

func myProxy(writer http.ResponseWriter, request *http.Request) {
	if !validateRequestMethod(writer, request.Method) {
		return
	}
	requestUrl := request.URL.Query().Get("request")
	cacheKey := cache.GetKey(requestUrl)
	response := getResponseFromCache(cacheKey)
	if response == nil {
		response = getResponseFromUpstream(requestUrl, cacheKey)
		writer.Header()["X-Cache"] = []string{"MISS"}
	}
	serve(writer, response)
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
func getResponseFromCache(cacheKey string) *r.Response {
	return nil
}

func getResponseFromUpstream(requestUrl string, cacheKey string) *r.Response {
	resp, err := http.Get(requestUrl)
	if err != nil {
		return nil
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	filteredHeaders := getFilteredHeaders(resp.Header, bodyBytes)

	response := r.NewResponse(resp.StatusCode, filteredHeaders, bodyBytes)
	go cache.Store(cache.NewCacheableResponse(resp.Proto, response), cacheKey)
	return response
}

type response interface {
	GetStatusCode() int
	GetHeaders() http.Header
	GetBody() []byte
}

func serve(writer http.ResponseWriter, r response) {
	writeHeaders(writer, r.GetHeaders())
	writer.WriteHeader(r.GetStatusCode())

	_, err := writer.Write(r.GetBody())
	if err != nil {
		errors.Log(getResponseFromUpstream, err)
	}
}

func writeHeaders(writer http.ResponseWriter, headers http.Header) {
	for name, values := range headers {
		writer.Header()[name] = values
	}
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
