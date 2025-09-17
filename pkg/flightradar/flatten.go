package flightradar

import pb "github.com/igolaizola/fr24/pkg/proto"

// LiveFeedFlightRecord flattens pb.Flight into a convenient struct.
type LiveFeedFlightRecord struct {
	TimestampMS   uint64        `csv:"timestamp" json:"timestamp"`
	FlightID      uint32        `csv:"flightid" json:"flightid"`
	Latitude      float32       `csv:"latitude" json:"latitude"`
	Longitude     float32       `csv:"longitude" json:"longitude"`
	Track         int32         `csv:"track" json:"track"`
	Altitude      int32         `csv:"altitude" json:"altitude"`
	GroundSpeed   int32         `csv:"ground_speed" json:"ground_speed"`
	OnGround      bool          `csv:"on_ground" json:"on_ground"`
	Callsign      string        `csv:"callsign" json:"callsign"`
	Source        pb.DataSource `csv:"source" json:"source"`
	Registration  string        `csv:"registration" json:"registration"`
	Origin        string        `csv:"origin" json:"origin"`
	Destination   string        `csv:"destination" json:"destination"`
	Typecode      string        `csv:"typecode" json:"typecode"`
	ETA           uint32        `csv:"eta" json:"eta"`
	Squawk        int32         `csv:"squawk" json:"squawk"`
	VerticalSpeed int32         `csv:"vertical_speed" json:"vertical_speed"`
}

func LiveFeedFlightToRecord(f *pb.Flight) LiveFeedFlightRecord {
	var reg, orig, dest, tcode string
	if ei := f.GetExtraInfo(); ei != nil {
		reg = ei.GetReg()
		if r := ei.GetRoute(); r != nil {
			orig, dest = r.GetFrom(), r.GetTo()
		}
		tcode = ei.GetType()
	}
	return LiveFeedFlightRecord{
		TimestampMS:   f.GetTimestampMs(),
		FlightID:      uint32(f.GetFlightid()),
		Latitude:      f.GetLat(),
		Longitude:     f.GetLon(),
		Track:         f.GetTrack(),
		Altitude:      f.GetAlt(),
		GroundSpeed:   f.GetSpeed(),
		OnGround:      f.GetOnGround(),
		Callsign:      f.GetCallsign(),
		Source:        f.GetSource(),
		Registration:  reg,
		Origin:        orig,
		Destination:   dest,
		Typecode:      tcode,
		ETA:           uint32(f.GetExtraInfo().GetSchedule().GetEta()),
		Squawk:        f.GetExtraInfo().GetSquawk(),
		VerticalSpeed: f.GetExtraInfo().GetVspeed(),
	}
}

// NearestFlights flatteners
func NearbyToRecords(resp *pb.NearestFlightsResponse) []NearbyFlightRecord {
	out := make([]NearbyFlightRecord, 0, len(resp.GetFlightsList()))
	for _, nf := range resp.GetFlightsList() {
		rec := LiveFeedFlightToRecord(nf.GetFlight())
		out = append(out, NearbyFlightRecord{DistanceM: nf.GetDistance(), Live: rec})
	}
	return out
}

// Live flights status flattener
func LiveFlightsStatusToRecords(resp *pb.LiveFlightsStatusResponse) []LiveFlightsStatusRecord {
	out := make([]LiveFlightsStatusRecord, 0, len(resp.GetFlightsMap()))
	for _, st := range resp.GetFlightsMap() {
		d := st.GetData()
		out = append(out, LiveFlightsStatusRecord{
			FlightID:  st.GetFlightId(),
			Latitude:  d.GetLat(),
			Longitude: d.GetLon(),
			Status:    d.GetStatus(),
			Squawk:    d.GetSquawk(),
		})
	}
	return out
}

