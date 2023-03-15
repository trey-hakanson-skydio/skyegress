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

func (sm *SkyEgressStreamManager) AddStream(session *skyegresspb.Session) (*skyEgressStream, error) {
	sm.streamsLock.Lock()
	defer sm.streamsLock.Unlock()

	if _, ok := sm.streams[session.Sid]; ok {
		msg := fmt.Sprintf("stream with SID %s already exists", session.Sid)
		return nil, errors.New(msg)
	}

	stream := NewSkyEgressStream(session)
	sm.streams[session.Sid] = &stream
	return &stream, nil
}

func (sm *SkyEgressStreamManager) RemoveStream(sid string) {
	stream, ok := sm.GetStream(sid)
	if !ok {
		fmt.Printf("Stream %s did not exist\n", sid)
		return
	}

	err := stream.Stop()
	if err != nil {
		// TODO(trey): how can we handle this better?
		fmt.Println("Unable to stop stream successfully; still removing session")
	}

	sm.streamsLock.Lock()
	delete(sm.streams, sid)
	sm.streamsLock.Unlock()
}

func (sm *SkyEgressStreamManager) Sessions() []*skyegresspb.Session {
	sm.streamsLock.RLock()
	defer sm.streamsLock.RUnlock()

	sessions := make([]*skyegresspb.Session, 0, len(sm.streams))
	for _, stream := range sm.streams {
		sessions = append(sessions, stream.session)
	}

	return sessions
}
