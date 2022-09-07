package main

import (
	"bytes"
	"io"
	"my_proxy/internal/cache"
	h "my_proxy/internal/http"
	"net/http"
)

func myProxy(writer http.ResponseWriter, request *http.Request) {
	if !validateRequestMethod(writer, request.Method) {
		return
	}
	requestUrl := request.URL.Query().Get("request")
	cacheKey := cache.GetKey(requestUrl)
	if !serveFromCache(writer, cacheKey) {
		serveFromUpstream(writer, requestUrl, cacheKey)
	}
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

func serveFromCache(_ http.ResponseWriter, _ string) bool {
	return false
}

func serveFromUpstream(writer http.ResponseWriter, requestUrl, cacheKey string) {
	resp, err := http.Get(requestUrl)
	if err != nil {
		return
	}
	defer func() { _ = resp.Body.Close() }()

	writer.Header()["X-Cache"] = []string{"MISS"}

	buf := &bytes.Buffer{}
	response := h.NewResponse(resp)
	response.WithNewBody(io.TeeReader(resp.Body, buf)).Serve(writer)
	go store(response.WithNewBody(buf), cacheKey)
}

func store(r *h.Response, cacheKey string) {
	cr := &cache.CacheableResponse{Response: r}
	cr.Store(cacheKey)
}
