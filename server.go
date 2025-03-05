package birdwatcher

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/mailru/easyjson"
	"github.com/mileusna/useragent"
)

var (
	reqPool = sync.Pool{
		New: func() any { return new(UserRequest) },
	}
	resPool = sync.Pool{
		New: func() any { return new(UserResponse) },
	}
)

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

func populateUserResponse(resp *UserResponse, geo *Geo, ua useragent.UserAgent) {
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
	*IPDB
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
	populateUserResponse(res, geo, ua)
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
