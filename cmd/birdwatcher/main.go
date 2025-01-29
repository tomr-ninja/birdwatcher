package main

import (
	"log"

	"net/http"

	"github.com/tomr-ninja/birdwatcher"
)

func main() {
	ipdb, err := birdwatcher.LoadFromDump("data/geoip.dat")
	if err != nil {
		log.Fatal("failed to load IP database:", err)
	}

	http.Handle("/parse", &birdwatcher.UserHandler{IPDB: ipdb})

	if err = http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("failed to start server:", err)
	}
}
