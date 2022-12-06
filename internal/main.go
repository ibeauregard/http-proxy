package main

import (
	"fmt"
	"github.com/ibeauregard/http-proxy/internal/cache"
	"github.com/ztrue/shutdown"
	"log"
	"net/http"
	"syscall"
)

func main() {
	shutdown.Add(func() {
		cache.Persist()
	})
	go func() {
		cache.Load()
		fmt.Println("Proxy listening on http://localhost:8080")
		log.Panic(http.ListenAndServe(":8080", http.HandlerFunc(myProxy)))
	}()
	shutdown.Listen(syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
}
