package flightradar

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
)

type ServiceFactory struct{ C *Client }

func NewServices(c *Client) *ServiceFactory { return &ServiceFactory{C: c} }

// ---------- Flight List (JSON) ----------
type FlightListService struct{ f *ServiceFactory }

func (s *ServiceFactory) FlightList() *FlightListService { return &FlightListService{s} }

type FlightListResult struct {
	Request  FlightListParams
	Response *http.Response
}

func (svc *FlightListService) Fetch(ctx context.Context, p FlightListParams) (*FlightListResult, error) {
	resp, err := svc.f.C.FlightList(ctx, p)
	if err != nil {
		return nil, err
	}
	return &FlightListResult{Request: p, Response: resp}, nil
}
func (r *FlightListResult) Records() ([]FlightListRecord, error) {
    defer func() { _ = r.Response.Body.Close() }()
	b, _ := io.ReadAll(r.Response.Body)
	return ParseFlightList(b)
}
func (r *FlightListResult) WriteCSV(path string) error {
	recs, err := r.Records()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dirOf(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
    defer func() { _ = f.Close() }()
	return WriteCSV(f, recs)
}

// ---------- Playback (JSON) ----------
type PlaybackService struct{ f *ServiceFactory }

func (s *ServiceFactory) Playback() *PlaybackService { return &PlaybackService{s} }

type PlaybackResult struct {
	Request  PlaybackParams
	Response *http.Response
}

func (svc *PlaybackService) Fetch(ctx context.Context, p PlaybackParams) (*PlaybackResult, error) {
	resp, err := svc.f.C.Playback(ctx, p)
	if err != nil {
		return nil, err
	}
	return &PlaybackResult{Request: p, Response: resp}, nil
}
func (r *PlaybackResult) Records() ([]PlaybackTrack, error) {
    defer func() { _ = r.Response.Body.Close() }()
	b, _ := io.ReadAll(r.Response.Body)
	return ParsePlayback(b)
}
func (r *PlaybackResult) WriteCSV(path string) error {
	recs, err := r.Records()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dirOf(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
    defer func() { _ = f.Close() }()
	return WriteCSV(f, recs)
}

// ---------- Live Feed (gRPC) ----------
type LiveFeedService struct{ f *ServiceFactory }

func (s *ServiceFactory) LiveFeed() *LiveFeedService { return &LiveFeedService{s} }

type LiveFeedResult struct {
	Request  LiveFeedParams
	Response *http.Response
}

func (svc *LiveFeedService) Fetch(ctx context.Context, p LiveFeedParams) (*LiveFeedResult, error) {
	resp, err := svc.f.C.GrpcLiveFeed(ctx, p)
	if err != nil {
		return nil, err
	}
	return &LiveFeedResult{Request: p, Response: resp}, nil
}
func (r *LiveFeedResult) Records() ([]LiveFeedFlightRecord, error) {
    defer func() { _ = r.Response.Body.Close() }()
	b, _ := io.ReadAll(r.Response.Body)
	msg, err := ParseLiveFeedGRPC(b)
	if err != nil {
		return nil, err
	}
	out := make([]LiveFeedFlightRecord, 0, len(msg.GetFlightsList()))
	for _, f := range msg.GetFlightsList() {
		out = append(out, LiveFeedFlightToRecord(f))
	}
	return out, nil
}

// ---------- Nearest Flights (gRPC) ----------
type NearestFlightsService struct{ f *ServiceFactory }

func (s *ServiceFactory) NearestFlights() *NearestFlightsService { return &NearestFlightsService{s} }

type NearestFlightsResult struct {
	Request  NearestFlightsParams
	Response *http.Response
}

func (svc *NearestFlightsService) Fetch(ctx context.Context, p NearestFlightsParams) (*NearestFlightsResult, error) {
	resp, err := svc.f.C.GrpcNearestFlights(ctx, p)
	if err != nil {
		return nil, err
	}
	return &NearestFlightsResult{Request: p, Response: resp}, nil
}
func (r *NearestFlightsResult) Records() ([]NearbyFlightRecord, error) {
    defer func() { _ = r.Response.Body.Close() }()
	b, _ := io.ReadAll(r.Response.Body)
	msg, err := ParseNearestFlightsGRPC(b)
	if err != nil {
		return nil, err
	}
	return NearbyToRecords(msg), nil
}

// ---------- Live Flights Status (gRPC) ----------
type LiveFlightsStatusService struct{ f *ServiceFactory }

func (s *ServiceFactory) LiveFlightsStatus() *LiveFlightsStatusService {
	return &LiveFlightsStatusService{s}
}

type LiveFlightsStatusResult struct {
	Request  LiveFlightsStatusParams
	Response *http.Response
}

func (svc *LiveFlightsStatusService) Fetch(ctx context.Context, p LiveFlightsStatusParams) (*LiveFlightsStatusResult, error) {
	resp, err := svc.f.C.GrpcLiveFlightsStatus(ctx, p)
	if err != nil {
		return nil, err
	}
	return &LiveFlightsStatusResult{Request: p, Response: resp}, nil
}
func (r *LiveFlightsStatusResult) Records() ([]LiveFlightsStatusRecord, error) {
    defer func() { _ = r.Response.Body.Close() }()
	b, _ := io.ReadAll(r.Response.Body)
	msg, err := ParseLiveFlightsStatusGRPC(b)
	if err != nil {
		return nil, err
	}
	return LiveFlightsStatusToRecords(msg), nil
}

// ---------- Top Flights (gRPC) ----------
type TopFlightsService struct{ f *ServiceFactory }

func (s *ServiceFactory) TopFlights() *TopFlightsService { return &TopFlightsService{s} }

type TopFlightsResult struct {
	Request  TopFlightsParams
	Response *http.Response
}

func (svc *TopFlightsService) Fetch(ctx context.Context, p TopFlightsParams) (*TopFlightsResult, error) {
	resp, err := svc.f.C.GrpcTopFlights(ctx, p)
	if err != nil {
		return nil, err
	}
	return &TopFlightsResult{Request: p, Response: resp}, nil
}
func (r *TopFlightsResult) Records() ([]TopFlightRecord, error) {
    defer func() { _ = r.Response.Body.Close() }()
	b, _ := io.ReadAll(r.Response.Body)
	msg, err := ParseTopFlightsGRPC(b)
	if err != nil {
		return nil, err
	}
	out := make([]TopFlightRecord, 0, len(msg.GetScoreboardList()))
	for _, ff := range msg.GetScoreboardList() {
		out = append(out, TopFlightToRecord(ff))
	}
	return out, nil
}

// ---------- Flight Details (gRPC) ----------
type FlightDetailsService struct{ f *ServiceFactory }

func (s *ServiceFactory) FlightDetails() *FlightDetailsService { return &FlightDetailsService{s} }

type FlightDetailsResult struct {
	Request  FlightDetailsParams
	Response *http.Response
}

func (svc *FlightDetailsService) Fetch(ctx context.Context, p FlightDetailsParams) (*FlightDetailsResult, error) {
	resp, err := svc.f.C.GrpcFlightDetails(ctx, p)
	if err != nil {
		return nil, err
	}
	return &FlightDetailsResult{Request: p, Response: resp}, nil
}
func (r *FlightDetailsResult) Record() (FlightDetailsRecord, error) {
    defer func() { _ = r.Response.Body.Close() }()
	b, _ := io.ReadAll(r.Response.Body)
	msg, err := ParseFlightDetailsGRPC(b)
	if err != nil {
		return FlightDetailsRecord{}, err
	}
	return FlightDetailsToRecord(msg), nil
}

// ---------- Playback Flight (gRPC) ----------
type PlaybackFlightService struct{ f *ServiceFactory }

func (s *ServiceFactory) PlaybackFlight() *PlaybackFlightService { return &PlaybackFlightService{s} }

type PlaybackFlightResult struct {
	Request  PlaybackFlightParams
	Response *http.Response
}

func (svc *PlaybackFlightService) Fetch(ctx context.Context, p PlaybackFlightParams) (*PlaybackFlightResult, error) {
	resp, err := svc.f.C.GrpcPlaybackFlight(ctx, p)
	if err != nil {
		return nil, err
	}
	return &PlaybackFlightResult{Request: p, Response: resp}, nil
}
func (r *PlaybackFlightResult) Record() (PlaybackFlightRecord, error) {
    defer func() { _ = r.Response.Body.Close() }()
	b, _ := io.ReadAll(r.Response.Body)
	msg, err := ParsePlaybackFlightGRPC(b)
	if err != nil {
		return PlaybackFlightRecord{}, err
	}
	return PlaybackFlightToRecord(msg), nil
}

// ---------- Find (JSON) ----------
type FindService struct{ f *ServiceFactory }

func (s *ServiceFactory) Find() *FindService { return &FindService{s} }

type FindResult struct {
	Request  FindParams
	Response *http.Response
}

func (svc *FindService) Fetch(ctx context.Context, p FindParams) (*FindResult, error) {
	resp, err := svc.f.C.Find(ctx, p)
	if err != nil {
		return nil, err
	}
	return &FindResult{Request: p, Response: resp}, nil
}
func (r *FindResult) JSON(v any) error {
    defer func() { _ = r.Response.Body.Close() }()
	return json.NewDecoder(r.Response.Body).Decode(v)
}

// ---------- Airport list (JSON) ----------
type AirportListService struct{ f *ServiceFactory }

func (s *ServiceFactory) AirportList() *AirportListService { return &AirportListService{s} }

type AirportListResult struct {
	Request  AirportListParams
	Response *http.Response
}

func (svc *AirportListService) Fetch(ctx context.Context, p AirportListParams) (*AirportListResult, error) {
	resp, err := svc.f.C.AirportList(ctx, p)
	if err != nil {
		return nil, err
	}
	return &AirportListResult{Request: p, Response: resp}, nil
}
func (r *AirportListResult) JSON(v any) error {
    defer func() { _ = r.Response.Body.Close() }()
	return json.NewDecoder(r.Response.Body).Decode(v)
}

func dirOf(p string) string {
	i := len(p) - 1
	for i >= 0 && p[i] != '/' {
		i--
	}
	if i > 0 {
		return p[:i]
	}
	return "."
}
