package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/external-api", handleExternalAPI)
	log.Fatal(http.ListenAndServe(":8081", nil))
}

func handleExternalAPI(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello from external API!")
}
