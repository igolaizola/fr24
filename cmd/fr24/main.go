package main

import (
    "compress/gzip"
    "context"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "os/signal"
    "runtime/debug"
    "strings"
    "time"

    lib "github.com/igolaizola/fr24/pkg/flightradar"
    "github.com/peterbourgon/ff/v3"
    "github.com/peterbourgon/ff/v3/ffcli"
    "github.com/peterbourgon/ff/v3/ffyaml"
)

// Build flags
var version = ""
var commit = ""
var date = ""

func main() {
    // Signal-based context
    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
    defer cancel()

    cmd := newCommand()
    if err := cmd.ParseAndRun(ctx, os.Args[1:]); err != nil {
        log.Fatal(err)
    }
}

func newCommand() *ffcli.Command {
    fs := flag.NewFlagSet("fr24", flag.ExitOnError)
    return &ffcli.Command{
        ShortUsage: "fr24 [flags] <subcommand>",
        FlagSet:    fs,
        Exec: func(context.Context, []string) error {
            return flag.ErrHelp
        },
        Subcommands: []*ffcli.Command{
            newVersionCommand(),
            cmdLogin(),
            cmdDirs(),
            cmdFlightList(),
            cmdAirportList(),
            cmdFind(),
            cmdLiveFeed(),
            cmdPlaybackFeed(),
            cmdNearest(),
            cmdLiveStatus(),
            cmdTopFlights(),
            cmdFlightDetails(),
            cmdPlaybackFlight(),
            cmdFollowFlight(),
        },
    }
}

func newVersionCommand() *ffcli.Command {
    return &ffcli.Command{
        Name:       "version",
        ShortUsage: "fr24 version",
        ShortHelp:  "print version",
        Exec: func(ctx context.Context, args []string) error {
            v := version
            if v == "" {
                if bi, ok := debug.ReadBuildInfo(); ok {
                    v = bi.Main.Version
                }
            }
            if v == "" {
                v = "dev"
            }
            fields := []string{v}
            if commit != "" {
                fields = append(fields, commit)
            }
            if date != "" {
                fields = append(fields, date)
            }
            fmt.Println(strings.Join(fields, " "))
            return nil
        },
    }
}

func cmdLogin() *ffcli.Command {
    fs := flag.NewFlagSet("login", flag.ExitOnError)
    return &ffcli.Command{
        Name:       "login",
        ShortUsage: "fr24 login",
        ShortHelp:  "authenticate using env/config",
        FlagSet:    fs,
        Exec: func(ctx context.Context, args []string) error {
            c := lib.New()
            if err := c.LoginFromEnvOrConfig(); err != nil {
                return err
            }
            if c.AuthMode() == "anonymous" {
                fmt.Println("login anonymous")
            } else {
                fmt.Println("login ok")
            }
            return nil
        },
    }
}

func cmdDirs() *ffcli.Command {
    fs := flag.NewFlagSet("dirs", flag.ExitOnError)
    return &ffcli.Command{
        Name:       "dirs",
        ShortUsage: "fr24 dirs",
        ShortHelp:  "print cache directories",
        FlagSet:    fs,
        Exec: func(ctx context.Context, args []string) error {
            cache, err := lib.DefaultCache()
            if err != nil {
                return err
            }
            b, _ := json.MarshalIndent(map[string]string{"base": cache.Base()}, "", "  ")
            _, _ = os.Stdout.Write(b)
            return nil
        },
    }
}

func cmdFlightList() *ffcli.Command {
    fs := flag.NewFlagSet("flightlist", flag.ExitOnError)
    reg := fs.String("reg", "", "registration")
    flt := fs.String("flight", "", "flight number")
    return &ffcli.Command{
        Name:       "flightlist",
        ShortUsage: "fr24 flightlist [flags]",
        ShortHelp:  "list flights by registration or number",
        FlagSet:    fs,
        Exec: func(ctx context.Context, args []string) error {
            c := lib.New()
            _ = c.LoginFromEnvOrConfig()
            resp, err := c.FlightList(ctx, lib.FlightListParams{Reg: *reg, Flight: *flt, Page: 1, Limit: 10})
            if err != nil {
                return err
            }
            defer func() { _ = resp.Body.Close() }()
            body, _ := readBody(resp)
            recs, err := lib.ParseFlightList(body)
            if err != nil {
                return err
            }
            enc := json.NewEncoder(os.Stdout)
            enc.SetIndent("", "  ")
            return enc.Encode(recs)
        },
    }
}

