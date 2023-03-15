package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	gosundheit "github.com/AppsFlyer/go-sundheit"
	healthhttp "github.com/AppsFlyer/go-sundheit/http"
	"github.com/aler9/gortsplib/v2"
	"github.com/aler9/gortsplib/v2/pkg/base"
	"github.com/aler9/gortsplib/v2/pkg/format"
	"github.com/aler9/gortsplib/v2/pkg/media"
	lksdk "github.com/livekit/server-sdk-go"
	"github.com/livekit/server-sdk-go/pkg/samplebuilder"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v3"
	"google.golang.org/protobuf/proto"

	"github.com/treyhaknson/skyegress/gen/pbtypes/skyegresspb"
	"github.com/treyhaknson/skyegress/pkg/util"
)

// TODO(trey): need to check if pointers a nil throughout to avoid crashing on a bad dereference

const (
	maxVideoLate = 500 // ~1s
)

type ServeCmd struct{}

type skyEgressServer struct {
	*Common

	streamsLock sync.RWMutex
	streams     map[string]*skyEgressStream
}

func NewSkyEgressServer(cmn *Common) skyEgressServer {
	return skyEgressServer{
		Common:  cmn,
		streams: make(map[string]*skyEgressStream),
	}
}

type skyEgressStream struct {
	session    *skyegresspb.Session
	room       *lksdk.Room
	rtspClient *gortsplib.Client
	rtspStream *gortsplib.ServerStream
}

func sessionID(roomName string, trackName string) string {
	return strings.ToLower(roomName + "-" + trackName)
}

func writeError(w http.ResponseWriter, res proto.Message) {
	resb, err := proto.Marshal(res)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(http.StatusBadRequest)
	w.Write(resb)
}

