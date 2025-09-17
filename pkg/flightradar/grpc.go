package flightradar

import (
	"bufio"
	"context"
	"io"
	"net/http"

	pb "github.com/igolaizola/fr24/pkg/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// BoundingBox mirrors the Python NamedTuple.
type BoundingBox struct {
	South float32
	North float32
	West  float32
	East  float32
}

// LiveFeedParams builds pb.LiveFeedRequest.
type LiveFeedParams struct {
	BoundingBox BoundingBox
	Stats       bool
	Limit       int32
	MaxAge      int32
	Fields      []string
}

func (p LiveFeedParams) toProto() *pb.LiveFeedRequest {
	// default fields similar to Python
	fields := p.Fields
	if len(fields) == 0 {
		fields = []string{"flight", "reg", "route", "type"}
	}
	return &pb.LiveFeedRequest{
		Bounds: &pb.LocationBoundaries{
			North: p.BoundingBox.North,
			South: p.BoundingBox.South,
			West:  p.BoundingBox.West,
			East:  p.BoundingBox.East,
		},
		Settings: &pb.VisibilitySettings{
			SourcesList:    []pb.DataSource{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
			ServicesList:   []pb.Service{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
			TrafficType:    pb.TrafficType_ALL,
			OnlyRestricted: protoBool(false),
		},
		FieldMask:       &fieldmaskpb.FieldMask{Paths: fields},
		HighlightMode:   false,
		Stats:           protoBool(p.Stats),
		Limit:           protoInt32(p.Limit, 1500),
		Maxage:          protoInt32(p.MaxAge, 14400),
		RestrictionMode: pb.RestrictionVisibility_NOT_VISIBLE.Enum(),
	}
}

func protoBool(b bool) *bool { return &b }
func protoInt32(v int32, def int32) *int32 {
	if v == 0 {
		v = def
	}
	return &v
}

// GrpcLiveFeed sends the LiveFeed request and returns raw HTTP response.
func (c *Client) GrpcLiveFeed(ctx context.Context, p LiveFeedParams) (*http.Response, error) {
	reqHeaders := defaultGRPCHeaders(c.deviceID, c.grpcBearer())
	req, err := constructGRPCRequest("LiveFeed", p.toProto(), reqHeaders)
	if err != nil {
		return nil, err
	}
	return c.do(ctx, req)
}

// Playback (gRPC) request. Mirrors LiveFeedPlaybackParams in Python with
// timestamp and prefetch computed on caller side.
type LiveFeedPlaybackParams struct {
	LiveFeed  LiveFeedParams
	Timestamp int32 // seconds (now - duration) if 0
	Duration  int32 // default 7
	HFreq     *int32
}

func (p LiveFeedPlaybackParams) toProto() *pb.PlaybackRequest {
	ts := p.Timestamp
	if ts == 0 {
		ts = int32(UnixNow()) - max32(p.Duration, 7)
	}
	req := &pb.PlaybackRequest{
		LiveFeedRequest: p.LiveFeed.toProto(),
		Timestamp:       ts,
		Prefetch:        ts + max32(p.Duration, 7),
	}
	if p.HFreq != nil {
		req.Hfreq = p.HFreq
	}
	return req
}

func max32(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

func (c *Client) GrpcPlayback(ctx context.Context, p LiveFeedPlaybackParams) (*http.Response, error) {
	reqHeaders := defaultGRPCHeaders(c.deviceID, c.grpcBearer())
	req, err := constructGRPCRequest("Playback", p.toProto(), reqHeaders)
	if err != nil {
		return nil, err
	}
	return c.do(ctx, req)
}

// Nearest flights
type NearestFlightsParams struct {
	Lat, Lon      float32
	Radius, Limit int32
}

func (p NearestFlightsParams) toProto() *pb.NearestFlightsRequest {
	r := p.Radius
	if r == 0 {
		r = 10000
	}
	l := p.Limit
	if l == 0 {
		l = 1500
	}
	return &pb.NearestFlightsRequest{
		Location: &pb.Geolocation{Lat: p.Lat, Lon: p.Lon}, Radius: uint32(r), Limit: uint32(l),
	}
}
func (c *Client) GrpcNearestFlights(ctx context.Context, p NearestFlightsParams) (*http.Response, error) {
	reqHeaders := defaultGRPCHeaders(c.deviceID, c.grpcBearer())
	req, err := constructGRPCRequest("NearestFlights", p.toProto(), reqHeaders)
	if err != nil {
		return nil, err
	}
	return c.do(ctx, req)
}

// Live flights status
type LiveFlightsStatusParams struct{ FlightIDs []uint32 }

func (p LiveFlightsStatusParams) toProto() *pb.LiveFlightsStatusRequest {
	return &pb.LiveFlightsStatusRequest{FlightIdsList: p.FlightIDs}
}
func (c *Client) GrpcLiveFlightsStatus(ctx context.Context, p LiveFlightsStatusParams) (*http.Response, error) {
	reqHeaders := defaultGRPCHeaders(c.deviceID, c.grpcBearer())
	req, err := constructGRPCRequest("LiveFlightsStatus", p.toProto(), reqHeaders)
	if err != nil {
		return nil, err
	}
	return c.do(ctx, req)
}

// Top flights
type TopFlightsParams struct{ Limit int32 }

func (p TopFlightsParams) toProto() *pb.TopFlightsRequest {
	if p.Limit <= 0 || p.Limit > 10 {
		p.Limit = 10
	}
	return &pb.TopFlightsRequest{Limit: uint32(p.Limit)}
}
func (c *Client) GrpcTopFlights(ctx context.Context, p TopFlightsParams) (*http.Response, error) {
	reqHeaders := defaultGRPCHeaders(c.deviceID, c.grpcBearer())
	req, err := constructGRPCRequest("TopFlights", p.toProto(), reqHeaders)
	if err != nil {
		return nil, err
	}
	return c.do(ctx, req)
}

// Live trail
func (c *Client) GrpcLiveTrail(ctx context.Context, flightID uint32) (*http.Response, error) {
	reqHeaders := defaultGRPCHeaders(c.deviceID, c.grpcBearer())
	req, err := constructGRPCRequest("LiveTrail", &pb.LiveTrailRequest{FlightId: flightID}, reqHeaders)
	if err != nil {
		return nil, err
	}
	return c.do(ctx, req)
}

// Historic trail
func (c *Client) GrpcHistoricTrail(ctx context.Context, flightID uint32) (*http.Response, error) {
	reqHeaders := defaultGRPCHeaders(c.deviceID, c.grpcBearer())
	req, err := constructGRPCRequest("HistoricTrail", &pb.HistoricTrailRequest{FlightId: flightID}, reqHeaders)
	if err != nil {
		return nil, err
	}
	return c.do(ctx, req)
}

// Flight details (live)
type FlightDetailsParams struct {
	FlightID    uint32
	Restriction pb.RestrictionVisibility
	Verbose     bool
}

func (p FlightDetailsParams) toProto() *pb.FlightDetailsRequest {
	return &pb.FlightDetailsRequest{FlightId: p.FlightID, RestrictionMode: p.Restriction, Verbose: p.Verbose}
}
func (c *Client) GrpcFlightDetails(ctx context.Context, p FlightDetailsParams) (*http.Response, error) {
	reqHeaders := defaultGRPCHeaders(c.deviceID, c.grpcBearer())
	req, err := constructGRPCRequest("FlightDetails", p.toProto(), reqHeaders)
	if err != nil {
		return nil, err
	}
	return c.do(ctx, req)
}

// Playback flight (historic)
type PlaybackFlightParams struct {
	FlightID  uint32
	Timestamp uint64
}

func (p PlaybackFlightParams) toProto() *pb.PlaybackFlightRequest {
	return &pb.PlaybackFlightRequest{FlightId: p.FlightID, Timestamp: p.Timestamp}
}
func (c *Client) GrpcPlaybackFlight(ctx context.Context, p PlaybackFlightParams) (*http.Response, error) {
	reqHeaders := defaultGRPCHeaders(c.deviceID, c.grpcBearer())
	req, err := constructGRPCRequest("PlaybackFlight", p.toProto(), reqHeaders)
	if err != nil {
		return nil, err
	}
	return c.do(ctx, req)
}

// Follow flight streaming: returns a channel of raw frames and a cancel function.
func (c *Client) GrpcFollowFlightStream(ctx context.Context, flightID uint32, restriction pb.RestrictionVisibility) (<-chan []byte, func(), error) {
	reqHeaders := defaultGRPCHeaders(c.deviceID, c.grpcBearer())
	req, err := constructGRPCRequest("FollowFlight", &pb.FollowFlightRequest{FlightId: flightID, RestrictionMode: restriction}, reqHeaders)
	if err != nil {
		return nil, nil, err
	}
	// Force no overall timeout to keep stream open unless caller cancels.
	hc := *c.http
	hc.Timeout = 0
	resp, err := hc.Do(req.WithContext(ctx))
	if err != nil {
		return nil, nil, err
	}
	ch := make(chan []byte, 8)
	done := make(chan struct{})
    go func() {
        defer close(ch)
        defer func() { _ = resp.Body.Close() }()
		br := bufio.NewReader(resp.Body)
		for {
			// Read until EOF; deliver raw frames (as chunks may align with frames).
			// We read by frame prefix: 1 + 4 bytes, then payload.
			header := make([]byte, 5)
			if _, err := io.ReadFull(br, header); err != nil {
				return
			}
			n := int(header[1])<<24 | int(header[2])<<16 | int(header[3])<<8 | int(header[4])
			payload := make([]byte, n)
			if _, err := io.ReadFull(br, payload); err != nil {
				return
			}
			// Reassemble the full frame to match parseData expectations
			frame := append(header[:5:5], payload...)
			select {
			case ch <- frame:
			case <-done:
				return
			}
		}
	}()
    cancel := func() { close(done); _ = resp.Body.Close() }
    return ch, cancel, nil
}

// Helpers to extract token for grpc headers.
func (c *Client) grpcBearer() string {
	// In a fuller implementation, this would read c.auth.userData.accessToken
	// For now, we support setting it via WithSubscriptionKey + WithAuthToken
	return c.authToken
}

// Optional: set bearer for grpc auth flows.
func (c *Client) WithAuthToken(token string) *Client { c.authToken = token; return c }