func cmdAirportList() *ffcli.Command {
    fs := flag.NewFlagSet("airportlist", flag.ExitOnError)
    code := fs.String("code", "HKG", "IATA code")
    mode := fs.String("mode", "arrivals", "arrivals|departures|ground")
    return &ffcli.Command{
        Name:       "airportlist",
        ShortUsage: "fr24 airportlist [flags]",
        ShortHelp:  "airport schedule list",
        FlagSet:    fs,
        Exec: func(ctx context.Context, args []string) error {
            c := lib.New()
            _ = c.LoginFromEnvOrConfig()
            resp, err := c.AirportList(ctx, lib.AirportListParams{Airport: *code, Mode: lib.AirportMode(*mode), Page: 1, Limit: 10})
            if err != nil {
                return err
            }
            defer func() { _ = resp.Body.Close() }()
            _, err = io.Copy(os.Stdout, mustReadBodyReader(resp))
            return err
        },
    }
}

func cmdFind() *ffcli.Command {
    fs := flag.NewFlagSet("find", flag.ExitOnError)
    q := fs.String("q", "A359", "query")
    return &ffcli.Command{
        Name:       "find",
        ShortUsage: "fr24 find [flags]",
        ShortHelp:  "search entities",
        FlagSet:    fs,
        Exec: func(ctx context.Context, args []string) error {
            c := lib.New()
            _ = c.LoginFromEnvOrConfig()
            resp, err := c.Find(ctx, lib.FindParams{Query: *q, Limit: 50})
            if err != nil {
                return err
            }
            defer func() { _ = resp.Body.Close() }()
            _, err = io.Copy(os.Stdout, mustReadBodyReader(resp))
            return err
        },
    }
}

func cmdLiveFeed() *ffcli.Command {
    fs := flag.NewFlagSet("livefeed", flag.ExitOnError)
    south := fs.Float64("south", 42, "south")
    north := fs.Float64("north", 52, "north")
    west := fs.Float64("west", -8, "west")
    east := fs.Float64("east", 10, "east")
    return &ffcli.Command{
        Name:       "livefeed",
        ShortUsage: "fr24 livefeed [flags]",
        ShortHelp:  "live feed in bounding box",
        FlagSet:    fs,
        Exec: func(ctx context.Context, args []string) error {
            c := lib.New()
            _ = c.LoginFromEnvOrConfig()
            p := lib.LiveFeedParams{BoundingBox: lib.BoundingBox{South: float32(*south), North: float32(*north), West: float32(*west), East: float32(*east)}}
            resp, err := c.GrpcLiveFeed(ctx, p)
            if err != nil {
                return err
            }
            defer func() { _ = resp.Body.Close() }()
            b, _ := io.ReadAll(resp.Body)
            msg, err := lib.ParseLiveFeedGRPC(b)
            if err != nil {
                return err
            }
            out := make([]lib.LiveFeedFlightRecord, 0, len(msg.GetFlightsList()))
            for _, f := range msg.GetFlightsList() {
                out = append(out, lib.LiveFeedFlightToRecord(f))
            }
            return json.NewEncoder(os.Stdout).Encode(out)
        },
    }
}

func cmdPlaybackFeed() *ffcli.Command {
    fs := flag.NewFlagSet("playbackfeed", flag.ExitOnError)
    south := fs.Float64("south", 42, "south")
    north := fs.Float64("north", 52, "north")
    west := fs.Float64("west", -8, "west")
    east := fs.Float64("east", 10, "east")
    dur := fs.Int("duration", 7, "duration seconds")
    return &ffcli.Command{
        Name:       "playbackfeed",
        ShortUsage: "fr24 playbackfeed [flags]",
        ShortHelp:  "historical live feed snapshot",
        FlagSet:    fs,
        Exec: func(ctx context.Context, args []string) error {
            c := lib.New()
            _ = c.LoginFromEnvOrConfig()
            p := lib.LiveFeedPlaybackParams{LiveFeed: lib.LiveFeedParams{BoundingBox: lib.BoundingBox{South: float32(*south), North: float32(*north), West: float32(*west), East: float32(*east)}}, Duration: int32(*dur)}
            resp, err := c.GrpcPlayback(ctx, p)
            if err != nil {
                return err
            }
            defer func() { _ = resp.Body.Close() }()
            b, _ := io.ReadAll(resp.Body)
            msg, err := lib.ParsePlaybackGRPC(b)
            if err != nil {
                return err
            }
            out := make([]lib.LiveFeedFlightRecord, 0, len(msg.GetLiveFeedResponse().GetFlightsList()))
            for _, f := range msg.GetLiveFeedResponse().GetFlightsList() {
                out = append(out, lib.LiveFeedFlightToRecord(f))
            }
            return json.NewEncoder(os.Stdout).Encode(out)
        },
    }
}

