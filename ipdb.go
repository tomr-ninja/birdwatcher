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
	netAddrs []simpleIP
	netMasks []simpleMask
	geoIDs   []uint32

	geos []*Geo
}

func (db *IPDB) LookupIP(ipv4 net.IP) (*Geo, error) {
	if len(ipv4) != 4 {
		return nil, fmt.Errorf("invalid IP length %d", len(ipv4))
	}

	lookupIPKey := binary.BigEndian.Uint32(ipv4)

	i, found := slices.BinarySearch(db.netAddrs, simpleIP(lookupIPKey))
	if !found && i > 0 {
		i -= 1
	}

	ipNet := net.IPNet{IP: db.netAddrs[i].IP(), Mask: db.netMasks[i].IPMask()}

	if !ipNet.Contains(ipv4) {
		return nil, fmt.Errorf("IP %s not found in any network", ipv4)
	}

	return db.geos[db.geoIDs[i]], nil
}

type DBDump struct {
	NetAddrs []simpleIP
	NetMasks []simpleMask
	GeoIDs   []uint32
	Geos     []*Geo
}

type GeoIPNetwork struct {
	netAddr uint32
	netMask uint32
	geoID   uint32
}

type Geo struct {
	City      string  `msgpack:"cy"`
	MetroCode string  `msgpack:"mc"`
	Country   Country `msgpack:"cr"`
	IsEU      bool    `msgpack:"eu"`
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
