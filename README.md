<p align="center">
  <img src="https://i.imgur.com/f0xxZuJ.png"/>
</p>

## Birdwatcher: because you already have Grafana

Birdwatcher is a microservice that exposes a simple pixel tracking endpoint and relies on your
existing metrics stack (Prometheus, Grafana, etc.) to collect, store and visualize your website
analytics data.

If you have a Grafana instance, you can use the provided dashboard to visualize the data. Or
you can use something else. Almost everything supports Prometheus these days.

**Then you will get yourself a poor man's Google Analytics.**

It's easy to self-host and runs perfectly fine on a Raspberry Pi. You can add the tracking pixel
to any static HTML page.

## Features

- No cookies, no JavaScript
- Integrated GeoIP
- Integrated User-Agent parsing
- 'API-mode' in case if you just need the GeoIP and User-Agent data (`/parse`)
- Probably fully GDPR-compliant, no need to even ask for consent

There are some limitations of the chosen approach:

- Doesn't count unique visitors (sorry)
- No Chrome Client Hints (yet)

## Build yourself

Prerequisites:
- Go 1.23+

First, you need to download the GeoIP database from MaxMind. You need to create an account and get a license key.

```bash
git clone https://github.com/tomr-ninja/birdwatcher.git
cd birdwatcher
go build ./cmd/geoip
./geoip download --account-id={YOUR_MAXMIND_ACCOUNT} --license-key={YOUR_MAXMIND_LICENSE_KEY}
```

By default, the `download` command will create a folder and extract files there. Expected output:

```
Extracted GeoLite2-City-Locations-en.csv to GeoLite2-City-CSV_20250415
Extracted GeoLite2-City-Blocks-IPv4.csv to GeoLite2-City-CSV_20250415
```

You don't have to use the 'en' locale (see `geoip download --help` & `geoip compile --help`)

The next step is to compile a binary database file.

```bash
./geoip compile \
  --blocks-csv=./GeoLite2-City-CSV_20250415/GeoLite2-City-Blocks-IPv4.csv \
  --locations-csv=./GeoLite2-City-CSV_20250415/GeoLite2-City-Locations-en.csv
```

The command will produce a `geoip.dat` file. You can test the compiled database with the `geoip exec` command:

```bash
./geoip exec 109.68.230.145
# City: Berlin
# Region: State of Berlin
# Country: Germany (DE)
```

Now you can build and run the Birdwatcher service.

You can build it using Docker:

```bash
docker build -t birdwatcher:20250415 .
docker run --rm -p 3000:3000 birdwatcher:20250415
```

Or just compile a binary:

```bash
go build ./cmd/birdwatcher
./birdwatcher
```

By default, it will use the `geoip.dat` in the current directory. You can specify a different path via an env variable:

```bash
docker run --rm -p 3000:3000 -e GEOIP_DB="/path/to/file" -v /path/to/file:/path/to/file birdwatcher:20250415
```

```bash
GEOIP_DB="/path/to/file" ./birdwatcher
```