func (sc *ServeCmd) Run(cmn *Common) error {
	skyEgress := NewSkyEgressServer(cmn)
	ctx, cancelCtx := context.WithCancel(context.Background())

	mux := http.NewServeMux()
	// TODO(trey): investigate twirp; I don't like the structure of these endpoints
	mux.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Received start request")
		res := &skyegresspb.StartSessionResponse{}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			res.Result = &skyegresspb.StartSessionResponse_Error{Error: err.Error()}
			writeError(w, res)
			return
		}

		req := skyegresspb.StartSessionRequest{}
		err = proto.Unmarshal(body, &req)
		if err != nil {
			res.Result = &skyegresspb.StartSessionResponse_Error{Error: err.Error()}
			writeError(w, res)
			return
		}
		fmt.Printf("Parsed start request %s, %s\n", req.RoomName, req.TrackName)

		if len(req.RoomName) == 0 {
			res.Result = &skyegresspb.StartSessionResponse_Error{Error: "room_name must be provided"}
			writeError(w, res)
			return
		}

		if len(req.TrackName) == 0 {
			res.Result = &skyegresspb.StartSessionResponse_Error{Error: "track_name must be provided"}
			writeError(w, res)
			return
		}

		sid := sessionID(req.RoomName, req.TrackName)
		session := &skyegresspb.Session{}
		session.RoomName = req.RoomName
		session.TrackName = req.TrackName
		session.EgressIdentity = fmt.Sprintf("skyegress-%s-%s", req.RoomName, req.TrackName)

		fmt.Println("Adding session")
		skyEgress.streamsLock.Lock()
		skyEgress.streams[sid] = &skyEgressStream{session: session}
		skyEgress.streamsLock.Unlock()

		// NOTE: doing this separately to ensure the stream and session are availble within the
		// OnAnnounce handler
		fmt.Println("Creating RTSP client")
		medi := &media.Media{
			Type: media.TypeVideo,
			Formats: []format.Format{&format.H264{
				PayloadTyp:        125, // TODO: where does this come from? LiveKit uses 125, gortsplib examples use 96
				PacketizationMode: 1,
			}},
		}
		client := &gortsplib.Client{}
		streamAddr := fmt.Sprintf("rtsp://localhost:8554/%s", sid)
		err = client.StartRecording(streamAddr, media.Medias{medi})
		if err != nil {
			// TODO: cleanup the stream
			res.Result = &skyegresspb.StartSessionResponse_Error{Error: err.Error()}
			writeError(w, res)
			return
		}

		fmt.Println("Attaching RTSP client to session")
		skyEgress.streamsLock.Lock()
		skyEgress.streams[sid].rtspClient = client
		skyEgress.streamsLock.Unlock()

		fmt.Println("Sending response")
		res.Result = &skyegresspb.StartSessionResponse_Session{Session: session}
		resb, err := proto.Marshal(res)
		if err != nil {
			res.Result = &skyegresspb.StartSessionResponse_Error{Error: err.Error()}
			writeError(w, res)
			return
		}
		w.Write(resb)
	})
	mux.HandleFunc("/list", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Received list request")
		res := &skyegresspb.ListSessionsResponse{}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			res.Result = &skyegresspb.ListSessionsResponse_Error{Error: err.Error()}
			writeError(w, res)
			return
		}

		req := skyegresspb.StopSessionRequest{}
		err = proto.Unmarshal(body, &req)
		if err != nil {
			res.Result = &skyegresspb.ListSessionsResponse_Error{Error: err.Error()}
			writeError(w, res)
			return
		}

		sessions := &skyegresspb.Sessions{}

		skyEgress.streamsLock.RLock()
		for _, stream := range skyEgress.streams {
			sessions.Sessions = append(sessions.Sessions, stream.session)
		}
		skyEgress.streamsLock.RUnlock()

		res.Result = &skyegresspb.ListSessionsResponse_Sessions{Sessions: sessions}
		resb, err := proto.Marshal(res)
		if err != nil {
			res.Result = &skyegresspb.ListSessionsResponse_Error{Error: err.Error()}
			writeError(w, res)
			return
		}
		w.Write(resb)
	})
	mux.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Received stop request")
		res := &skyegresspb.StopSessionResponse{}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			res.Result = &skyegresspb.StopSessionResponse_Error{Error: err.Error()}
			writeError(w, res)
			return
		}

		req := skyegresspb.StopSessionRequest{}
		err = proto.Unmarshal(body, &req)
		if err != nil {
			res.Result = &skyegresspb.StopSessionResponse_Error{Error: err.Error()}
			writeError(w, res)
			return
		}

		sid := sessionID(req.RoomName, req.TrackName)
		fmt.Printf("Parsed stop request %s, %s, %s\n", sid, req.RoomName, req.TrackName)

		// TODO(trey): move teardown into a dedicated function so it can be called elsewhere
		skyEgress.streamsLock.Lock()
		stream := skyEgress.streams[sid]
		res.Result = &skyegresspb.StopSessionResponse_Session{Session: stream.session}
		err = stream.rtspStream.Close()
		if err != nil {
			// TODO(trey): really need to handle this better
			fmt.Println("Failed to close RTSP client, session is in a bad state!")
		}
		stream.room.Disconnect()
		delete(skyEgress.streams, sid)
		skyEgress.streamsLock.Unlock()

		fmt.Println("Closed connections and deleted session")

		resb, err := proto.Marshal(res)
		if err != nil {
			res.Result = &skyegresspb.StopSessionResponse_Error{Error: err.Error()}
			writeError(w, res)
			return
		}
		w.Write(resb)
	})
	gh := gosundheit.New()
	lkAuthCheck := util.NewLiveKitAuthCheck(cmn.Host, cmn.ApiKey, cmn.ApiSecret)
	err := gh.RegisterCheck(
		lkAuthCheck,
		gosundheit.InitialDelay(5*time.Second),
		gosundheit.ExecutionPeriod(10*time.Second),
	)
	if err != nil {
		// TODO(trey): this is goofy, we don't need to crash here
		panic(err)
	}
	mux.HandleFunc("/health", healthhttp.HandleHealthJSON(gh))

	httpServer := &http.Server{
		Addr:    ":8008",
		Handler: mux,
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}

	rtspServer := &gortsplib.Server{
		Handler:           &skyEgress,
		RTSPAddress:       ":8554",
		UDPRTPAddress:     ":8000",
		UDPRTCPAddress:    ":8001",
		MulticastIPRange:  "224.1.0.0/16",
		MulticastRTPPort:  8002,
		MulticastRTCPPort: 8003,
	}

	go func() {
		fmt.Println("starting http server on 8008")
		err := httpServer.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			fmt.Println("http server closed")
		} else if err != nil {
			fmt.Printf("error listening for http server %s\n", err)
		}
		cancelCtx()
	}()

	go func() {
		fmt.Println("starting rtsp server on 8554")
		err := rtspServer.StartAndWait()
		if err != nil {
			fmt.Printf("error listening for rtsp server: %s\n", err)
		}
		cancelCtx()
	}()

	<-ctx.Done()

	return nil
}

func (se *skyEgressServer) onTrackSubscribed(
	track *webrtc.TrackRemote,
	publication *lksdk.RemoteTrackPublication,
	rp *lksdk.RemoteParticipant,
) {
	switch {
	case strings.EqualFold(track.Codec().MimeType, "video/h264"):
		sb := samplebuilder.New(maxVideoLate, &codecs.H264Packet{}, track.Codec().ClockRate, samplebuilder.WithPacketDroppedHandler(func() {
			rp.WritePLI(track.SSRC())
		}))
		// FIXME(trey): hardcoding the session ID for now!
		sid := sessionID("devroom", "demo")
		go se.relay(sid, track, sb)
	default:
		break
	}
}

