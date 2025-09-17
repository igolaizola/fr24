package flightradar

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// ---- Flight List ----

type FlightListParams struct {
	// Exactly one of Reg or Flight must be set
	Reg    string
	Flight string
	Page   int
	Limit  int
	// TimestampS: nil => now; otherwise use provided unix seconds
	TimestampS *int64
}

func (p *FlightListParams) validate() error {
	if (p.Reg == "" && p.Flight == "") || (p.Reg != "" && p.Flight != "") {
		return fmt.Errorf("exactly one of Reg or Flight is required")
	}
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.Limit <= 0 {
		p.Limit = 10
	}
	return nil
}

// FlightListRecord matches the flattened record produced by Python utils.
type FlightListRecord struct {
	FlightID     *int64  `json:"flight_id,omitempty"`
	Number       *string `json:"number,omitempty"`
	Callsign     *string `json:"callsign,omitempty"`
	ICAO24       *int64  `json:"icao24,omitempty"`
	Registration *string `json:"registration,omitempty"`
	Typecode     *string `json:"typecode,omitempty"`
	Origin       *string `json:"origin,omitempty"`
	Destination  *string `json:"destination,omitempty"`
	Status       *string `json:"status,omitempty"`
	STOD         *int64  `json:"STOD,omitempty"`
	ETOD         *int64  `json:"ETOD,omitempty"`
	ATOD         *int64  `json:"ATOD,omitempty"`
	STOA         *int64  `json:"STOA,omitempty"`
	ETOA         *int64  `json:"ETOA,omitempty"`
	ATOA         *int64  `json:"ATOA,omitempty"`
}

// FlightList performs the JSON flight list request and returns the raw HTTP response.
func (c *Client) FlightList(ctx context.Context, p FlightListParams) (*http.Response, error) {
	if err := p.validate(); err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Set("query", firstNonEmpty(p.Reg, p.Flight))
	if p.Reg != "" {
		q.Set("fetchBy", "reg")
	} else {
		q.Set("fetchBy", "flight")
	}
	q.Set("page", strconv.Itoa(zeroDefault(p.Page, 1)))
	q.Set("limit", strconv.Itoa(zeroDefault(p.Limit, 10)))
	if p.TimestampS != nil {
		q.Set("timestamp", strconv.FormatInt(*p.TimestampS, 10))
	} else {
		q.Set("timestamp", strconv.FormatInt(UnixNow(), 10))
	}
	withAuthParams(&q, c.subscriptionKey, c.deviceID)

	req, _ := http.NewRequest("GET", "https://api.flightradar24.com/common/v1/flight/list.json", nil)
	req.URL.RawQuery = q.Encode()
	return c.do(ctx, req)
}

