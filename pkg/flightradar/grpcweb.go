package flightradar

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net/http"

	pb "github.com/igolaizola/fr24/pkg/proto"
	"google.golang.org/protobuf/proto"
)

// encodeMessage builds a gRPC-web framed message: 1-byte flag + 4-byte len (BE) + payload.
func encodeMessage(m proto.Message) ([]byte, error) {
	body, err := proto.Marshal(m)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(make([]byte, 0, 1+4+len(body)))
	buf.WriteByte(0) // uncompressed
	var len4 [4]byte
	binary.BigEndian.PutUint32(len4[:], uint32(len(body)))
	buf.Write(len4[:])
	buf.Write(body)
	return buf.Bytes(), nil
}

// parseData parses a single DATA frame payload into the target message.
// It mirrors the Python parse_data behavior: erroring on compressed frames and
// decoding trailers as gRPC errors.
func parseData(data []byte, into proto.Message) error {
	if len(data) == 0 {
		return &GrpcError{Message: "empty DATA frame", Raw: data}
	}
	flag := data[0]
	if flag == 1 {
		return &GrpcError{Message: "message is compressed, not implemented", Raw: data}
	}
	if flag != 0 {
		// trailers frame
		return parseTrailers(data)
	}
	if len(data) < 5 {
		return &GrpcError{Message: "short frame", Raw: data}
	}
	n := binary.BigEndian.Uint32(data[1:5])
	if n == 0 {
		return &GrpcError{Message: "empty message payload", Raw: data}
	}
	msg := data[5 : 5+int(n)]
	if err := proto.Unmarshal(msg, into); err != nil {
		return &ProtoParseError{Err: fmt.Errorf("failed to parse message: %w", err), Raw: data}
	}
	return nil
}

// constructGRPCRequest builds an HTTP request for the FR24 gRPC-web endpoint.
func constructGRPCRequest(method string, message proto.Message, headers http.Header) (*http.Request, error) {
	body, err := encodeMessage(message)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", "https://data-feed.flightradar24.com/fr24.feed.api.v1.Feed/"+method, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	for k, vs := range headers {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	// Required gRPC-web headers
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/grpc-web+proto")
	}
	if req.Header.Get("X-User-Agent") == "" {
		req.Header.Set("X-User-Agent", "grpc-web-javascript/0.1")
	}
	if req.Header.Get("X-Grpc-Web") == "" {
		req.Header.Set("X-Grpc-Web", "1")
	}
	return req, nil
}

// parseTrailers extracts grpc-status and grpc-message from a trailer frame.
func parseTrailers(data []byte) error {
	// Skip 5-byte header, remainder contains trailers as ASCII lines
	trailers := data[5:]
	lines := bytes.Split(bytes.TrimSpace(trailers), []byte{'\n'})
	ge := &GrpcError{Message: "gRPC errored", Raw: data}
	for _, ln := range lines {
		if bytes.HasPrefix(ln, []byte("grpc-status:")) {
			ge.Status = string(bytes.TrimSpace(ln[len("grpc-status:"):]))
		} else if bytes.HasPrefix(ln, []byte("grpc-message:")) {
			ge.StatusMessage = string(bytes.TrimSpace(ln[len("grpc-message:"):]))
		} else if bytes.HasPrefix(ln, []byte("grpc-status-details-bin:")) {
			ge.StatusDetails = ln[len("grpc-status-details-bin:"):]
		}
	}
	return ge
}

// GrpcError mirrors Python's GrpcError with minimal fields.
type GrpcError struct {
	Message       string
	Raw           []byte
	Status        string
	StatusMessage string
	StatusDetails []byte
}

func (e *GrpcError) Error() string {
	if e == nil {
		return ""
	}
	if e.Status != "" || e.StatusMessage != "" {
		return fmt.Sprintf("%s: status=%s message=%s", e.Message, e.Status, e.StatusMessage)
	}
	return e.Message
}

type ProtoParseError struct {
	Err error
	Raw []byte
}

func (e *ProtoParseError) Error() string {
	if e == nil {
		return ""
	}
	return e.Err.Error()
}

// Helpers to decode into concrete response types.
func parseLiveFeedResponse(data []byte) (*pb.LiveFeedResponse, error) {
	var out pb.LiveFeedResponse
	return &out, parseData(data, &out)
}

func parsePlaybackResponse(data []byte) (*pb.PlaybackResponse, error) {
	var out pb.PlaybackResponse
	return &out, parseData(data, &out)
}

func parseNearestFlightsResponse(data []byte) (*pb.NearestFlightsResponse, error) {
    var out pb.NearestFlightsResponse
    if err := parseData(data, &out); err != nil {
        if ge, ok := err.(*GrpcError); ok {
            // Some deployments occasionally return a zero-length DATA frame
            // for NearestFlights when there are no nearby results. Treat this
            // as an empty response instead of an error to align with expected
            // semantics (empty list of flights).
            if ge.Message == "empty message payload" || ge.Message == "empty DATA frame" {
                return &out, nil
            }
        }
        return nil, err
    }
    return &out, nil
}

func parseLiveFlightsStatusResponse(data []byte) (*pb.LiveFlightsStatusResponse, error) {
	var out pb.LiveFlightsStatusResponse
	return &out, parseData(data, &out)
}

func parseTopFlightsResponse(data []byte) (*pb.TopFlightsResponse, error) {
	var out pb.TopFlightsResponse
	return &out, parseData(data, &out)
}

func parseFlightDetailsResponse(data []byte) (*pb.FlightDetailsResponse, error) {
	var out pb.FlightDetailsResponse
	return &out, parseData(data, &out)
}

func parsePlaybackFlightResponse(data []byte) (*pb.PlaybackFlightResponse, error) {
	var out pb.PlaybackFlightResponse
	return &out, parseData(data, &out)
}

// util
var ErrUnexpectedFrame = errors.New("unexpected gRPC-web frame")

// Exported parse helpers for consumers.
func ParseLiveFeedGRPC(data []byte) (*pb.LiveFeedResponse, error) { return parseLiveFeedResponse(data) }
func ParsePlaybackGRPC(data []byte) (*pb.PlaybackResponse, error) { return parsePlaybackResponse(data) }
func ParseNearestFlightsGRPC(data []byte) (*pb.NearestFlightsResponse, error) {
	return parseNearestFlightsResponse(data)
}
func ParseLiveFlightsStatusGRPC(data []byte) (*pb.LiveFlightsStatusResponse, error) {
	return parseLiveFlightsStatusResponse(data)
}
func ParseTopFlightsGRPC(data []byte) (*pb.TopFlightsResponse, error) {
	return parseTopFlightsResponse(data)
}
func ParseFlightDetailsGRPC(data []byte) (*pb.FlightDetailsResponse, error) {
	return parseFlightDetailsResponse(data)
}
func ParsePlaybackFlightGRPC(data []byte) (*pb.PlaybackFlightResponse, error) {
	return parsePlaybackFlightResponse(data)
}
