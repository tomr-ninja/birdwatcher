package main

import (
	"cmp"
	"log"
	"net/http"
	"os"

	"github.com/VictoriaMetrics/metrics"

	"github.com/tomr-ninja/birdwatcher"
)

func main() {
	dbPath := cmp.Or(os.Getenv("GEOIP_DB"), "data/geoip.dat")

	ipdb, err := birdwatcher.LoadFromDump(dbPath)
	if err != nil {
		log.Fatal("failed to load IP database:", err)
	}

	metricsSet := metrics.NewSet()

	http.Handle("/parse", &birdwatcher.UserHandler{IPDB: ipdb})
	http.Handle("/pixel.gif", &birdwatcher.PixelHandler{IPDB: ipdb, Metrics: metricsSet})
	http.HandleFunc("/metrics", func(w http.ResponseWriter, req *http.Request) {
		metricsSet.WritePrometheus(w)
	})

	if err = http.ListenAndServe(":3000", nil); err != nil {
		log.Fatal("failed to start server:", err)
	}
}
