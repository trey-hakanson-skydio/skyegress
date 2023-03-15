package stream

import (
	"errors"
	"fmt"
	"sync"

	"github.com/treyhaknson/skyegress/gen/pbtypes/skyegresspb"
)

type SkyEgressStreamManager struct {
	streamsLock sync.RWMutex
	streams     map[string]*skyEgressStream
}

func NewSkyEgressStreamManager() SkyEgressStreamManager {
	return SkyEgressStreamManager{
		streams: make(map[string]*skyEgressStream),
	}
}

func (sm *SkyEgressStreamManager) GetStream(sid string) (*skyEgressStream, bool) {
	sm.streamsLock.RLock()
	defer sm.streamsLock.RUnlock()

	stream, ok := sm.streams[sid]

	return stream, ok
}

func (sm *SkyEgressStreamManager) AddStream(session *skyegresspb.Session) *skyEgressStream {
	sm.streamsLock.Lock()
	defer sm.streamsLock.Unlock()

	// TODO: what if a stream matching this session already exists?
	stream := &skyEgressStream{session: session}
	sm.streams[session.Sid] = stream
	return stream
}

func (sm *SkyEgressStreamManager) RemoveStream(sid string) error {
	stream, ok := sm.GetStream(sid)
	if !ok {
		msg := fmt.Sprintf("Stream %s did not exist", sid)
		return errors.New(msg)
	}

	// TODO: if this fails, should we still delete the stream from map?
	err := stream.Stop()
	if err != nil {
		return err
	}

	sm.streamsLock.Lock()
	delete(sm.streams, sid)
	sm.streamsLock.Unlock()

	return nil
}

func (sm *SkyEgressStreamManager) Sessions() []*skyegresspb.Session {
	sm.streamsLock.RLock()
	defer sm.streamsLock.RUnlock()

	sessions := make([]*skyegresspb.Session, len(sm.streams))
	for _, stream := range sm.streams {
		sessions = append(sessions, stream.session)
	}

	return sessions
}
