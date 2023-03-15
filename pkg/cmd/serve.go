package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/aler9/gortsplib/v2"

	"github.com/treyhaknson/skyegress/pkg/config"
	"github.com/treyhaknson/skyegress/pkg/service"
	"github.com/treyhaknson/skyegress/pkg/stream"
)

type ServeCmd struct{}

func (sc *ServeCmd) Run(cfg *config.Config) error {
	ctx, cancelCtx := context.WithCancel(context.Background())

	mux := http.NewServeMux()

	manager := stream.NewSkyEgressStreamManager()
	sh := service.NewSessionHandler(cfg, &manager)
	sh.Mount(mux)

	hh := service.NewHealthHandler(cfg)
	hh.Mount(mux)

	httpServer := &http.Server{
		Addr:    ":8008",
		Handler: mux,
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}

	rtspServer := &gortsplib.Server{
		RTSPAddress:       ":8554",
		UDPRTPAddress:     ":8000",
		UDPRTCPAddress:    ":8001",
		MulticastIPRange:  "224.1.0.0/16",
		MulticastRTPPort:  8002,
		MulticastRTCPPort: 8003,
	}
	rtspHandler := service.NewRTSPHandler(&manager)
	rtspHandler.Mount(rtspServer)

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

// RTSP server handlers
