package birdwatcher

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/VictoriaMetrics/metrics"
	"github.com/mailru/easyjson"
	"github.com/mileusna/useragent"
)

const (
	labelUnknown = "<UNKNOWN>"
	labelError   = "<ERROR>"
)

var (
	reqPool = sync.Pool{
		New: func() any { return new(UserRequest) },
	}
	resPool = sync.Pool{
		New: func() any { return new(UserResponse) },
	}

	pixelGIF = func() []byte {
		img := image.NewRGBA(image.Rect(0, 0, 1, 1))
		img.Set(0, 0, color.RGBA{R: 255, A: 0})

		var buf bytes.Buffer
		_ = gif.Encode(&buf, img, nil)

		return buf.Bytes()
	}()
)

type ipdb interface {
	LookupIP(net.IP) (*Geo, error)
}

//go:generate easyjson $GOFILE

//easyjson:json
type UserRequest struct {
	IP string `json:"ip"`
	UA string `json:"ua"`
}

//easyjson:json
type UserResponse struct {
	Geo struct {
		City       string `json:"city"`
		Region     string `json:"state_province"`
		Country    string `json:"country"`
		CountryISO string `json:"country_iso"`
		IsEU       bool   `json:"is_eu"`
	} `json:"geo"`
	UserAgent struct {
		Name      string `json:"name"`
		Version   string `json:"version"`
		OS        string `json:"os"`
		OSVersion string `json:"os_version"`
		Device    string `json:"device"`
		Mobile    bool   `json:"mobile"`
		Tablet    bool   `json:"tablet"`
		Desktop   bool   `json:"desktop"`
		Bot       bool   `json:"bot"`
	} `json:"user_agent"`
}

func (resp *UserResponse) populate(geo *Geo, ua useragent.UserAgent) {
	resp.Geo.City = geo.City
	resp.Geo.Region = geo.Region
	resp.Geo.Country = geo.Country.String()
	resp.Geo.CountryISO = string(geo.Country[:])
	resp.Geo.IsEU = geo.IsEU
	resp.UserAgent.Name = ua.Name
	resp.UserAgent.Version = ua.Version
	resp.UserAgent.OS = ua.OS
	resp.UserAgent.OSVersion = ua.OSVersion
	resp.UserAgent.Device = ua.Device
	resp.UserAgent.Mobile = ua.Mobile
	resp.UserAgent.Tablet = ua.Tablet
	resp.UserAgent.Desktop = ua.Desktop
	resp.UserAgent.Bot = ua.Bot
}

type UserHandler struct {
	IPDB ipdb
}

func (h *UserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Parse the request
	req := reqPool.Get().(*UserRequest)
	defer reqPool.Put(req)

	switch r.Method {
	case http.MethodGet:
		req.IP = r.URL.Query().Get("ip")
		req.UA = r.URL.Query().Get("ua")
	case http.MethodPost, http.MethodPut:
		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			respondError(w, err, http.StatusBadRequest)
			return
		}
	default:
		respondError(w, fmt.Errorf("method not allowed"), http.StatusMethodNotAllowed)
	}

	// Lookup the IP
	geo, err := h.IPDB.LookupIP(net.ParseIP(req.IP))
	if err != nil {
		respondError(w, err, http.StatusInternalServerError)
		return
	}

	// Parse the user agent
	ua := useragent.Parse(req.UA)

	// Send the response
	w.Header().Set("Content-Type", "application/json")
	res := resPool.Get().(*UserResponse)
	defer resPool.Put(res)
	res.populate(geo, ua)
	if _, err = easyjson.MarshalToWriter(res, w); err != nil {
		respondError(w, err, http.StatusInternalServerError)
		return
	}
}

//easyjson:json
type ErrorResponse struct {
	Error string `json:"error"`
}

func respondError(w http.ResponseWriter, err error, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	_, _ = easyjson.MarshalToWriter(ErrorResponse{Error: err.Error()}, w)
}

type PixelHandler struct {
	IPDB    ipdb
	Metrics *metrics.Set
}

func (h *PixelHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	geo, err := h.IPDB.LookupIP(net.ParseIP(r.RemoteAddr))
	if err != nil {
		respondError(w, err, http.StatusInternalServerError)
		return
	}

	// https://www.chromium.org/updates/ua-reduction/
	//
	// Because of UA reduction (in Chrome), only very basic information is available in the User-Agent header:
	// platform (Windows, Android, etc.), architecture (x86, ARM, etc.), and the browser's major version.
	//
	// TODO: Use the `Sec-CH-UA-*` headers to get more information about the user agent, if available.
	ua := useragent.Parse(r.UserAgent())

	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Referer
	referer := r.Header.Get("Referer")
	if len(referer) == 0 {
		referer = r.URL.Query().Get("bw-referrer")
		if len(referer) == 0 {
			referer = r.URL.Query().Get("bw-referer") // a common (and standardized) typo
		}
	}

	h.fire(referer, geo, ua)

	hs := w.Header()
	hs.Set("Content-Type", "image/gif")
	hs.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	hs.Set("Pragma", "no-cache")
	hs.Set("Expires", "0")

	_, _ = w.Write(pixelGIF)
}

func (h *PixelHandler) fire(referer string, geo *Geo, ua useragent.UserAgent) {
	var (
		host = labelError
		page = labelError
	)
	if len(referer) != 0 {
		if u, err := url.Parse(referer); err == nil {
			host = u.Host
			page = u.Path
		}
	}

	var (
		country = cmp.Or(string(geo.Country[:]), labelUnknown)
		region  = cmp.Or(geo.Region, labelUnknown)
		city    = cmp.Or(geo.City, labelUnknown)
		isEU    = strconv.FormatBool(geo.IsEU)

		uaName    = cmp.Or(ua.Name, labelUnknown)
		uaVersion = cmp.Or(ua.Version, labelUnknown)
		os        = cmp.Or(ua.OS, labelUnknown)
		isBot     = strconv.FormatBool(ua.Bot)
		isMobile  = strconv.FormatBool(ua.Mobile || ua.Tablet)
	)

	var b strings.Builder
	b.Grow(128)
	b.WriteString(`page_visits{host="`)
	b.WriteString(host)
	b.WriteString(`",page="`)
	b.WriteString(page)
	b.WriteString(`",country="`)
	b.WriteString(country)
	b.WriteString(`",region="`)
	b.WriteString(region)
	b.WriteString(`",city="`)
	b.WriteString(city)
	b.WriteString(`",is_eu="`)
	b.WriteString(isEU)
	b.WriteString(`",ua_name="`)
	b.WriteString(uaName)
	b.WriteString(`",ua_version="`)
	b.WriteString(uaVersion)
	b.WriteString(`",os="`)
	b.WriteString(os)
	b.WriteString(`",is_bot="`)
	b.WriteString(isBot)
	b.WriteString(`",is_mobile="`)
	b.WriteString(isMobile)
	b.WriteString(`"}`)

	h.Metrics.GetOrCreateCounter(b.String()).Inc()
}
