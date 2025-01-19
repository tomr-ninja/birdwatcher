package birdwatcher

import (
	"os"

	"github.com/vmihailenco/msgpack/v5"
)

func Dump(db *IPDB, to string) error {
	d := DBDump{
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
		netAddrs: d.NetAddrs,
		netMasks: d.NetMasks,
		geoIDs:   d.GeoIDs,
		geos:     d.Geos,
	}

	return db, nil
}
