package birdwatcher

import (
	"compress/gzip"
	"os"

	"github.com/tinylib/msgp/msgp"
)

//go:generate go run github.com/tinylib/msgp@v1.2.5 -tests=false -o dump_msgp.go

type DBDump struct {
	NetAddrs [256][256][]uint16 `msg:"adds"`
	NetMasks [256][256][]uint8  `msg:"masks"`
	GeoIDs   [256][256][]uint32 `msg:"geo_ids"`

	Geos []*Geo `msg:"geos"`
}

func Dump(db *IPDB, to string) error {
	d := DBDump{
		NetAddrs: [256][256][]uint16{},
		NetMasks: [256][256][]uint8{},
		GeoIDs:   [256][256][]uint32{},
		Geos:     db.geos,
	}

	for i, addr := range db.netAddrs {
		i1 := uint8(addr & 0xff000000 >> 24) // first 8 bits
		i2 := uint8(addr & 0x00ff0000 >> 16) // second 8 bits

		d.NetAddrs[i1][i2] = append(d.NetAddrs[i1][i2], uint16(addr&0x0000ffff))
		d.NetMasks[i1][i2] = append(d.NetMasks[i1][i2], uint8(db.netMasks[i]))
		d.GeoIDs[i1][i2] = append(d.GeoIDs[i1][i2], db.geoIDs[i])
	}

	f, err := os.Create(to)
	if err != nil {
		return err
	}
	defer f.Close()

	w := gzip.NewWriter(f)
	defer w.Close()

	return msgp.Encode(w, &d)
}

func LoadFromDump(filePath string) (*IPDB, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var d DBDump
	err = msgp.Decode(r, &d)
	if err != nil {
		return nil, err
	}

	db := &IPDB{
		geos: d.Geos,
	}

	for i1, parts := range d.NetAddrs {
		for i2, suffixes := range parts {
			for i, addrSuffix := range suffixes {
				db.netAddrs = append(db.netAddrs, simpleIP(uint32(i1)<<24|uint32(i2)<<16|uint32(addrSuffix)))
				db.netMasks = append(db.netMasks, simpleMask(d.NetMasks[i1][i2][i]))
				db.geoIDs = append(db.geoIDs, d.GeoIDs[i1][i2][i])
			}
		}
	}

	return db, nil
}
