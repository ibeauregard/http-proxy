package main

import (
	"log"
	"net/http"
)

func main() {
	log.Panic(http.ListenAndServe(":8080", http.HandlerFunc(myProxy)))
}
