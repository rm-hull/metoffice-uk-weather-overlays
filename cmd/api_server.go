package cmd

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Depado/ginprom"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/rm-hull/metoffice-uk-weather-overlays/internal"
	healthcheck "github.com/tavsec/gin-healthcheck"
	"github.com/tavsec/gin-healthcheck/checks"
	hc_config "github.com/tavsec/gin-healthcheck/config"
)

func ApiServer(rootDir string, port int, debug bool) {

	apiKey := os.Getenv("METOFFICE_DATAHUB_API_KEY")
	if apiKey == "" {
		log.Fatal("Error: METOFFICE_DATAHUB_API_KEY environment variable not set.")
	}

	orderId := os.Getenv("METOFFICE_ORDER_ID")
	if orderId == "" {
		log.Fatal("Error: METOFFICE_ORDER_ID environment variable not set.")
	}

	sched, err := internal.NewScheduler(apiKey, orderId, rootDir)
	if err != nil {
		log.Fatal(err)
	}

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

	err = healthcheck.New(r, hc_config.DefaultConfig(), []checks.Check{})
	if err != nil {
		log.Fatalf("failed to initialize healthcheck: %v", err)
	}

	r.Static("/v1/metoffice/datahub", rootDir)

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Starting HTTP API Server on port %d...", port)
	if err := r.Run(addr); err != nil && err != http.ErrServerClosed {
		log.Fatalf("HTTP API Server failed to start on port %d: %v", port, err)
	}

	err = sched.Shutdown()
	if err != nil {
		log.Fatalf("failed to shutdown scheduler: %v", err)
	}
}
