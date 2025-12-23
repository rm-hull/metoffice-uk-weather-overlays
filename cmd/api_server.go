package cmd

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Depado/ginprom"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/rm-hull/godx"
	"github.com/rm-hull/metoffice-uk-weather-overlays/internal"
	healthcheck "github.com/tavsec/gin-healthcheck"
	"github.com/tavsec/gin-healthcheck/checks"
	hc_config "github.com/tavsec/gin-healthcheck/config"
)

const staticPathPrefix = "/v1/metoffice/datahub/"

var forecastPathRegexp = regexp.MustCompile(`^([^/]+)/(\d{4}/\d{2}/\d{2})/(\d{2})\.png$`)

// ApiServer starts an HTTP server to serve static files from rootDir on the given port.
// If debug is true, pprof endpoints are enabled.
func ApiServer(rootDir string, port int, debug bool) error {
	godx.GitVersion()
	godx.UserInfo()
	godx.EnvironmentVars()

	apiKey := os.Getenv("METOFFICE_DATAHUB_API_KEY")
	if apiKey == "" {
		return errors.New("environment variable METOFFICE_DATAHUB_API_KEY not set")
	}

	orderId := os.Getenv("METOFFICE_ORDER_ID")
	if orderId == "" {
		return errors.New("environment variable METOFFICE_ORDER_ID not set")
	}

	_, err := internal.StartCron(rootDir, apiKey, orderId)
	if err != nil {
		return err
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
		return fmt.Errorf("failed to initialize healthcheck: %v", err)
	}

	r.Static(staticPathPrefix, rootDir)

	// Global 404 handler for unmatched routes (including static file misses)
	r.NoRoute(func(c *gin.Context) {

		if strings.HasPrefix(c.Request.URL.Path, staticPathPrefix) {
			err := tryPreviousDaysForecast(c)
			if err == nil {
				return
			}
			log.Printf("Error handling previous day's forecast redirect: %v", err)
		}

		c.JSON(404, gin.H{
			"error": "Resource not found",
			"path":  c.Request.URL.Path,
		})
	})

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Starting HTTP API Server on port %d...", port)
	if err := r.Run(addr); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTP API Server failed to start on port %d: %v", port, err)
	}
	return nil
}

// tryPreviousDaysForecast attempts to handle requests for missing forecast files
// by redirecting to the previous day's forecast at the same hour + 24.
// For example, a request for /v1/metoffice/datahub/cloud_amount_total/2023/10/15/20.png
// would be redirected to /v1/metoffice/datahub/cloud_amount_total/2023/10/14/44.png
// if the original file is not found.
func tryPreviousDaysForecast(c *gin.Context) error {
	// Use regex to extract overlay, year, month, day, hour
	// Example path: /v1/metoffice/datahub/cloud_amount_total/2023/10/15/20.png
	trimmedPath := strings.TrimPrefix(c.Request.URL.Path, staticPathPrefix)
	matches := forecastPathRegexp.FindStringSubmatch(trimmedPath)
	if len(matches) != 4 {
		return fmt.Errorf("URL path does not match expected format: %s", trimmedPath)
	}
	overlay := matches[1]
	dt, err := time.Parse("2006/01/02", matches[2])
	if err != nil {
		return fmt.Errorf("invalid date format in URL: %v", err)
	}

	hour, err := strconv.Atoi(matches[3])
	if err != nil {
		return fmt.Errorf("invalid hour format in URL: %v", err)
	}

	// Subtract 1 day to get the previous day, then advance the hour by 24
	// to get the same time at the current date
	prevDay := dt.AddDate(0, 0, -1).Format("2006/01/02")
	prevHour := hour + 24

	if prevHour > 72 {
		return fmt.Errorf("calculated hour %d is out of range (0-72)", prevHour)
	}

	// Construct new URL and redirect
	newURL := fmt.Sprintf("%s%s/%s/%02d.png", staticPathPrefix, overlay, prevDay, prevHour)
	c.Redirect(http.StatusTemporaryRedirect, newURL)
	return nil
}
