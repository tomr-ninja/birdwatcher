package birdwatcher

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/VictoriaMetrics/metrics"
)

func TestPixelHandler_ServeHTTP(t *testing.T) {
	h := &PixelHandler{
		IPDB:    &mockIPDB{},
		Metrics: metrics.NewSet(),
	}

	req, _ := http.NewRequest(http.MethodGet, "https://example.org/pixel.gif", nil)
	req.Header.Set("Referer", "https://example.org/some/page")
	req.Header.Set("User-Agent", `Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/93.0.0.0 Mobile Safari/537.36`)
	pixelResp := httptest.NewRecorder()

	h.ServeHTTP(pixelResp, req)

	if pixelResp.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, pixelResp.Code)
	}

	metricsResp := httptest.NewRecorder()
	h.Metrics.WritePrometheus(metricsResp)

	if metricsResp.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, metricsResp.Code)
	}

	actual := metricsResp.Body.String()
	expected := `page_visits{host="example.org",page="/some/page",country="DE",region="Land Berlin",city="Berlin",is_eu="true",ua_name="Chrome",ua_version="93",os="Android",is_bot="false",is_mobile="true"} 1` + "\n"
	if actual != expected {
		t.Errorf("metrics: expected %q, got %q", expected, actual)
	}
}

type mockIPDB struct{}

func (m *mockIPDB) LookupIP(_ net.IP) (*Geo, error) {
	return &Geo{
		City:    "Berlin",
		Region:  "Land Berlin",
		Country: Country{'D', 'E'},
		IsEU:    true,
	}, nil
}
