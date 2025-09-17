package flightradar

import (
	"fmt"
	"os"
	"path/filepath"
)

type FR24Cache struct{ base string }

func DefaultCache() (*FR24Cache, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	base := filepath.Join(dir, "fr24")
	if err := os.MkdirAll(base, 0o755); err != nil {
		return nil, err
	}
	return &FR24Cache{base: base}, nil
}

func (c *FR24Cache) LiveFeedPath(ts int64) string {
	return filepath.Join(c.base, "live_feed", fmt.Sprintf("%d.csv", ts))
}
func (c *FR24Cache) PlaybackPath(flightID string) string {
	return filepath.Join(c.base, "playback", fmt.Sprintf("%s.csv", flightID))
}
func (c *FR24Cache) FlightDetailsPath(fid uint32, ts int64) string {
	return filepath.Join(c.base, "flight_details", fmt.Sprintf("%d_%d.csv", fid, ts))
}
func (c *FR24Cache) PlaybackFlightPath(fid uint32, ts uint64) string {
	return filepath.Join(c.base, "playback_flight", fmt.Sprintf("%d_%d.csv", fid, ts))
}

func (c *FR24Cache) Base() string { return c.base }