// ParseFlightList flattens a successful response body into records.
func ParseFlightList(body []byte) ([]FlightListRecord, error) {
	var root struct {
		Result struct {
			Response struct {
				Data []struct {
					Identification struct {
						ID     *string `json:"id"`
						Number struct {
							Default *string `json:"default"`
						} `json:"number"`
						Callsign *string `json:"callsign"`
					} `json:"identification"`
					Aircraft struct {
						Hex          *string `json:"hex"`
						Registration *string `json:"registration"`
						Model        struct {
							Code *string `json:"code"`
						} `json:"model"`
					} `json:"aircraft"`
					Airport struct {
						Origin *struct {
							Code struct {
								ICAO *string `json:"icao"`
							} `json:"code"`
						} `json:"origin"`
						Destination *struct {
							Code struct {
								ICAO *string `json:"icao"`
							} `json:"code"`
						} `json:"destination"`
					} `json:"airport"`
					Status struct {
						Text *string `json:"text"`
					} `json:"status"`
					Time struct {
						Scheduled struct {
							Departure *int64 `json:"departure"`
							Arrival   *int64 `json:"arrival"`
						} `json:"scheduled"`
						Estimated struct {
							Departure *int64 `json:"departure"`
							Arrival   *int64 `json:"arrival"`
						} `json:"estimated"`
						Real struct {
							Departure *int64 `json:"departure"`
							Arrival   *int64 `json:"arrival"`
						} `json:"real"`
					} `json:"time"`
				} `json:"data"`
			} `json:"response"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &root); err != nil {
		return nil, err
	}
	out := make([]FlightListRecord, 0, len(root.Result.Response.Data))
	for _, e := range root.Result.Response.Data {
		var rec FlightListRecord
		// flight id (hex -> int)
		if e.Identification.ID != nil {
			if n, err := strconv.ParseInt(*e.Identification.ID, 16, 64); err == nil {
				rec.FlightID = &n
			}
		}
		rec.Number = e.Identification.Number.Default
		rec.Callsign = e.Identification.Callsign
		if e.Aircraft.Hex != nil {
			if n, err := strconv.ParseInt(*e.Aircraft.Hex, 16, 64); err == nil {
				rec.ICAO24 = &n
			}
		}
		rec.Registration = e.Aircraft.Registration
		rec.Typecode = e.Aircraft.Model.Code
		if e.Airport.Origin != nil {
			rec.Origin = e.Airport.Origin.Code.ICAO
		}
		if e.Airport.Destination != nil {
			rec.Destination = e.Airport.Destination.Code.ICAO
		}
		rec.Status = e.Status.Text
		// seconds -> ms
		rec.STOD = mul1000(e.Time.Scheduled.Departure)
		rec.ETOD = mul1000(e.Time.Estimated.Departure)
		rec.ATOD = mul1000(e.Time.Real.Departure)
		rec.STOA = mul1000(e.Time.Scheduled.Arrival)
		rec.ETOA = mul1000(e.Time.Estimated.Arrival)
		rec.ATOA = mul1000(e.Time.Real.Arrival)
		out = append(out, rec)
	}
	return out, nil
}

// ---- Airport List ----

type AirportMode string

const (
	AirportArrivals   AirportMode = "arrivals"
	AirportDepartures AirportMode = "departures"
	AirportGround     AirportMode = "ground"
)

type AirportListParams struct {
	Airport    string
	Mode       AirportMode
	Page       int
	Limit      int
	TimestampS *int64
}

// AirportList performs the JSON airport list call.
func (c *Client) AirportList(ctx context.Context, p AirportListParams) (*http.Response, error) {
	if p.Airport == "" {
		return nil, errors.New("airport is required")
	}
	if p.Mode == "" {
		p.Mode = AirportArrivals
	}
	q := url.Values{}
	q.Set("code", p.Airport)
	q.Add("plugin[]", "schedule")
	q.Set("plugin-setting[schedule][mode]", string(p.Mode))
	q.Set("page", strconv.Itoa(zeroDefault(p.Page, 1)))
	q.Set("limit", strconv.Itoa(zeroDefault(p.Limit, 10)))
	if p.TimestampS != nil {
		q.Set("plugin-setting[schedule][timestamp]", strconv.FormatInt(*p.TimestampS, 10))
	} else {
		q.Set("plugin-setting[schedule][timestamp]", strconv.FormatInt(UnixNow(), 10))
	}
	withAuthParams(&q, c.subscriptionKey, c.deviceID)

	req, _ := http.NewRequest("GET", "https://api.flightradar24.com/common/v1/airport.json", nil)
	req.URL.RawQuery = q.Encode()
	return c.do(ctx, req)
}

// ---- Playback ----

type PlaybackParams struct {
	FlightIDHex string // hex string (no 0x)
	TimestampS  *int64 // optional (recommended); nil => now
}

// Playback performs the JSON playback call.
func (c *Client) Playback(ctx context.Context, p PlaybackParams) (*http.Response, error) {
	if p.FlightIDHex == "" {
		return nil, fmt.Errorf("FlightIDHex is required")
	}
	q := url.Values{}
	q.Set("flightId", ToFlightIDHex(p.FlightIDHex))
	if p.TimestampS != nil {
		q.Set("timestamp", strconv.FormatInt(*p.TimestampS, 10))
	} else {
		q.Set("timestamp", strconv.FormatInt(UnixNow(), 10))
	}
	withAuthParams(&q, c.subscriptionKey, c.deviceID)

	req, _ := http.NewRequest("GET", "https://api.flightradar24.com/common/v1/flight-playback.json", nil)
	req.URL.RawQuery = q.Encode()
	return c.do(ctx, req)
}

// Flatten structures for Playback track, mimicking the Python dict.
type PlaybackTrackEMS struct {
	Timestamp   *int64   `json:"timestamp,omitempty"`
	IAS         *float64 `json:"ias,omitempty"`
	TAS         *float64 `json:"tas,omitempty"`
	Mach        *float64 `json:"mach,omitempty"`
	MCP         *float64 `json:"mcp,omitempty"`
	FMS         *float64 `json:"fms,omitempty"`
	Autopilot   *bool    `json:"autopilot,omitempty"`
	OAT         *float64 `json:"oat,omitempty"`
	Track       *float64 `json:"track,omitempty"`
	Roll        *float64 `json:"roll,omitempty"`
	QNH         *float64 `json:"qnh,omitempty"`
	WindDir     *float64 `json:"wind_dir,omitempty"`
	WindSpeed   *float64 `json:"wind_speed,omitempty"`
	Precision   *float64 `json:"precision,omitempty"`
	AltitudeGPS *float64 `json:"altitude_gps,omitempty"`
	Emergency   *bool    `json:"emergency,omitempty"`
	TCAS_ACAS   *bool    `json:"tcas_acas,omitempty"`
	Heading     *float64 `json:"heading,omitempty"`
}

type PlaybackTrack struct {
	Timestamp     int64             `json:"timestamp"`
	Latitude      float64           `json:"latitude"`
	Longitude     float64           `json:"longitude"`
	AltitudeFeet  float64           `json:"altitude"`
	GroundSpeedKt float64           `json:"ground_speed"`
	VerticalFPM   float64           `json:"vertical_speed"`
	Track         float64           `json:"track"`
	SquawkOctal   int64             `json:"squawk"`
	EMS           *PlaybackTrackEMS `json:"ems,omitempty"`
}

// ParsePlayback flattens the playback JSON response into track points.
func ParsePlayback(body []byte) ([]PlaybackTrack, error) {
	// Only decode the fields we need
	var root struct {
		Result struct {
			Response struct {
				Data struct {
					Flight struct {
						Track []struct {
							Timestamp int64   `json:"timestamp"`
							Latitude  float64 `json:"latitude"`
							Longitude float64 `json:"longitude"`
							Altitude  struct {
								Feet float64 `json:"feet"`
							} `json:"altitude"`
							Speed struct {
								Kts float64 `json:"kts"`
							} `json:"speed"`
							VerticalSpeed struct {
								FPM float64 `json:"fpm"`
							} `json:"verticalSpeed"`
							Heading float64 `json:"heading"`
							Squawk  string  `json:"squawk"`
							EMS     *struct {
								TS        *int64   `json:"ts"`
								IAS       *float64 `json:"ias"`
								TAS       *float64 `json:"tas"`
								Mach      *float64 `json:"mach"`
								MCP       *float64 `json:"mcp"`
								FMS       *float64 `json:"fms"`
								Autopilot *bool    `json:"autopilot"`
								OAT       *float64 `json:"oat"`
								TrueTrack *float64 `json:"trueTrack"`
								RollAngle *float64 `json:"rollAngle"`
								QNH       *float64 `json:"qnh"`
								WindDir   *float64 `json:"windDir"`
								WindSpd   *float64 `json:"windSpd"`
								Precision *float64 `json:"precision"`
								AltGPS    *float64 `json:"altGPS"`
								Emergency *bool    `json:"emergencyStatus"`
								TCASACAS  *bool    `json:"tcasAcasDtatus"`
								Heading   *float64 `json:"heading"`
							} `json:"ems"`
						} `json:"track"`
					} `json:"flight"`
				} `json:"data"`
			} `json:"response"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &root); err != nil {
		return nil, err
	}
	out := make([]PlaybackTrack, 0, len(root.Result.Response.Data.Flight.Track))
	for _, pt := range root.Result.Response.Data.Flight.Track {
		var ems *PlaybackTrackEMS
		if pt.EMS != nil {
			ems = &PlaybackTrackEMS{
				Timestamp:   pt.EMS.TS,
				IAS:         pt.EMS.IAS,
				TAS:         pt.EMS.TAS,
				Mach:        pt.EMS.Mach,
				MCP:         pt.EMS.MCP,
				FMS:         pt.EMS.FMS,
				Autopilot:   pt.EMS.Autopilot,
				OAT:         pt.EMS.OAT,
				Track:       pt.EMS.TrueTrack,
				Roll:        pt.EMS.RollAngle,
				QNH:         pt.EMS.QNH,
				WindDir:     pt.EMS.WindDir,
				WindSpeed:   pt.EMS.WindSpd,
				Precision:   pt.EMS.Precision,
				AltitudeGPS: pt.EMS.AltGPS,
				Emergency:   pt.EMS.Emergency,
				TCAS_ACAS:   pt.EMS.TCASACAS,
				Heading:     pt.EMS.Heading,
			}
		}
		// squawk in JSON is octal string
		var squawk int64
		if pt.Squawk != "" {
			if n, err := strconv.ParseInt(pt.Squawk, 8, 64); err == nil {
				squawk = n
			}
		}

		out = append(out, PlaybackTrack{
			Timestamp:     pt.Timestamp,
			Latitude:      pt.Latitude,
			Longitude:     pt.Longitude,
			AltitudeFeet:  pt.Altitude.Feet,
			GroundSpeedKt: pt.Speed.Kts,
			VerticalFPM:   pt.VerticalSpeed.FPM,
			Track:         pt.Heading,
			SquawkOctal:   squawk,
			EMS:           ems,
		})
	}
	return out, nil
}

// ---- Find ----

type FindParams struct {
	Query string
	Limit int
}

func (c *Client) Find(ctx context.Context, p FindParams) (*http.Response, error) {
	if p.Query == "" {
		return nil, errors.New("query is required")
	}
	if p.Limit <= 0 {
		p.Limit = 50
	}
	q := url.Values{}
	q.Set("query", p.Query)
	q.Set("limit", strconv.Itoa(p.Limit))
	withAuthParams(&q, c.subscriptionKey, c.deviceID)
	req, _ := http.NewRequest("GET", "https://www.flightradar24.com/v1/search/web/find", nil)
	req.URL.RawQuery = q.Encode()
	return c.do(ctx, req)
}

// ---- helpers ----

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func zeroDefault(v, d int) int {
	if v == 0 {
		return d
	}
	return v
}

func mul1000(p *int64) *int64 {
	if p == nil {
		return nil
	}
	n := *p * 1000
	return &n
}

func withAuthParams(q *url.Values, subscriptionKey, deviceID string) {
	if subscriptionKey != "" {
		q.Set("token", subscriptionKey)
	} else {
		// emulate Python: include a device param when unauthenticated
		if deviceID == "" {
			deviceID = newDeviceID()
		}
		q.Set("device", deviceID)
	}
}
