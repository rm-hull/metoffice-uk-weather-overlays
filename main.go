package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"

	"github.com/Depado/ginprom"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/rm-hull/metoffice-uk-weather-overlays/internal"
	"github.com/rm-hull/metoffice-uk-weather-overlays/internal/png"
	healthcheck "github.com/tavsec/gin-healthcheck"
	"github.com/tavsec/gin-healthcheck/checks"
	hc_config "github.com/tavsec/gin-healthcheck/config"
)

func TestFetch() {

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	apiKey := os.Getenv("METOFFICE_DATAHUB_API_KEY")
	if apiKey == "" {
		log.Fatal("Error: METOFFICE_DATAHUB_API_KEY environment variable not set.")
	}

	client := internal.NewDataHubClient(apiKey)
	resp, err := client.GetLatest("o205748062845")
	if err != nil {
		panic(err)
	}

	re := regexp.MustCompile(`(.*?)_ts(\d{1,2})_(\d{4})(\d{2})(\d{2})00`)

	for _, file := range resp.OrderDetails.Files {
		matches := re.FindStringSubmatch(file.FileId)
		if matches == nil {
			continue
		}

		path, err := createPath(matches)
		if err != nil {
			panic(err)
		}

		hour, err := strconv.Atoi(matches[2])
		if err != nil {
			panic(err)
		}

		kind := matches[1]
		filename := fmt.Sprintf("%s/%02d.png", path, hour)

		if _, err := os.Stat(filename); err == nil {
			// File already exists, skip.
			continue
		} else if !os.IsNotExist(err) {
			// An unexpected error occurred (e.g., permissions).
			panic(err)
		}

		inFile, err := client.GetLatestDataFile(resp.OrderDetails.Order.OrderId, file.FileId)
		if err != nil {
			panic(err)
		}

		outFile, err := os.Create(filename)
		if err != nil {
			panic(err)
		}

		if kind == "total_precipitation_rate" {
			err = png.Smooth(inFile, outFile, 50, 1.0)
		} else {
			_, err = io.Copy(outFile, inFile)
		}
		if err != nil {
			panic(err)
		}
		if err := inFile.Close(); err != nil {
			log.Printf("failed to close data file: %v", err)
		}
		if err := outFile.Close(); err != nil {
			log.Printf("failed to close data file: %v", err)
		}
	}
}

func CreateAnimation() {

	dirPath := "data/datahub/temperature_at_surface/2025/09/15/"
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		panic(err)
	}

	files := make([]string, len(entries))
	for i, entry := range entries {
		fmt.Println(entry.Name()) // just the filename
		files[i] = dirPath + entry.Name()
	}

	apngBytes, err := png.Animate(files, 1.0)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile("data/temp.png", apngBytes, 0644)
	if err != nil {
		panic(err)
	}

}

func createPath(matches []string) (string, error) {
	path := fmt.Sprintf("data/datahub/%s/%s/%s/%s", matches[1], // type
		matches[3], // year
		matches[4], // month
		matches[5], // day
	)
	if err := os.MkdirAll(path, 0755); err != nil {
		return "", err
	}
	return path, nil
}

func Router(rootDir string, port int) {
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

	err := healthcheck.New(r, hc_config.DefaultConfig(), []checks.Check{})
	if err != nil {
		log.Fatalf("failed to initialize healthcheck: %v", err)
	}

	r.Static("/v1/metoffice/datahub", rootDir)

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Starting HTTP API Server on port %d...", port)
	err = r.Run(addr)
	log.Fatalf("HTTP API Server failed to start on port %d: %v", port, err)
}

func main() {
	Router("./data/datahub", 8080)
}