func cmdNearest() *ffcli.Command {
    fs := flag.NewFlagSet("nearest", flag.ExitOnError)
    lat := fs.Float64("lat", 22.3, "lat")
    lon := fs.Float64("lon", 114.2, "lon")
    return &ffcli.Command{
        Name:       "nearest",
        ShortUsage: "fr24 nearest [flags]",
        ShortHelp:  "nearest flights to a location",
        FlagSet:    fs,
        Exec: func(ctx context.Context, args []string) error {
            c := lib.New()
            _ = c.LoginFromEnvOrConfig()
            resp, err := c.GrpcNearestFlights(ctx, lib.NearestFlightsParams{Lat: float32(*lat), Lon: float32(*lon)})
            if err != nil {
                return err
            }
            defer func() { _ = resp.Body.Close() }()
            b, _ := io.ReadAll(resp.Body)
            msg, err := lib.ParseNearestFlightsGRPC(b)
            if err != nil {
                return err
            }
            return json.NewEncoder(os.Stdout).Encode(lib.NearbyToRecords(msg))
        },
    }
}

func cmdLiveStatus() *ffcli.Command {
    fs := flag.NewFlagSet("livestatus", flag.ExitOnError)
    id := fs.Uint("id", 0, "flight id")
    return &ffcli.Command{
        Name:       "livestatus",
        ShortUsage: "fr24 livestatus [flags]",
        ShortHelp:  "live flight status",
        FlagSet:    fs,
        Exec: func(ctx context.Context, args []string) error {
            if *id == 0 {
                return errors.New("missing -id")
            }
            c := lib.New()
            _ = c.LoginFromEnvOrConfig()
            resp, err := c.GrpcLiveFlightsStatus(ctx, lib.LiveFlightsStatusParams{FlightIDs: []uint32{uint32(*id)}})
            if err != nil {
                return err
            }
            defer func() { _ = resp.Body.Close() }()
            b, _ := io.ReadAll(resp.Body)
            msg, err := lib.ParseLiveFlightsStatusGRPC(b)
            if err != nil {
                return err
            }
            return json.NewEncoder(os.Stdout).Encode(lib.LiveFlightsStatusToRecords(msg))
        },
    }
}

func cmdTopFlights() *ffcli.Command {
    fs := flag.NewFlagSet("topflights", flag.ExitOnError)
    limit := fs.Int("limit", 10, "limit 1-10")
    return &ffcli.Command{
        Name:       "topflights",
        ShortUsage: "fr24 topflights [flags]",
        ShortHelp:  "most viewed flights",
        FlagSet:    fs,
        Exec: func(ctx context.Context, args []string) error {
            c := lib.New()
            _ = c.LoginFromEnvOrConfig()
            resp, err := c.GrpcTopFlights(ctx, lib.TopFlightsParams{Limit: int32(*limit)})
            if err != nil {
                return err
            }
            defer func() { _ = resp.Body.Close() }()
            body, _ := io.ReadAll(resp.Body)
            tf, err := lib.ParseTopFlightsGRPC(body)
            if err != nil {
                return err
            }
            var out []lib.TopFlightRecord
            for _, ff := range tf.GetScoreboardList() {
                out = append(out, lib.TopFlightToRecord(ff))
            }
            return json.NewEncoder(os.Stdout).Encode(out)
        },
    }
}

func cmdFlightDetails() *ffcli.Command {
    fs := flag.NewFlagSet("flightdetails", flag.ExitOnError)
    id := fs.Uint("id", 0, "flight id")
    return &ffcli.Command{
        Name:       "flightdetails",
        ShortUsage: "fr24 flightdetails [flags]",
        ShortHelp:  "details for a live flight",
        FlagSet:    fs,
        Exec: func(ctx context.Context, args []string) error {
            if *id == 0 {
                return errors.New("missing -id")
            }
            c := lib.New()
            _ = c.LoginFromEnvOrConfig()
            resp, err := c.GrpcFlightDetails(ctx, lib.FlightDetailsParams{FlightID: uint32(*id)})
            if err != nil {
                return err
            }
            defer func() { _ = resp.Body.Close() }()
            b, _ := io.ReadAll(resp.Body)
            msg, err := lib.ParseFlightDetailsGRPC(b)
            if err != nil {
                return err
            }
            return json.NewEncoder(os.Stdout).Encode(lib.FlightDetailsToRecord(msg))
        },
    }
}

