package flightradar

import pb "github.com/igolaizola/fr24/pkg/proto"

type LiveFlightsStatusRecord struct {
	FlightID  uint32    `csv:"flight_id" json:"flight_id"`
	Latitude  float32   `csv:"latitude" json:"latitude"`
	Longitude float32   `csv:"longitude" json:"longitude"`
	Status    pb.Status `csv:"status" json:"status"`
	Squawk    uint32    `csv:"squawk" json:"squawk"`
}

type NearbyFlightRecord struct {
	DistanceM uint32               `csv:"distance" json:"distance"`
	Live      LiveFeedFlightRecord `csv:"-" json:"live"`
}

type FlightDetailsRecord struct {
	// aircraft info
	ICAOAddress uint32 `csv:"icao_address" json:"icao_address"`
	Reg         string `csv:"reg" json:"reg"`
	Typecode    string `csv:"typecode" json:"typecode"`
	// schedule info
	FlightNumber       string `csv:"flight_number" json:"flight_number"`
	OriginID           uint32 `csv:"origin_id" json:"origin_id"`
	DestinationID      uint32 `csv:"destination_id" json:"destination_id"`
	DivertedID         uint32 `csv:"diverted_id" json:"diverted_id"`
	ScheduledDeparture uint32 `csv:"scheduled_departure" json:"scheduled_departure"`
	ScheduledArrival   uint32 `csv:"scheduled_arrival" json:"scheduled_arrival"`
	ActualDeparture    uint32 `csv:"actual_departure" json:"actual_departure"`
	ActualArrival      uint32 `csv:"actual_arrival" json:"actual_arrival"`
	// flight info
	TimestampMS   uint64  `csv:"timestamp_ms" json:"timestamp_ms"`
	FlightID      uint32  `csv:"flightid" json:"flightid"`
	Latitude      float32 `csv:"latitude" json:"latitude"`
	Longitude     float32 `csv:"longitude" json:"longitude"`
	Track         int32   `csv:"track" json:"track"`
	Altitude      int32   `csv:"altitude" json:"altitude"`
	GroundSpeed   int32   `csv:"ground_speed" json:"ground_speed"`
	VerticalSpeed int32   `csv:"vertical_speed" json:"vertical_speed"`
	OnGround      bool    `csv:"on_ground" json:"on_ground"`
	Callsign      string  `csv:"callsign" json:"callsign"`
	Squawk        int32   `csv:"squawk" json:"squawk"`
}

type PlaybackFlightRecord struct {
	// basic info
	ICAOAddress        uint32  `csv:"icao_address" json:"icao_address"`
	Reg                string  `csv:"reg" json:"reg"`
	Typecode           string  `csv:"typecode" json:"typecode"`
	FlightNumber       string  `csv:"flight_number" json:"flight_number"`
	OriginID           uint32  `csv:"origin_id" json:"origin_id"`
	DestinationID      uint32  `csv:"destination_id" json:"destination_id"`
	DivertedID         uint32  `csv:"diverted_id" json:"diverted_id"`
	ScheduledDeparture uint32  `csv:"scheduled_departure" json:"scheduled_departure"`
	ScheduledArrival   uint32  `csv:"scheduled_arrival" json:"scheduled_arrival"`
	ActualDeparture    uint32  `csv:"actual_departure" json:"actual_departure"`
	ActualArrival      uint32  `csv:"actual_arrival" json:"actual_arrival"`
	TimestampMS        uint64  `csv:"timestamp_ms" json:"timestamp_ms"`
	FlightID           uint32  `csv:"flightid" json:"flightid"`
	Latitude           float32 `csv:"latitude" json:"latitude"`
	Longitude          float32 `csv:"longitude" json:"longitude"`
	Track              int32   `csv:"track" json:"track"`
	Altitude           int32   `csv:"altitude" json:"altitude"`
	GroundSpeed        int32   `csv:"ground_speed" json:"ground_speed"`
	VerticalSpeed      int32   `csv:"vertical_speed" json:"vertical_speed"`
	OnGround           bool    `csv:"on_ground" json:"on_ground"`
	Callsign           string  `csv:"callsign" json:"callsign"`
	Squawk             int32   `csv:"squawk" json:"squawk"`
}
