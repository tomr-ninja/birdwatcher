package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/tomr-ninja/flag3"

	"github.com/tomr-ninja/birdwatcher"
)

const defaultDumpFilePath = "geoip.dat"

var folderRegex = regexp.MustCompile("GeoLite2-City-CSV_[0-9]{8}")

func main() {
	tree := flag3.NewCLI()
	tree.Subcommand("download")
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
	case "download":
		var accountID, licenseKey, includeLocalesStr, outDir string

		fs := flag.NewFlagSet("download", flag.ContinueOnError)
		fs.StringVar(&accountID, "account-id", "", "MaxMind account ID")
		fs.StringVar(&licenseKey, "license-key", "", "MaxMind license key")
		fs.StringVar(&includeLocalesStr, "locales", "en", "Comma-separated list of locales to download")
		fs.StringVar(&outDir, "out", "", "Output directory")

		if err = fs.Parse(cmd.Args()); err != nil {
			fatal(err)
		}
		if accountID == "" {
			fatal(fmt.Errorf("missing account ID or license key"))
		}
		if licenseKey == "" {
			fatal(fmt.Errorf("missing account ID or license key"))
		}
		includeLocales := []string{"en"}
		if includeLocalesStr != "" {
			includeLocales = strings.Split(includeLocalesStr, ",")
		}
		if err = download(accountID, licenseKey, includeLocales, outDir); err != nil {
			fatal(err)
		}
	case "compile":
		var blocksCSVPath, locationsCSVPath, dumpPath string

		fs := flag.NewFlagSet("compile", flag.ContinueOnError)
		fs.StringVar(&blocksCSVPath, "blocks-csv", "", "Path to GeoLite2-City-Blocks-IPv4.csv")
		fs.StringVar(&locationsCSVPath, "locations-csv", "", "Path to GeoLite2-City-Locations-{locale}.csv")
		fs.StringVar(&dumpPath, "out", defaultDumpFilePath, "Compiled database output path")

		if err = fs.Parse(cmd.Args()); err != nil {
			fatal(err)
		}
		if blocksCSVPath == "" {
			fatal(fmt.Errorf("missing IPv4 blocks CSV file"))
		}
		if locationsCSVPath == "" {
			fatal(fmt.Errorf("missing locations CSV file"))
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

func download(
	accountID, licenseKey string,
	includeLocales []string,
	outPath string,
) error {
	req, _ := http.NewRequest(
		http.MethodGet,
		"https://download.maxmind.com/geoip/databases/GeoLite2-City-CSV/download?suffix=zip",
		nil,
	)
	req.SetBasicAuth(accountID, licenseKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: %s", resp.Status)
	}

	tmpFile, err := os.CreateTemp("", "GeoLite2-City-CSV-*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	size, err := io.Copy(tmpFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	if err = tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}
	tmpFile, err = os.Open(tmpFile.Name())
	if err != nil {
		return fmt.Errorf("failed to open temporary file: %w", err)
	}
	defer tmpFile.Close()
	r, err := zip.NewReader(tmpFile, size)
	if err != nil {
		return fmt.Errorf("failed to create zip reader: %w", err)
	}

	// unpack requested files
	fileNames := make(map[string]struct{}, len(includeLocales)+1)
	fileNames["GeoLite2-City-Blocks-IPv4.csv"] = struct{}{}
	for _, locale := range includeLocales {
		fileNames[fmt.Sprintf("GeoLite2-City-Locations-%s.csv", locale)] = struct{}{}
	}

	for _, f := range r.File {
		folder, name := path.Split(f.Name)
		if folder == "" || !folderRegex.MatchString(folder) {
			continue
		}

		if _, ok := fileNames[name]; !ok {
			continue
		}

		rc, fErr := f.Open()
		if fErr != nil {
			return fmt.Errorf("failed to open zip file: %w", fErr)
		}
		defer rc.Close()

		if outPath == "" {
			outPath = path.Join(outPath, folder)
		}
		if fErr = os.MkdirAll(outPath, 0755); fErr != nil {
			return fmt.Errorf("failed to create output directory: %w", fErr)
		}
		outFile, fErr := os.Create(path.Join(outPath, name))
		if fErr != nil {
			return fmt.Errorf("failed to create output file: %w", fErr)
		}
		defer outFile.Close()

		if _, fErr = io.Copy(outFile, rc); fErr != nil {
			return fmt.Errorf("failed to copy file: %w", fErr)
		}

		fmt.Printf("Extracted %s to %s\n", name, outPath)
	}

	return nil
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
