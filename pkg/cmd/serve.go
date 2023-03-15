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

type ServeCmd struct {
	HTTPConfig config.HTTPConfig `kong:"embed,prefix='http-'"`
	RTSPConfig config.RTSPConfig `kong:"embed,prefix='rtsp-'"`
}

func (sc *ServeCmd) Run(cfg *config.Config) error {
	ctx, cancelCtx := context.WithCancel(context.Background())

	mux := http.NewServeMux()

	manager := stream.NewSkyEgressStreamManager()
	sh := service.NewSessionHandler(cfg, &manager)
	sh.Mount(mux)

	hh := service.NewHealthHandler(cfg)
	hh.Mount(mux)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", sc.HTTPConfig.Port),
		Handler: mux,
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}

	rtspServer := &gortsplib.Server{
		RTSPAddress:       fmt.Sprintf(":%d", sc.RTSPConfig.Port),
		UDPRTPAddress:     fmt.Sprintf(":%d", sc.RTSPConfig.UDPRTPPort),
		UDPRTCPAddress:    fmt.Sprintf(":%d", sc.RTSPConfig.UDPRTCPPort),
		MulticastIPRange:  sc.RTSPConfig.MulticastIPRange,
		MulticastRTPPort:  sc.RTSPConfig.MulticastRTPPort,
		MulticastRTCPPort: sc.RTSPConfig.MulticastRTCPPort,
	}
	rtspHandler := service.NewRTSPHandler(&manager)
	rtspHandler.Mount(rtspServer)

	go func() {
		fmt.Println("starting http server on", sc.HTTPConfig.Port)
		err := httpServer.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			fmt.Println("http server closed")
		} else if err != nil {
			fmt.Printf("error listening for http server %s\n", err)
		}
		cancelCtx()
	}()

	go func() {
		fmt.Println("starting rtsp server on", sc.RTSPConfig.Port)
		err := rtspServer.StartAndWait()
		if err != nil {
			fmt.Printf("error listening for rtsp server: %s\n", err)
		}
		cancelCtx()
	}()

	<-ctx.Done()

	return nil
}
