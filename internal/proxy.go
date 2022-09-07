package main

import (
	"bytes"
	"io"
	"my_proxy/internal/cache"
	"my_proxy/internal/errors"
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
	r, err := http.Get(requestUrl)
	if err != nil {
		errors.Log(serveFromUpstream, err)
		return
	}
	resp := h.NewResponse(r)
	defer resp.Body.Close()

	writer.Header()["X-Cache"] = []string{"MISS"}

	newBody := &bytes.Buffer{}
	resp.WithNewBody(io.TeeReader(r.Body, newBody)).Serve(writer)
	go store(resp.WithNewBody(newBody), cacheKey)
}

func store(r *h.Response, cacheKey string) {
	defer r.Body.Close()
	cr := &cache.CacheableResponse{Response: r}
	cr.Store(cacheKey)
}
