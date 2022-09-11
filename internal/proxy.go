package main

import (
	"io"
	"my_proxy/internal/cache"
	"my_proxy/internal/errors"
	"my_proxy/internal/http_"
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

func serveFromCache(writer http.ResponseWriter, cacheKey string) bool {
	resp := cache.Retrieve(cacheKey)
	if resp == nil {
		return false
	}
	resp.Serve(writer)
	return true
}

func serveFromUpstream(writer http.ResponseWriter, requestUrl, cacheKey string) {
	r, err := http.Get(requestUrl)
	if err != nil {
		errors.Log(serveFromUpstream, err)
		return
	}
	resp := http_.NewResponse(r)
	defer resp.Body.Close()

	writer.Header()["X-Cache"] = []string{"MISS"}

	newBodyReader, newBodyWriter := io.Pipe()
	defer func() { _ = newBodyWriter.Close() }()
	go store(resp.WithNewBody(newBodyReader), cacheKey)
	resp.WithNewBody(io.NopCloser(io.TeeReader(r.Body, newBodyWriter))).Serve(writer)
}

func store(r *http_.Response, cacheKey string) {
	cr := &cache.CacheableResponse{Response: r}
	cr.Store(cacheKey)
}
