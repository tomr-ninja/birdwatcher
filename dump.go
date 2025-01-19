package birdwatcher

import (
	"math"
	"os"

	"github.com/vmihailenco/msgpack/v5"
)

func Dump(db *IPDB, to string) error {
	d := DBDump{
		Keys:     db.keys,
		NetAddrs: db.netAddrs,
		NetMasks: db.netMasks,
		GeoIDs:   db.geoIDs,
		Geos:     db.geos,
	}

	f, err := os.Create(to)
	if err != nil {
		return err
	}
	defer f.Close()

	return msgpack.NewEncoder(f).Encode(d)
}

func LoadFromDump(filePath string) (*IPDB, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var d DBDump
	err = msgpack.NewDecoder(f).Decode(&d)
	if err != nil {
		return nil, err
	}

	db := &IPDB{
		keys:     d.Keys,
		netAddrs: d.NetAddrs,
		netMasks: d.NetMasks,
		geoIDs:   d.GeoIDs,
		geos:     d.Geos,
		ranges:   make(map[uint16]lookupRange, math.MaxUint16+1),
	}

	db.prepareRanges()

	return db, nil
}
