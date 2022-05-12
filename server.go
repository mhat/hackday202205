package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	fmt.Println("Listening on :8888")
	fileServer := http.FileServer(http.Dir("."))
	if err := http.ListenAndServe(":8888", fileServer); err != nil {
		log.Fatal(err)
	}
}
