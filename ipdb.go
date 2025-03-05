package birdwatcher

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"slices"
)

//go:generate go run github.com/tinylib/msgp@v1.2.5 -tests=false -o ipdb_msgp.go

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

//msgp:ignore IPDB
type IPDB struct {
	// 3 slices in sync with each other
	netAddrs []simpleIP
	netMasks []simpleMask
	geoIDs   []uint32

	geos []*Geo
}

func (db *IPDB) LookupIP(ipv4 net.IP) (*Geo, error) {
	if ipv4 = ipv4.To4(); ipv4 == nil {
		return nil, fmt.Errorf("invalid IP")
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

//msgp:tuple Geo
//msgp:replace Country with:[2]byte
type Geo struct {
	City    string  `msg:"cy"`
	Region  string  `msg:"re"`
	Country Country `msg:"cr"`
	IsEU    bool    `msg:"eu"`
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