func (se *skyEgressServer) relay(sid string, track *webrtc.TrackRemote, sb *samplebuilder.SampleBuilder) {
	fmt.Println("starting relay for stream", sid)

	for {
		pkt, _, err := track.ReadRTP()
		if err != nil {
			break
		}
		sb.Push(pkt)

		for _, p := range sb.PopPackets() {
			// FIXME(trey): this locking isn't great, because all relays will get paused whenever a stream
			// is added/killed
			// TODO(trey): handle errors better; don't want the relay to crash because of 1 bad packet or
			// something
			se.streamsLock.RLock()
			stream := se.streams[sid]
			for _, media := range stream.rtspStream.Medias() {
				stream.rtspClient.WritePacketRTP(media, p)
			}
			se.streamsLock.RUnlock()
		}
	}
}

// RTSP server handlers

func (se *skyEgressServer) OnDescribe(ctx *gortsplib.ServerHandlerOnDescribeCtx) (*base.Response, *gortsplib.ServerStream, error) {
	path := strings.TrimPrefix(ctx.Path, "/")
	fmt.Println("describe request", path)

	se.streamsLock.RLock()
	defer se.streamsLock.RUnlock()

	// attempt to locate the requested stream
	stream, ok := se.streams[path]
	if !ok {
		return &base.Response{
			StatusCode: base.StatusNotFound,
		}, nil, nil
	}

	// send the request stream
	return &base.Response{
		StatusCode: base.StatusOK,
	}, stream.rtspStream, nil
}

func (se *skyEgressServer) OnAnnounce(ctx *gortsplib.ServerHandlerOnAnnounceCtx) (*base.Response, error) {
	path := strings.TrimPrefix(ctx.Path, "/")
	fmt.Println("announce request", path)

	// TODO: kill pre-existing stream if one already exists
	stream := gortsplib.NewServerStream(ctx.Medias)

	se.streamsLock.RLock()
	se.streams[path].rtspStream = stream
	se.streamsLock.RUnlock()

	return &base.Response{
		StatusCode: base.StatusOK,
	}, nil
}

func (se *skyEgressServer) OnRecord(ctx *gortsplib.ServerHandlerOnRecordCtx) (*base.Response, error) {
	path := strings.TrimPrefix(ctx.Path, "/")
	fmt.Println("record request", path)

	se.streamsLock.RLock()
	stream, ok := se.streams[path]
	se.streamsLock.RUnlock()

	if !ok {
		return &base.Response{
			StatusCode: base.StatusNotFound,
		}, nil
	}

	fmt.Println("Joining LiveKit room")
	wsUrl := fmt.Sprintf("wss://%s", se.Host)
	_, err := lksdk.ConnectToRoom(wsUrl, lksdk.ConnectInfo{
		APIKey:              se.ApiKey,
		APISecret:           se.ApiSecret,
		RoomName:            stream.session.RoomName,
		ParticipantIdentity: stream.session.EgressIdentity,
	}, &lksdk.RoomCallback{
		ParticipantCallback: lksdk.ParticipantCallback{
			OnTrackSubscribed: se.onTrackSubscribed,
		},
	})

	if err != nil {
		// TODO: cleanup session if we fail to join?
		fmt.Println("Failed to join LiveKit room", err)
		return &base.Response{
			StatusCode: base.StatusInternalServerError,
		}, err
	}

	fmt.Println("Connected to LiveKit room")

	ctx.Session.OnPacketRTPAny(func(medi *media.Media, forma format.Format, pkt *rtp.Packet) {
		fmt.Printf("Received packet (%d): %+v\n", pkt.PayloadType, medi)
		stream.rtspStream.WritePacketRTP(medi, pkt)
	})

	return &base.Response{
		StatusCode: base.StatusOK,
	}, nil
}

func (se *skyEgressServer) OnSetup(ctx *gortsplib.ServerHandlerOnSetupCtx) (*base.Response, *gortsplib.ServerStream, error) {
	path := strings.TrimPrefix(ctx.Path, "/")
	fmt.Println("setup request", path)

	// attempt to locate the requested stream
	stream, ok := se.streams[path]
	if !ok {
		return &base.Response{
			StatusCode: base.StatusNotFound,
		}, nil, nil
	}

	// send the request stream
	return &base.Response{
		StatusCode: base.StatusOK,
	}, stream.rtspStream, nil
}

func (se *skyEgressServer) OnPlay(ctx *gortsplib.ServerHandlerOnPlayCtx) (*base.Response, error) {
	path := strings.TrimPrefix(ctx.Path, "/")
	fmt.Println("play request", path)

	return &base.Response{
		StatusCode: base.StatusOK,
	}, nil
}
