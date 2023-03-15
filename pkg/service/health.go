package service

import (
	"net/http"
	"time"

	gosundheit "github.com/AppsFlyer/go-sundheit"
	healthhttp "github.com/AppsFlyer/go-sundheit/http"

	"github.com/treyhaknson/skyegress/pkg/config"
	"github.com/treyhaknson/skyegress/pkg/util"
)

type healthHandler struct {
	health gosundheit.Health
}

func NewHealthHandler(cfg *config.Config) healthHandler {
	gh := gosundheit.New()

	lkAuthCheck := util.NewLiveKitAuthCheck(
		cfg.LiveKitConfig.Host,
		cfg.LiveKitConfig.ApiKey,
		cfg.LiveKitConfig.ApiSecret,
	)

	err := gh.RegisterCheck(
		lkAuthCheck,
		gosundheit.InitialDelay(5*time.Second),
		gosundheit.ExecutionPeriod(10*time.Second),
	)
	if err != nil {
		panic(err)
	}

	return healthHandler{health: gh}
}

func (hh *healthHandler) Mount(mux *http.ServeMux) {
	handler := healthhttp.HandleHealthJSON(hh.health)
	mux.HandleFunc("/health", handler)
}
