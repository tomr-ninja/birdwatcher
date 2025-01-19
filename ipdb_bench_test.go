package birdwatcher_test

import (
	"net"
	"testing"

	"github.com/tomr-ninja/birdwatcher"
)

func BenchmarkIPDB_LookupIP(b *testing.B) {
	ipDB, err := birdwatcher.LoadFromDump("data/geoip.dat")
	if err != nil {
		b.Fatal(err)
	}

	myIP := net.ParseIP("5.28.100.233").To4()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = ipDB.LookupIP(myIP)
	}
}
