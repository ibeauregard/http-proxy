package main

import (
	"io"
	"log"
	"net/http"
)

func main() {
	myServer := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		// Request method names are case-sensitive
		// See https://www.rfc-editor.org/rfc/rfc7230#section-3.1.1
		if request.Method != "GET" {
			http.Error(writer, "Invalid request method; use GET", http.StatusMethodNotAllowed)
			return
		}
		resp, err := http.Get(request.URL.Query().Get("page"))
		if err != nil {
			return
		}
		defer resp.Body.Close()
		for name, values := range resp.Header {
			writer.Header()[name] = values
		}
		writer.WriteHeader(resp.StatusCode)
		io.Copy(writer, resp.Body)
	})
	log.Fatal(http.ListenAndServe(":8080", myServer))
}
