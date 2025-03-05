<p align="center">
  <img src="https://i.imgur.com/f0xxZuJ.png"/>
</p>

## Birdwatcher: because you already have Grafana

Birdwatcher is a microservice that exposes a simple pixel tracking endpoint and relies on your
existing metrics stack (Prometheus, Grafana, etc.) to collect, store and visualize the data.

If you have a Grafana instance, you can use the provided dashboard to visualize the data. Or
you can use something else. Almost everything supports Prometheus these days.

**Then you will get yourself a poor man's Google Analytics.**

It's easy to self-host and runs perfectly fine on a Raspberry Pi. You can add the tracking pixel
to any static HTML page.

### Features

- No cookies, no JavaScript
- Integrated GeoIP
- Integrated User-Agent parsing
- 'API-mode' in case if you just need the GeoIP and User-Agent data (`/parse`)
- Probably fully GDPR-compliant, no need to even ask for consent

There are some limitations of the chosen approach:

- Doesn't count unique visitors (sorry)
- No Chrome Client Hints (yet)
