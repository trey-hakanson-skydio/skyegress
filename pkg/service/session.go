package service

import (
	"fmt"
	"io"
	"net/http"

	lksdk "github.com/livekit/server-sdk-go"
	"github.com/treyhaknson/skyegress/gen/pbtypes/skyegresspb"
	"github.com/treyhaknson/skyegress/pkg/config"
	"github.com/treyhaknson/skyegress/pkg/stream"
	"google.golang.org/protobuf/proto"
)

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

type sessionHandler struct {
	cfg     *config.Config
	manager *stream.SkyEgressStreamManager
}

func NewSessionHandler(cfg *config.Config, manager *stream.SkyEgressStreamManager) sessionHandler {
	return sessionHandler{cfg: cfg, manager: manager}
}

func (sh *sessionHandler) start(w http.ResponseWriter, r *http.Request) {
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

	sid := fmt.Sprintf("%s/%s", req.RoomName, req.TrackName)
	identity := fmt.Sprintf("skyegress-%s-%s", req.RoomName, req.TrackName)
	session := &skyegresspb.Session{
		Sid:            sid,
		RoomName:       req.RoomName,
		TrackName:      req.TrackName,
		EgressIdentity: identity,
	}

	fmt.Println("Adding new stream")
	stream := sh.manager.AddStream(session)
	err = stream.Start(sh.cfg.LiveKitConfig.Host, lksdk.ConnectInfo{
		APIKey:              sh.cfg.LiveKitConfig.ApiKey,
		APISecret:           sh.cfg.LiveKitConfig.ApiSecret,
		RoomName:            req.RoomName,
		ParticipantIdentity: identity,
	})

	if err != nil {
		// TODO: cleanup if we fail to join
		fmt.Println("Failed to start stream", err)
		res.Result = &skyegresspb.StartSessionResponse_Error{Error: err.Error()}
		writeError(w, res)
		return
	}

	fmt.Println("Sending response")
	res.Result = &skyegresspb.StartSessionResponse_Session{Session: session}
	resb, err := proto.Marshal(res)
	if err != nil {
		res.Result = &skyegresspb.StartSessionResponse_Error{Error: err.Error()}
		writeError(w, res)
		return
	}
	w.Write(resb)
}

func (sh *sessionHandler) list(w http.ResponseWriter, r *http.Request) {
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

	sessions := &skyegresspb.Sessions{Sessions: sh.manager.Sessions()}
	res.Result = &skyegresspb.ListSessionsResponse_Sessions{Sessions: sessions}
	resb, err := proto.Marshal(res)
	if err != nil {
		res.Result = &skyegresspb.ListSessionsResponse_Error{Error: err.Error()}
		writeError(w, res)
		return
	}
	w.Write(resb)
}

func (sh *sessionHandler) stop(w http.ResponseWriter, r *http.Request) {
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

	fmt.Println("Removing stream", req.Sid)
	err = sh.manager.RemoveStream(req.Sid)
	if err != nil {
		// TODO(trey): need to handle this better
		fmt.Println("Failed to close RTSP client, session is in a bad state!")
		res.Result = &skyegresspb.StopSessionResponse_Error{Error: err.Error()}
		writeError(w, res)
		return
	}

	fmt.Println("Closed connections and deleted session")
	resb, err := proto.Marshal(res)
	if err != nil {
		res.Result = &skyegresspb.StopSessionResponse_Error{Error: err.Error()}
		writeError(w, res)
		return
	}
	w.Write(resb)
}

func (sh *sessionHandler) Mount(mux *http.ServeMux) {
	mux.HandleFunc("/session/start", sh.start)
	mux.HandleFunc("/session/list", sh.list)
	mux.HandleFunc("/session/stop", sh.stop)
}
