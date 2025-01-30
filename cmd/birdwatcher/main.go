package main

import (
	"flag"
	"fmt"
	"log"

	"net/http"

	"github.com/tomr-ninja/birdwatcher"
)

const (
	defaultDumpFilePath = "data/geoip.dat"
	defaultListenPort   = 8080
)

func main() {
	var (
		dumpPath string
		port     uint64
	)

	flag.StringVar(&dumpPath, "db", defaultDumpFilePath, "Path to GeoIP database file")
	flag.Uint64Var(&port, "port", defaultListenPort, "Port to listen on")
	flag.Parse()

	ipdb, err := birdwatcher.LoadFromDump(dumpPath)
	if err != nil {
		log.Fatal("failed to load IP database:", err)
	}

	http.Handle("/parse", &birdwatcher.UserHandler{IPDB: ipdb})

	if err = http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		log.Fatal("failed to start server:", err)
	}
}
