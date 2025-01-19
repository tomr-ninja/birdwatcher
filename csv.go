package birdwatcher

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"io"
	"math"
	"net"
	"os"
	"sort"
	"strconv"
)

func LoadFromCSV(netFilePath, locFilePath string) (*IPDB, error) {
	netBlocks, err := loadNetBlocks(netFilePath)
	if err != nil {
		return nil, err
	}

	geoMap, err := loadGeos(locFilePath)
	if err != nil {
		return nil, err
	}

	db := &IPDB{}

	// convert map into slice
	geoIDMapping := make(map[uint32]uint32)
	db.geos = make([]*Geo, 0, len(geoMap))
	for geoID, geo := range geoMap {
		geoIDMapping[geoID] = uint32(len(db.geos))
		db.geos = append(db.geos, &geo)
	}

	db.keys = make([]uint32, len(netBlocks))
	db.netAddrs = make([]simpleIP, len(netBlocks))
	db.netMasks = make([]simpleMask, len(netBlocks))
	db.geoIDs = make([]uint32, len(netBlocks))
	db.ranges = make(map[uint16]lookupRange, math.MaxUint16+1)

	for i, network := range netBlocks {
		db.keys[i] = network.key
		db.netAddrs[i] = newSimpleIP(network.ipNet.IP)
		db.netMasks[i] = newSimpleMask(network.ipNet.Mask)
		db.geoIDs[i] = geoIDMapping[network.geoID]
	}

	db.prepareRanges()

	return db, nil
}

func loadNetBlocks(filePath string) ([]GeoIPNetwork, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	count, err := countLines(f)
	if err != nil {
		return nil, err
	}
	count-- // -1 for header

	_, err = f.Seek(0, 0)
	if err != nil {
		return nil, err
	}

	s := bufio.NewScanner(f)
	s.Scan() // skip header
	netBlocks := make([]GeoIPNetwork, count)
	for i := 0; i < count && s.Scan(); i++ {
		row := s.Bytes()

		netStart := 0
		netLen := bytes.IndexByte(row, ',')
		if netLen < 0 {
			return nil, io.ErrUnexpectedEOF
		}

		ipNetBytes := row[netStart : netStart+netLen]
		if len(ipNetBytes) == 0 {
			continue
		}
		_, netBlocks[i].ipNet, err = net.ParseCIDR(string(ipNetBytes)) // allocates 16 bytes instead of 4 :(
		if err != nil {
			return nil, err
		}
		// ipv4 only
		netBlocks[i].ipNet.IP = netBlocks[i].ipNet.IP.To4()
		netBlocks[i].ipNet.Mask = netBlocks[i].ipNet.Mask[len(netBlocks[i].ipNet.Mask)-4:]

		netBlocks[i].key = ipNetKey(netBlocks[i].ipNet)

		geoIDStart := netStart + netLen + 1
		geoIDLen := bytes.IndexByte(row[geoIDStart:], ',')
		if geoIDLen < 0 {
			return nil, io.ErrUnexpectedEOF
		}

		geoIDBytes := row[geoIDStart : geoIDStart+geoIDLen]
		if len(geoIDBytes) == 0 {
			continue
		}
		geoIDInt, err := strconv.Atoi(string(geoIDBytes))
		if err != nil {
			return nil, err
		}
		netBlocks[i].geoID = uint32(geoIDInt)
	}

	sort.Slice(netBlocks, func(i, j int) bool {
		return netBlocks[i].key < netBlocks[j].key
	})

	return netBlocks, nil
}

func loadGeos(filePath string) (map[uint32]Geo, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	c := csv.NewReader(f)
	cols, err := c.Read()
	if err != nil {
		return nil, err
	}

	var (
		geoIDIdx     int
		continentIdx int
		countryIdx   int
		cityIdx      int
		metroCodeIdx int
		isEUIdx      int
	)

	for i, col := range cols {
		switch col {
		case "geoname_id":
			geoIDIdx = i
		case "continent_code":
			continentIdx = i
		case "country_iso_code":
			countryIdx = i
		case "city_name":
			cityIdx = i
		case "metro_code":
			metroCodeIdx = i
		case "is_in_european_union":
			isEUIdx = i
		}
	}

	geoMap := make(map[uint32]Geo)
	for {
		row, err := c.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(row) == 0 {
			continue
		}

		geoIDInt, err := strconv.Atoi(row[geoIDIdx])
		if err != nil {
			return nil, err
		}

		continent := Continent{'-', '-'}
		if len(row[continentIdx]) == 2 {
			continent = Continent{row[continentIdx][0], row[continentIdx][1]}
		}

		country := Country{'-', '-'}
		if len(row[countryIdx]) == 2 {
			country = Country{row[countryIdx][0], row[countryIdx][1]}
		}

		geoMap[uint32(geoIDInt)] = Geo{
			Continent: continent,
			Country:   country,
			City:      row[cityIdx],
			MetroCode: row[metroCodeIdx],
			IsEU:      row[isEUIdx] == "1",
		}
	}

	return geoMap, nil
}