// Flight details flattener
func FlightDetailsToRecord(resp *pb.FlightDetailsResponse) FlightDetailsRecord {
	ai := resp.GetAircraftInfo()
	si := resp.GetScheduleInfo()
	fi := resp.GetFlightInfo()
	return FlightDetailsRecord{
		ICAOAddress:        ai.GetIcaoAddress(),
		Reg:                ai.GetReg(),
		Typecode:           ai.GetType(),
		FlightNumber:       si.GetFlightNumber(),
		OriginID:           si.GetOriginId(),
		DestinationID:      si.GetDestinationId(),
		DivertedID:         si.GetDivertedToId(),
		ScheduledDeparture: si.GetScheduledDeparture(),
		ScheduledArrival:   si.GetScheduledArrival(),
		ActualDeparture:    si.GetActualDeparture(),
		ActualArrival:      si.GetActualArrival(),
		TimestampMS:        fi.GetTimestampMs(),
		FlightID:           uint32(fi.GetFlightid()),
		Latitude:           fi.GetLat(),
		Longitude:          fi.GetLon(),
		Track:              fi.GetTrack(),
		Altitude:           fi.GetAlt(),
		GroundSpeed:        fi.GetSpeed(),
		VerticalSpeed:      fi.GetVspeed(),
		OnGround:           fi.GetOnGround(),
		Callsign:           fi.GetCallsign(),
		Squawk:             fi.GetSquawk(),
	}
}

// Playback flight flattener
func PlaybackFlightToRecord(resp *pb.PlaybackFlightResponse) PlaybackFlightRecord {
	ai := resp.GetAircraftInfo()
	si := resp.GetScheduleInfo()
	fi := resp.GetFlightInfo()
	return PlaybackFlightRecord{
		ICAOAddress: ai.GetIcaoAddress(), Reg: ai.GetReg(), Typecode: ai.GetType(),
		FlightNumber: si.GetFlightNumber(), OriginID: si.GetOriginId(), DestinationID: si.GetDestinationId(),
		DivertedID: si.GetDivertedToId(), ScheduledDeparture: si.GetScheduledDeparture(),
		ScheduledArrival: si.GetScheduledArrival(), ActualDeparture: si.GetActualDeparture(), ActualArrival: si.GetActualArrival(),
		TimestampMS: fi.GetTimestampMs(), FlightID: uint32(fi.GetFlightid()), Latitude: fi.GetLat(), Longitude: fi.GetLon(),
		Track: fi.GetTrack(), Altitude: fi.GetAlt(), GroundSpeed: fi.GetSpeed(), VerticalSpeed: fi.GetVspeed(),
		OnGround: fi.GetOnGround(), Callsign: fi.GetCallsign(), Squawk: fi.GetSquawk(),
	}
}

// TopFlightRecord mirrors Python's top flights dict flattener.
type TopFlightRecord struct {
	FlightID     uint32 `json:"flight_id"`
	LiveClicks   uint32 `json:"live_clicks"`
	TotalClicks  uint32 `json:"total_clicks"`
	FlightNumber string `json:"flight_number"`
	Callsign     string `json:"callsign"`
	Squawk       uint32 `json:"squawk"`
	FromIATA     string `json:"from_iata"`
	FromCity     string `json:"from_city"`
	ToIATA       string `json:"to_iata"`
	ToCity       string `json:"to_city"`
	Type         string `json:"type"`
	FullDesc     string `json:"full_description"`
}

func TopFlightToRecord(ff *pb.FollowedFlight) TopFlightRecord {
	return TopFlightRecord{
		FlightID:     ff.GetFlightId(),
		LiveClicks:   ff.GetLiveClicks(),
		TotalClicks:  ff.GetTotalClicks(),
		FlightNumber: ff.GetFlightNumber(),
		Callsign:     ff.GetCallsign(),
		Squawk:       ff.GetSquawk(),
		FromIATA:     ff.GetFromIata(),
		FromCity:     ff.GetFromCity(),
		ToIATA:       ff.GetToIata(),
		ToCity:       ff.GetToCity(),
		Type:         ff.GetType(),
		FullDesc:     ff.GetFullDescription(),
	}
}
