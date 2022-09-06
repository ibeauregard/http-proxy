package main

import (
	"io"
	"my_proxy/internal/cache"
	h "my_proxy/internal/http"
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
		response = getResponseFromUpstream(requestUrl)
		writer.Header()["X-Cache"] = []string{"MISS"}
	}
	response.Serve(writer)
	go store(response, cacheKey)
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

func getResponseFromCache(_ string) *h.Response {
	return nil
}

func getResponseFromUpstream(requestUrl string) *h.Response {
	resp, err := http.Get(requestUrl)
	if err != nil {
		return nil
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	filteredHeaders := getFilteredHeaders(resp.Header, bodyBytes)

	return &h.Response{Proto: resp.Proto, StatusCode: resp.StatusCode, Headers: filteredHeaders, Body: bodyBytes}
}

func store(r *h.Response, cacheKey string) {
	cr := &cache.CacheableResponse{Response: r}
	cr.Store(cacheKey)
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
