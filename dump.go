package birdwatcher

import (
	"compress/gzip"
	"os"

	"github.com/tinylib/msgp/msgp"
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
		netAddrs: d.NetAddrs,
		netMasks: d.NetMasks,
		geoIDs:   d.GeoIDs,
		geos:     d.Geos,
	}

	return db, nil
}
