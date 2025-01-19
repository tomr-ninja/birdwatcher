package main

import (
	"fmt"
	"net"

	"github.com/tomr-ninja/birdwatcher"
)

const (
	defaultNetFilePath = "data/GeoLite2-City-Blocks-IPv4.csv"
	defaultLocFilePath = "data/GeoLite2-City-Locations-en.csv"
)

var myIP = net.ParseIP("5.28.100.233").To4()

func main() {
	//loadCSV(true)
	loadDump()
}

func loadDump() {
	ipDB, err := birdwatcher.LoadFromDump("data/geoip.dat")
	if err != nil {
		panic(err)
	}

	geo, err := ipDB.LookupIP(myIP)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf(
		"IP=%s found: %s, %s",
		myIP, geo.City, geo.Country,
	)
}

func loadCSV(dump bool) {
	ipDB, err := birdwatcher.LoadFromCSV(defaultNetFilePath, defaultLocFilePath)
	if err != nil {
		panic(err)
	}

	geo, err := ipDB.LookupIP(myIP)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf(
		"IP=%s found: %s, %s",
		myIP, geo.City, geo.Country,
	)

	if dump {
		if err = birdwatcher.Dump(ipDB, "data/geoip.dat"); err != nil {
			panic(err)
		}
	}
}