func cmdPlaybackFlight() *ffcli.Command {
    fs := flag.NewFlagSet("playbackflight", flag.ExitOnError)
    id := fs.Uint("id", 0, "flight id")
    ts := fs.Uint64("ts", uint64(time.Now().Unix()), "departure ts")
    return &ffcli.Command{
        Name:       "playbackflight",
        ShortUsage: "fr24 playbackflight [flags]",
        ShortHelp:  "details for a historic flight",
        FlagSet:    fs,
        Exec: func(ctx context.Context, args []string) error {
            if *id == 0 {
                return errors.New("missing -id")
            }
            c := lib.New()
            _ = c.LoginFromEnvOrConfig()
            resp, err := c.GrpcPlaybackFlight(ctx, lib.PlaybackFlightParams{FlightID: uint32(*id), Timestamp: *ts})
            if err != nil {
                return err
            }
            defer func() { _ = resp.Body.Close() }()
            b, _ := io.ReadAll(resp.Body)
            msg, err := lib.ParsePlaybackFlightGRPC(b)
            if err != nil {
                return err
            }
            return json.NewEncoder(os.Stdout).Encode(lib.PlaybackFlightToRecord(msg))
        },
    }
}

func cmdFollowFlight() *ffcli.Command {
    fs := flag.NewFlagSet("followflight", flag.ExitOnError)
    id := fs.Uint("id", 0, "flight id")
    timeout := fs.Int("timeout", 0, "seconds to run (0=until Ctrl-C)")
    once := fs.Bool("once", false, "exit after first frame")
    return &ffcli.Command{
        Name:       "followflight",
        ShortUsage: "fr24 followflight [flags]",
        ShortHelp:  "stream updates for a flight",
        FlagSet:    fs,
        Options: []ff.Option{
            ff.WithConfigFileFlag("config"),
            ff.WithConfigFileParser(ffyaml.Parser),
            ff.WithEnvVarPrefix("FR24"),
        },
        Exec: func(ctx context.Context, args []string) error {
            if *id == 0 {
                return errors.New("missing -id")
            }
            c := lib.New()
            _ = c.LoginFromEnvOrConfig()
            // Optional timeout for consistent tests
            if *timeout > 0 {
                var cancelTimeout context.CancelFunc
                ctx, cancelTimeout = context.WithTimeout(ctx, time.Duration(*timeout)*time.Second)
                defer cancelTimeout()
            }
            ch, cancel, err := c.GrpcFollowFlightStream(ctx, uint32(*id), 0)
            if err != nil {
                return err
            }
            defer cancel()
            enc := json.NewEncoder(os.Stdout)
            wrote := false
            for frame := range ch {
                if msg, err := lib.ParseLiveFeedGRPC(frame); err == nil {
                    out := make([]lib.LiveFeedFlightRecord, 0, len(msg.GetFlightsList()))
                    for _, f := range msg.GetFlightsList() {
                        out = append(out, lib.LiveFeedFlightToRecord(f))
                    }
                    if err := enc.Encode(out); err != nil {
                        return err
                    }
                    wrote = true
                    if *once {
                        break
                    }
                }
            }
            _ = wrote
            return nil
        },
    }
}

// Helpers preserved from previous implementation
func mustReadBodyReader(resp *http.Response) io.Reader {
    if resp.Header.Get("Content-Encoding") == "gzip" {
        zr, err := gzip.NewReader(resp.Body)
        if err == nil {
            return zr
        }
    }
    return resp.Body
}

func readBody(resp *http.Response) ([]byte, error) {
    defer func() { _ = resp.Body.Close() }()
    if resp.Header.Get("Content-Encoding") == "gzip" {
        zr, err := gzip.NewReader(resp.Body)
        if err != nil {
            return nil, err
        }
        defer func() { _ = zr.Close() }()
        return io.ReadAll(zr)
    }
    return io.ReadAll(resp.Body)
}
