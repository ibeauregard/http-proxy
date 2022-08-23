package main

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"log"
	"net/http"
)

func main() {
	log.Fatal(http.ListenAndServe(":8080", http.HandlerFunc(myProxy)))
}

func myProxy(writer http.ResponseWriter, request *http.Request) {
	if !validateRequestMethod(writer, request.Method) {
		return
	}
	requestUrl := request.URL.Query().Get("request")
	cacheKey := getCacheKey(requestUrl)
	if serveFromCache(writer, cacheKey) {
		return
	}
	serveFromUpstream(writer, requestUrl)
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
	return false
}

func serveFromUpstream(writer http.ResponseWriter, requestUrl string) {
	resp, err := http.Get(requestUrl)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	for name, values := range resp.Header {
		writer.Header()[name] = values
	}
	writer.WriteHeader(resp.StatusCode)
	io.Copy(writer, resp.Body)
}

func getCacheKey(url string) string {
	checksum := md5.Sum([]byte(url))
	return hex.EncodeToString(checksum[:])
}
