package birdwatcher

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"slices"
)

type simpleIP uint32

func newSimpleIP(ip net.IP) simpleIP {
	return simpleIP(binary.BigEndian.Uint32(ip.To4()))
}

func (ip simpleIP) IP() net.IP {
	ipBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(ipBytes, uint32(ip))

	return ipBytes
}

type simpleMask uint8

func newSimpleMask(mask net.IPMask) simpleMask {
	ones, _ := mask.Size()

	return simpleMask(ones)
}

func (m simpleMask) IPMask() net.IPMask {
	mask32 := uint32(0xffffffff) << (32 - m)
	maskBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(maskBytes, mask32)

	return maskBytes
}

type IPDB struct {
	// 3 slices in sync with each other
	keys     []uint32
	netAddrs []simpleIP
	netMasks []simpleMask
	geoIDs   []uint32

	geos []*Geo

	ranges map[uint16]lookupRange // a little speedup for the binary search
}

type lookupRange struct {
	start int
	end   int
}

func (db *IPDB) LookupIP(ipv4 net.IP) (*Geo, error) {
	if len(ipv4) != 4 {
		return nil, fmt.Errorf("invalid IP length %d", len(ipv4))
	}

	lookupIPKey := binary.BigEndian.Uint32(ipv4)
	r := db.ranges[uint16(lookupIPKey>>16)]

	i, _ := slices.BinarySearch(db.keys[r.start:r.end+1], lookupIPKey)
	if i < 1 {
		i = 1
	}

	idx := r.start + i - 1

	ipNet := net.IPNet{IP: db.netAddrs[idx].IP(), Mask: db.netMasks[idx].IPMask()}

	if !ipNet.Contains(ipv4) {
		return nil, fmt.Errorf("IP %s not found in any network", ipv4)
	}

	return db.geos[db.geoIDs[idx]], nil
}

func (db *IPDB) prepareRanges() {
	for i, key := range db.keys {
		rangeKey := uint16(key >> 16)
		if r, ok := db.ranges[rangeKey]; ok {
			r.end = i
			db.ranges[rangeKey] = r
		} else {
			db.ranges[rangeKey] = lookupRange{start: i, end: i}
		}
	}
}

type DBDump struct {
	Keys     []uint32
	NetAddrs []simpleIP
	NetMasks []simpleMask
	GeoIDs   []uint32
	Geos     []*Geo
}

type GeoIPNetwork struct {
	ipNet *net.IPNet
	geoID uint32
	key   uint32
}

type Geo struct {
	City      string    `msgpack:"cy"`
	MetroCode string    `msgpack:"mc"`
	Continent Continent `msgpack:"ct"`
	Country   Country   `msgpack:"cr"`
	IsEU      bool      `msgpack:"eu"`
}

func ipNetKey(n *net.IPNet) uint32 {
	ip := n.IP.Mask(n.Mask).To4()

	return binary.BigEndian.Uint32(ip)
}

func countLines(r io.Reader) (int, error) {
	buf := make([]byte, 32*1024)
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := r.Read(buf)
		count += bytes.Count(buf[:c], lineSep)

		switch {
		case err == io.EOF:
			return count, nil

		case err != nil:
			return 0, err
		}
	}
}
