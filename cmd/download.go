package cmd

import (
	"errors"
	"fmt"
	"image/color"
	"log"
	"os"
	"regexp"
	"strconv"

	"github.com/rm-hull/godx"
	"github.com/rm-hull/metoffice-uk-weather-overlays/internal"
	metoffice "github.com/rm-hull/metoffice-uk-weather-overlays/internal/models/met_office"
	"github.com/rm-hull/metoffice-uk-weather-overlays/internal/png"
	"github.com/rm-hull/metoffice-uk-weather-overlays/internal/png/stage"
)

var pipelines map[string][]png.PipelineStage
var fileIdRegex *regexp.Regexp

func init() {
	pipelines = map[string][]png.PipelineStage{
		"total_precipitation_rate": {
			&stage.ReplaceColorStage{Tolerance: 50, Replace: color.White},
			&stage.GaussianBlurStage{Sigma: 1.0},
			&stage.ResampleStage{},
		},
		"cloud_amount_total": {
			&stage.ReplaceColorStage{Tolerance: 250, Replace: color.NRGBA{R: 0, G: 0xff, B: 0, A: 0xff}},
			&stage.GreyscaleStage{},
			&stage.GaussianBlurStage{Sigma: 1.0},
			&stage.ResampleStage{},
		},
		// NoOp's
		"mean_sea_level_pressure": {},
		"temperature_at_surface":  {},
	}

	fileIdRegex = regexp.MustCompile(`(.*?)_ts(\d{1,2})_(\d{4})(\d{2})(\d{2})00`)
}

func createPath(rootDir string, matches []string) (string, error) {
	path := fmt.Sprintf("%s/%s/%s/%s/%s", rootDir,
		matches[1], // type
		matches[3], // year
		matches[4], // month
		matches[5], // day
	)
	if err := os.MkdirAll(path, 0755); err != nil {
		return "", err
	}
	return path, nil
}

func Download(rootDir string, poolSize int) error {
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

	client := internal.NewDataHubClient(apiKey)
	resp, err := client.GetLatest(orderId, internal.NewQueryParams("dataSpec", "1.1.0"))
	if err != nil {
		return fmt.Errorf("failed to retrieve order %s: %w", orderId, err)
	}

	log.Printf("Order %s contains %d files", orderId, len(resp.OrderDetails.Files))

	if len(resp.OrderDetails.Files) == 0 {
		log.Printf("No files to download")
		return nil
	}

	log.Printf("Starting downloading files with pool size: %d", poolSize)

	// Use metoffice.File for jobs
	jobs := make(chan metoffice.File)
	results := make(chan error)

	// Worker function
	worker := func() {
		for file := range jobs {
			matches := fileIdRegex.FindStringSubmatch(file.FileId)
			if matches == nil {
				results <- nil
				continue
			}

			path, err := createPath(rootDir, matches)
			if err != nil {
				results <- fmt.Errorf("failed to create path: %w", err)
				continue
			}

			hour, err := strconv.Atoi(matches[2])
			if err != nil {
				results <- fmt.Errorf("failed to convert %s to integer: %w", matches[2], err)
				continue
			}

			kind := matches[1]
			filename := fmt.Sprintf("%s/%02d.png", path, hour)

			if _, err := os.Stat(filename); err == nil {
				// File already exists, skip.
				results <- nil
				continue
			} else if !os.IsNotExist(err) {
				// An unexpected error occurred (e.g., permissions).
				results <- err
				continue
			}

			params := internal.NewQueryParams("dataSpec", "1.1.0")
			if kind == "cloud_amount_total" {
				params.Add("styleName", "iso_fill_bu_gn_30_100_pc")
			}
			inFile, err := client.GetLatestDataFile(resp.OrderDetails.Order.OrderId, file.FileId, params)
			if err != nil {
				results <- fmt.Errorf("failed to retrieve datafile %s for order %s: %w", file.FileId, orderId, err)
				continue
			}

			tmpFile, err := os.CreateTemp(path, "download-*.tmp")
			if err != nil {
				_ = inFile.Close()
				results <- fmt.Errorf("failed to create temporary file: %w", err)
				continue
			}

			pipeline := pipelines[kind]
			if pipeline == nil {
				_ = inFile.Close()
				_ = tmpFile.Close()
				_ = os.Remove(tmpFile.Name())
				results <- fmt.Errorf("no processing pipeline defined for data type %s", kind)
				continue
			}

			img, err := png.NewPngFromReader(inFile)
			if err != nil {
				_ = inFile.Close()
				_ = tmpFile.Close()
				_ = os.Remove(tmpFile.Name())
				results <- fmt.Errorf("failed to decode PNG from data file: %w", err)
				continue
			}

			if err := img.Pipeline(pipeline...); err != nil {
				_ = inFile.Close()
				_ = tmpFile.Close()
				_ = os.Remove(tmpFile.Name())
				results <- fmt.Errorf("failed to process image pipeline: %w", err)
				continue
			}

			if err := img.Write(tmpFile); err != nil {
				_ = inFile.Close()
				_ = tmpFile.Close()
				_ = os.Remove(tmpFile.Name())
				results <- fmt.Errorf("failed to write processed image to temporary file: %w", err)
				continue
			}

			if err := tmpFile.Close(); err != nil {
				_ = inFile.Close()
				_ = os.Remove(tmpFile.Name())
				results <- fmt.Errorf("failed to close temporary file before rename: %w", err)
				continue
			}

			if err := os.Rename(tmpFile.Name(), filename); err != nil {
				_ = inFile.Close()
				_ = os.Remove(tmpFile.Name())
				results <- fmt.Errorf("failed to rename temporary file: %w", err)
				continue
			}

			_ = inFile.Close()
			results <- nil
		}
	}

	// Start workers
	for range poolSize {
		go worker()
	}

	// Send jobs
	go func() {
		for _, file := range resp.OrderDetails.Files {
			jobs <- file
		}
		close(jobs)
	}()

	// Wait for all results
	var firstErr error
	for i := 0; i < len(resp.OrderDetails.Files); i++ {
		err := <-results
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if firstErr != nil {
		return firstErr
	}

	return nil
}
