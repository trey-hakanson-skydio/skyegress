package stream

import (
	"fmt"
	"strings"

	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v3"
	"github.com/treyhaknson/skyegress/gen/pbtypes/skyegresspb"

	"github.com/aler9/gortsplib/v2"
	"github.com/aler9/gortsplib/v2/pkg/format"
	"github.com/aler9/gortsplib/v2/pkg/media"
	lksdk "github.com/livekit/server-sdk-go"
	"github.com/livekit/server-sdk-go/pkg/samplebuilder"
)

const (
	maxVideoLate = 500 // ~1s
)

type skyEgressStream struct {
	session    *skyegresspb.Session
	room       *lksdk.Room
	rtspStream *gortsplib.ServerStream
}

func (ss *skyEgressStream) RTSPStream() *gortsplib.ServerStream {
	return ss.rtspStream
}

func (ss *skyEgressStream) Start(host string, info lksdk.ConnectInfo) error {
	wsURL := fmt.Sprintf("wss://%s", host)
	room, err := lksdk.ConnectToRoom(
		wsURL,
		info,
		&lksdk.RoomCallback{
			ParticipantCallback: lksdk.ParticipantCallback{
				OnTrackSubscribed: ss.onTrackSubscribed,
			},
		},
	)

	ss.room = room
	ss.rtspStream = gortsplib.NewServerStream(media.Medias{{
		Type: media.TypeVideo,
		Formats: []format.Format{&format.H264{
			PayloadTyp:        125, // TODO: where does this come from? LiveKit uses 125, gortsplib examples use 96
			PacketizationMode: 1,
		}},
	}})

	return err
}

func (ss *skyEgressStream) Stop() error {
	// TODO: also need to kill the relay goroutine (some sort of channel/context)
	ss.room.Disconnect()
	err := ss.rtspStream.Close()
	return err
}

func (ss *skyEgressStream) onTrackSubscribed(
	track *webrtc.TrackRemote,
	publication *lksdk.RemoteTrackPublication,
	rp *lksdk.RemoteParticipant,
) {
	switch {
	case strings.EqualFold(track.Codec().MimeType, "video/h264"):
		sb := samplebuilder.New(maxVideoLate, &codecs.H264Packet{}, track.Codec().ClockRate, samplebuilder.WithPacketDroppedHandler(func() {
			rp.WritePLI(track.SSRC())
		}))
		go ss.relay(track, sb)
	default:
		break
	}
}

func (ss *skyEgressStream) relay(track *webrtc.TrackRemote, sb *samplebuilder.SampleBuilder) {
	fmt.Println("starting relay for stream", ss.session.Sid)

	for {
		pkt, _, err := track.ReadRTP()
		if err != nil {
			// TODO: should we continue instead of breaking? If we need to break, we need to let the rest
			// of the application know, and likely attempt to re-connect
			break
		}
		sb.Push(pkt)

		for _, p := range sb.PopPackets() {
			for _, medi := range ss.rtspStream.Medias() {
				ss.rtspStream.WritePacketRTP(medi, p)
			}
		}
	}
}
