package service

import (
	"fmt"
	"strings"

	"github.com/aler9/gortsplib/v2"
	"github.com/aler9/gortsplib/v2/pkg/base"
	"github.com/treyhaknson/skyegress/pkg/stream"
)

func pathToSID(path string) string {
	return strings.TrimPrefix(path, "/")
}

type rtspHandler struct {
	manager *stream.SkyEgressStreamManager
}

func NewRTSPHandler(manager *stream.SkyEgressStreamManager) rtspHandler {
	return rtspHandler{manager: manager}
}

func (rh *rtspHandler) Mount(server *gortsplib.Server) {
	server.Handler = rh
}

func (rh *rtspHandler) OnDescribe(ctx *gortsplib.ServerHandlerOnDescribeCtx) (*base.Response, *gortsplib.ServerStream, error) {
	sid := pathToSID(ctx.Path)
	fmt.Println("describe request", sid)

	// attempt to locate the requested stream
	stream, ok := rh.manager.GetStream(sid)
	if !ok {
		return &base.Response{
			StatusCode: base.StatusNotFound,
		}, nil, nil
	}

	// send the request stream
	return &base.Response{
		StatusCode: base.StatusOK,
	}, stream.RTSPStream(), nil
}

func (rh *rtspHandler) OnSetup(ctx *gortsplib.ServerHandlerOnSetupCtx) (*base.Response, *gortsplib.ServerStream, error) {
	sid := pathToSID(ctx.Path)
	fmt.Println("setup request", sid)

	stream, ok := rh.manager.GetStream(sid)
	if !ok {
		return &base.Response{
			StatusCode: base.StatusNotFound,
		}, nil, nil
	}

	// send the request stream
	return &base.Response{
		StatusCode: base.StatusOK,
	}, stream.RTSPStream(), nil
}

func (rh *rtspHandler) OnPlay(ctx *gortsplib.ServerHandlerOnPlayCtx) (*base.Response, error) {
	sid := pathToSID(ctx.Path)
	fmt.Println("play request", sid)

	if _, ok := rh.manager.GetStream(sid); !ok {
		return &base.Response{
			StatusCode: base.StatusNotFound,
		}, nil
	}

	return &base.Response{
		StatusCode: base.StatusOK,
	}, nil
}
