package main

import (
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/tomr-ninja/flag3"

	"github.com/tomr-ninja/birdwatcher"
)

const (
	defaultNetFilePath  = "data/GeoLite2-City-Blocks-IPv4.csv"
	defaultLocFilePath  = "data/GeoLite2-City-Locations-en.csv"
	defaultDumpFilePath = "data/geoip.dat"
)

func main() {
	tree := flag3.NewCLI()
	tree.Subcommand("compile")
	tree.Subcommand("exec")

	cmd, err := flag3.ParseCLI(tree)
	if err != nil {
		fatal(err)
	}

	_ = cmd.Next()
	executableName := cmd.Command()

	ok := cmd.Next()
	if !ok {
		fmt.Printf("Usage: %s [compile|exec]\n", executableName)
	}

	switch cmd.Command() {
	case "compile":
		var blocksCSVPath, locationsCSVPath, dumpPath string

		fs := flag.NewFlagSet("compile", flag.ContinueOnError)
		fs.StringVar(&blocksCSVPath, "blocks-csv", defaultNetFilePath, "Path to GeoLite2-City-Blocks-IPv4.csv")
		fs.StringVar(&locationsCSVPath, "locations-csv", defaultLocFilePath, "Path to GeoLite2-City-Locations-{locale}.csv")
		fs.StringVar(&dumpPath, "out", defaultDumpFilePath, "Compiled database output path")

		if err = fs.Parse(cmd.Args()); err != nil {
			fatal(err)
		}

		if err = compile(blocksCSVPath, locationsCSVPath, dumpPath); err != nil {
			fatal(err)
		}
	case "exec":
		var dumpPath, ipArg string

		fs := flag.NewFlagSet("exec", flag.ContinueOnError)
		fs.StringVar(&dumpPath, "db", defaultDumpFilePath, "Path to database file")

		if err = fs.Parse(cmd.Args()); err != nil {
			fatal(err)
		}

		if len(fs.Args()) < 1 {
			fatal(fmt.Errorf("missing IP argument"))
		}
		ipArg = fs.Args()[0]

		if err = exec(dumpPath, ipArg); err != nil {
			fatal(err)
		}
	}
}

func exec(dumpPath, ipArg string) error {
	ipDB, err := birdwatcher.LoadFromDump(dumpPath)
	if err != nil {
		return err
	}

	geo, err := ipDB.LookupIP(net.ParseIP(ipArg).To4())
	if err != nil {
		return fmt.Errorf("failed to lookup IP '%s': %w", ipArg, err)
	}

	fmt.Printf(
		"City: %s\nRegion: %s\nCountry: %s (%s)\n",
		geo.City, geo.Region, geo.Country, string(geo.Country[:]),
	)

	return nil
}

func compile(netFilePath, locFilePath, outPath string) error {
	ipDB, err := birdwatcher.LoadFromCSV(netFilePath, locFilePath)
	if err != nil {
		return err
	}

	if err = birdwatcher.Dump(ipDB, outPath); err != nil {
		return err
	}

	return nil
}

func fatal(err error) {
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
