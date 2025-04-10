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

	"github.com/tomr-ninja/birdwatcher/ua"
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
		Name     string `json:"name"`
		Version  string `json:"version"`
		OS       string `json:"os"`
		IsMobile bool   `json:"is_mobile"`
		IsBot    bool   `json:"is_bot"`
	} `json:"user_agent"`
}

func (resp *UserResponse) populate(geo *Geo, ua ua.Data) {
	resp.Geo.City = geo.City
	resp.Geo.Region = geo.Region
	resp.Geo.Country = geo.Country.String()
	resp.Geo.CountryISO = string(geo.Country[:])
	resp.Geo.IsEU = geo.IsEU
	resp.UserAgent.Name = ua.UAName
	resp.UserAgent.Version = strconv.FormatInt(int64(ua.UAVersion), 10)
	resp.UserAgent.OS = ua.OS
	resp.UserAgent.IsMobile = ua.IsMobile
	resp.UserAgent.IsBot = ua.IsBot
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

	// Send the response
	w.Header().Set("Content-Type", "application/json")
	res := resPool.Get().(*UserResponse)
	defer resPool.Put(res)
	res.populate(geo, ua.Parse(req.UA))
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

	uaData := ua.Parse(r.UserAgent())

	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Referer
	referer := r.Header.Get("Referer")
	if len(referer) == 0 {
		referer = r.URL.Query().Get("bw-referrer")
		if len(referer) == 0 {
			referer = r.URL.Query().Get("bw-referer") // a common (and standardized) typo
		}
	}

	h.fire(referer, geo, uaData)

	hs := w.Header()
	hs.Set("Content-Type", "image/gif")
	hs.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	hs.Set("Pragma", "no-cache")
	hs.Set("Expires", "0")

	_, _ = w.Write(pixelGIF)
}

func (h *PixelHandler) fire(referer string, geo *Geo, ua ua.Data) {
	host, page := labelError, labelError
	if len(referer) != 0 {
		if u, err := url.Parse(referer); err == nil {
			host = u.Host
			page = u.Path
		}
	}

	uaVersion := labelUnknown
	if ua.UAVersion != 0 {
		uaVersion = strconv.FormatInt(int64(ua.UAVersion), 10)
	}

	m := formatMetric(
		"page_visits",
		"host", host,
		"page", page,
		"country", cmp.Or(string(geo.Country[:]), labelUnknown),
		"region", cmp.Or(geo.Region, labelUnknown),
		"city", cmp.Or(geo.City, labelUnknown),
		"is_eu", strconv.FormatBool(geo.IsEU),
		"ua_name", cmp.Or(ua.UAName, labelUnknown),
		"ua_version", uaVersion,
		"os", cmp.Or(ua.OS, labelUnknown),
		"is_bot", strconv.FormatBool(ua.IsBot),
		"is_mobile", strconv.FormatBool(ua.IsMobile),
	)

	h.Metrics.GetOrCreateCounter(m).Inc()
}

func formatMetric(name string, tags ...string) string {
	if len(tags)%2 != 0 {
		panic("invalid tags")
	}

	var b strings.Builder
	b.Grow(len(name) + len(tags)*32) // imperfect heuristic

	b.WriteString(name)
	b.WriteByte('{')
	for i := 0; i < len(tags); i += 2 {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(tags[i])
		b.WriteString(`="`)
		b.WriteString(tags[i+1])
		b.WriteByte('"')
	}
	b.WriteByte('}')

	return b.String()
}
