package cmd

import (
	"errors"
	"fmt"

	"github.com/treyhaknson/skyegress/gen/pbtypes/skyegresspb"
	"github.com/treyhaknson/skyegress/pkg/util"
)

type ClientCmd struct {
	URL string `kong:"help='The url of the skyegress service to connect to',default='http://localhost:8008'"`

	Start ClientStartCmd `kong:"cmd,help='Start an egress session'"`
	List  ClientListCmd  `kong:"cmd,help='List active egress sessions'"`
	Stop  ClientStopCmd  `kong:"cmd,help='Start an egress session'"`
}

type ClientStartCmd struct {
	RoomName  string `kong:"help='Name of the LiveKit room to join'"`
	TrackName string `kong:"help='Name of the track in the LiveKit room to egress'"`
}

func (cs *ClientStartCmd) Run(cmn *ClientCmd) error {
	req := &skyegresspb.StartSessionRequest{RoomName: cs.RoomName, TrackName: cs.TrackName}
	res := &skyegresspb.StartSessionResponse{}
	pc := util.NewProtoClient(cmn.URL)
	err := pc.Request(util.POST, "/start", req, res)
	if err != nil {
		panic(err)
	}
	switch res.Result.(type) {
	case *skyegresspb.StartSessionResponse_Error:
		panic(errors.New(res.GetError()))
	case *skyegresspb.StartSessionResponse_Session:
		fmt.Printf("Successfully started session %+v", res.GetSession())
	}
	return nil
}

type ClientStopCmd struct {
	RoomName  string `kong:"help='Room name associated with the session to end'"`
	TrackName string `kong:"help='Track name associated with the session to end'"`
}

func (cs *ClientStopCmd) Run(cmn *ClientCmd) error {
	req := &skyegresspb.StopSessionRequest{RoomName: cs.RoomName, TrackName: cs.TrackName}
	res := &skyegresspb.StopSessionResponse{}
	pc := util.NewProtoClient(cmn.URL)
	err := pc.Request(util.POST, "/stop", req, res)
	if err != nil {
		panic(err)
	}
	switch res.Result.(type) {
	case *skyegresspb.StopSessionResponse_Error:
		panic(errors.New(res.GetError()))
	case *skyegresspb.StopSessionResponse_Session:
		fmt.Printf("Successfully stopped session %+v", res.GetSession())
	}
	return nil
}

type ClientListCmd struct{}

func (cl *ClientListCmd) Run(cmn *ClientCmd) error {
	req := &skyegresspb.ListSessionsRequest{}
	res := &skyegresspb.ListSessionsResponse{}
	pc := util.NewProtoClient(cmn.URL)
	err := pc.Request(util.POST, "/list", req, res)
	if err != nil {
		panic(err)
	}
	switch res.Result.(type) {
	case *skyegresspb.ListSessionsResponse_Error:
		panic(errors.New(res.GetError()))
	case *skyegresspb.ListSessionsResponse_Sessions:
		for i, session := range res.GetSessions().Sessions {
			fmt.Printf("%d\t%s\t%s\t%s\n", i, session.RoomName, session.TrackName, session.EgressIdentity)
		}
	}
	return nil
}
