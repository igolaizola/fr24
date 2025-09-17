package flightradar

import (
	"encoding/hex"
	"strconv"
	"strings"
	"time"
)

// UnixNow returns current unix seconds.
func UnixNow() int64 { return time.Now().Unix() }

// ToUnixSeconds attempts to normalize inputs to unix seconds.
// Accepts strings like "now", decimal seconds, or ISO8601 that Go parses.
// For this MVP, support only "now" and integer strings; callers can pass nil
// to indicate now for optional timestamps.
func ToUnixSeconds(s string) (int64, bool) {
	if s == "now" {
		return UnixNow(), true
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n, true
	}
	return 0, false
}

// ToFlightIDHex normalizes a flight id to lower-case hex without 0x prefix.
func ToFlightIDHex(v string) string {
	v = strings.ToLower(strings.TrimPrefix(v, "0x"))
	// validate hex-ish; if not hex, return as-is and let server reject
	if _, err := hex.DecodeString(v); err == nil {
		return v
	}
	return v
}
