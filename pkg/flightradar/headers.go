package flightradar

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
)

// defaultJSONHeaders returns the base set of headers used for FR24 JSON calls.
func defaultJSONHeaders(device string) map[string][]string {
	h := map[string][]string{
		"User-Agent":      {"Mozilla/5.0 (X11; Linux x86_64; rv:136.0) Gecko/20100101 Firefox/136.0"},
		"Accept":          {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8"},
		"Accept-Language": {"en-US,en;q=0.5"},
		"Origin":          {"https://www.flightradar24.com"},
		"Connection":      {"keep-alive"},
		"Referer":         {"https://www.flightradar24.com/"},
		"Sec-Fetch-Dest":  {"empty"},
		"Sec-Fetch-Mode":  {"cors"},
		"Sec-Fetch-Site":  {"same-site"},
		"TE":              {"trailers"},
	}
	if device != "" {
		h["fr24-device-id"] = []string{device}
	}
	return h
}

// newDeviceID creates a random web-like device id similar to the Python impl.
func newDeviceID() string {
	// Use URL-safe base64 to mimic secrets.token_urlsafe(32).
	// token_urlsafe(n) -> base64url(b) where len(b)=n bytes.
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// fallback to a short constant if RNG fails (unlikely)
		return "web-anon"
	}
	// StdEncoding with RawURLEncoding style without padding
	s := base64.RawURLEncoding.EncodeToString(b)
	return "web-" + strings.TrimRight(s, "=")
}
