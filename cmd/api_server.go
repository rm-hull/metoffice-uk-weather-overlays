package cmd

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Depado/ginprom"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/rm-hull/godx"
	healthcheck "github.com/tavsec/gin-healthcheck"
	"github.com/tavsec/gin-healthcheck/checks"
	hc_config "github.com/tavsec/gin-healthcheck/config"
)

func ApiServer(rootDir string, port int, debug bool) {
	godx.GitVersion()
	godx.UserInfo()
	godx.EnvironmentVars()

	r := gin.New()

	prometheus := ginprom.New(
		ginprom.Engine(r),
		ginprom.Path("/metrics"),
		ginprom.Ignore("/healthz"),
	)

	r.Use(
		gin.Recovery(),
		gin.LoggerWithWriter(gin.DefaultWriter, "/healthz", "/metrics"),
		prometheus.Instrument(),
	)

	if debug {
		log.Println("WARNING: pprof endpoints are enabled and exposed. Do not run with this flag in production.")
		pprof.Register(r)
	}

	err := healthcheck.New(r, hc_config.DefaultConfig(), []checks.Check{})
	if err != nil {
		log.Fatalf("failed to initialize healthcheck: %v", err)
	}

	r.Static("/v1/metoffice/datahub", rootDir)

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Starting HTTP API Server on port %d...", port)
	if err := r.Run(addr); err != nil && err != http.ErrServerClosed {
		log.Fatalf("HTTP API Server failed to start on port %d: %v", port, err)
	}
}
