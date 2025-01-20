package birdwatcher

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/csv"
	"io"
	"net"
	"os"
	"sort"
	"strconv"
)

type GeoIPNetwork struct {
	netAddr uint32
	netMask uint32
	geoID   uint32
}

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

	db.netAddrs = make([]simpleIP, len(netBlocks))
	db.netMasks = make([]simpleMask, len(netBlocks))
	db.geoIDs = make([]uint32, len(netBlocks))

	for i, network := range netBlocks {
		mask := make(net.IPMask, 4)
		binary.BigEndian.PutUint32(mask, network.netMask)
		db.netAddrs[i] = simpleIP(network.netAddr)
		db.netMasks[i] = newSimpleMask(mask)
		db.geoIDs[i] = geoIDMapping[network.geoID]
	}

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
		_, ipNet, err := net.ParseCIDR(string(ipNetBytes))
		if err != nil {
			return nil, err
		}
		// ipv4 only
		ipNet.IP = ipNet.IP.To4()
		ipNet.Mask = ipNet.Mask[len(ipNet.Mask)-4:]

		netBlocks[i].netAddr = binary.BigEndian.Uint32(ipNet.IP)
		netBlocks[i].netMask = binary.BigEndian.Uint32(ipNet.Mask)

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
		return netBlocks[i].netAddr&netBlocks[i].netMask < netBlocks[j].netAddr&netBlocks[j].netMask
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
		geoIDIdx   int
		countryIdx int
		cityIdx    int
		regionIdx  int
		isEUIdx    int
	)

	for i, col := range cols {
		switch col {
		case "geoname_id":
			geoIDIdx = i
		case "country_iso_code":
			countryIdx = i
		case "city_name":
			cityIdx = i
		case "subdivision_1_name":
			regionIdx = i
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

		country := Country{'-', '-'}
		if len(row[countryIdx]) == 2 {
			country = Country{row[countryIdx][0], row[countryIdx][1]}
		}

		geoMap[uint32(geoIDInt)] = Geo{
			Country: country,
			City:    row[cityIdx],
			Region:  row[regionIdx],
			IsEU:    row[isEUIdx] == "1",
		}
	}

	return geoMap, nil
}
