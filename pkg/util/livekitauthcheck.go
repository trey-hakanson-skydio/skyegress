package util

import (
	"context"
	"fmt"

	lkproto "github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go"
)

// TODO(trey): axonbridge used the same check; move into a shared go lib

type livekitAuthCheck struct {
	url       string
	apiKey    string
	apiSecret string
}

func NewLiveKitAuthCheck(url string, apiKey string, apiSecret string) *livekitAuthCheck {
	return &livekitAuthCheck{url: url, apiKey: apiKey, apiSecret: apiSecret}
}

func (lc *livekitAuthCheck) Name() string {
	return "livekit.access"
}

func (lc *livekitAuthCheck) Execute(ctx context.Context) (details interface{}, err error) {
	httpUrl := fmt.Sprintf("https://%s", lc.url)
	roomClient := lksdk.NewRoomServiceClient(httpUrl, lc.apiKey, lc.apiSecret)
	resp, err := roomClient.ListRooms(context.Background(), &lkproto.ListRoomsRequest{})
	if err != nil {
		return nil, err
	}
	d := fmt.Sprintf("succeess (%d rooms)", len(resp.Rooms))
	return d, err
}
