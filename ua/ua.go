package ua

import (
	"github.com/mileusna/useragent"
)

type Data struct {
	OS        string
	UAName    string
	UAVersion int
	IsMobile  bool
	IsBot     bool
}

// Parse parses the user agent string and returns a Data struct.
//
// https://www.chromium.org/updates/ua-reduction/
// Because of UA reduction (in Chrome), only very basic information is available in the User-Agent header.
//
// TODO: Use the `Sec-CH-UA-*` headers to get more information about the user agent, if available.
func Parse(userAgent string) Data {
	v := useragent.Parse(userAgent)

	return Data{
		OS:        v.OS,
		UAName:    v.Name,
		UAVersion: v.VersionNo.Major,
		IsMobile:  v.Mobile || v.Tablet,
		IsBot:     v.Bot,
	}
}
