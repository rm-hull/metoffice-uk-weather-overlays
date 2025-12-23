package internal

import (
	"errors"
	"fmt"
	"image/color"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	metoffice "github.com/rm-hull/metoffice-uk-weather-overlays/internal/models/met_office"
	"github.com/rm-hull/metoffice-uk-weather-overlays/internal/png"
	"github.com/rm-hull/metoffice-uk-weather-overlays/internal/png/stage"
)

type Processor struct {
	startTime   time.Time
	endTime     time.Time
	rootDir     string
	poolSize    int
	maxJobs     int
	jobs        chan metoffice.File
	results     chan error
	client      DataHubClient
	files       []metoffice.File
	orderId     string
	fileIdRegex *regexp.Regexp
	pipelines   map[string][]png.PipelineStage
}

func NewDownloader(rootDir string, poolSize int, apiKey, orderId string) (*Processor, error) {
	if poolSize < 1 {
		return nil, errors.New("pool size must be at least 1")
	}
	startTime := time.Now()
	orderId = url.QueryEscape(orderId)
	client := NewDataHubClient(apiKey)
	resp, err := client.GetLatest(orderId, NewQueryParams("dataSpec", "1.1.0"))
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve order %s: %w", orderId, err)
	}

	log.Printf("Order %s contains %d files", orderId, len(resp.OrderDetails.Files))
	if len(resp.OrderDetails.Files) == 0 {
		return nil, errors.New("no files to download")
	}

	return &Processor{
		startTime:   startTime,
		rootDir:     rootDir,
		poolSize:    poolSize,
		maxJobs:     -1,
		jobs:        make(chan metoffice.File),
		results:     make(chan error),
		client:      client,
		files:       resp.OrderDetails.Files,
		orderId:     orderId,
		fileIdRegex: regexp.MustCompile(`(.*?)_ts(\d{1,2})_(\d{4})(\d{2})(\d{2})00`),
		pipelines: map[string][]png.PipelineStage{
			"total_precipitation_rate": {
				&stage.ReplaceColorStage{Tolerance: 50, Replace: color.White},
				&stage.GaussianBlurStage{Sigma: 1.0},
				&stage.ResampleStage{},
			},
			"cloud_amount_total": {
				&stage.ReplaceColorStage{Tolerance: 50, Replace: color.White},
				&stage.GreyscaleStage{},
				&stage.GaussianBlurStage{Sigma: 1.0},
				&stage.ResampleStage{},
			},
			// NoOp's
			"mean_sea_level_pressure": {},
			"temperature_at_surface":  {},
		},
	}, nil
}

// dispatchJobs sends files to the jobs channel for processing by workers.
// When maxJobs is greater than zero, it limits the number of jobs dispatched,
// hence set to -1 to dispatch all jobs.
func (p *Processor) DispatchJobs() {

	go func() {
		for n, file := range p.files {
			if p.maxJobs > 0 && n >= p.maxJobs {
				break
			}
			p.jobs <- file
		}
		close(p.jobs)
	}()
}

func (p *Processor) StartWorkers() {
	log.Printf("Starting downloading files with pool size: %d", p.poolSize)

	for i := range p.poolSize {
		go p.worker(i)
	}
}

func (p *Processor) worker(i int) {
	log.Printf("Worker %d started", i)
	for file := range p.jobs {
		p.results <- p.processFile(file)
	}
	log.Printf("Worker %d finished", i)
}

func (p *Processor) processFile(file metoffice.File) error {
	matches := p.fileIdRegex.FindStringSubmatch(file.FileId)
	if matches == nil {
		return nil
	}

	path, err := p.createPath(matches)
	if err != nil {
		return fmt.Errorf("failed to create path: %w", err)
	}

	hour, err := strconv.Atoi(matches[2])
	if err != nil {
		return fmt.Errorf("failed to convert %s to integer: %w", matches[2], err)
	}

	kind := matches[1]
	filename := fmt.Sprintf("%s/%02d.png", path, hour)

	// if the file already exists, skip processing
	if _, err := os.Stat(filename); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	params := NewQueryParams("dataSpec", "1.1.0")
	if kind == "cloud_amount_total" {
		params.Add("styleName", "iso_fill_bu_gn_30_100_pc")
	}

	inFile, err := p.client.GetLatestDataFile(p.orderId, file.FileId, params)
	if err != nil {
		return fmt.Errorf("failed to retrieve datafile %s for order %s: %w", file.FileId, p.orderId, err)
	}
	defer func() {
		_ = inFile.Close()
	}()

	tmpFile, err := os.CreateTemp(path, "download-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	cleanupTemp := true
	defer func() {
		_ = tmpFile.Close()
		if cleanupTemp {
			_ = os.Remove(tmpFile.Name())
		}
	}()

	pipeline := p.pipelines[kind]
	if pipeline == nil {
		return fmt.Errorf("no processing pipeline defined for data type %s", kind)
	}

	img, err := png.NewPngFromReader(inFile)
	if err != nil {
		return fmt.Errorf("failed to decode PNG from data file: %w", err)
	}

	if err := img.Pipeline(pipeline...); err != nil {
		return fmt.Errorf("failed to process image pipeline: %w", err)
	}

	if err := img.Write(tmpFile); err != nil {
		return fmt.Errorf("failed to write processed image to temporary file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file before rename: %w", err)
	}

	if err := os.Rename(tmpFile.Name(), filename); err != nil {
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	cleanupTemp = false // Successfully renamed, don't delete
	return nil
}

func (p *Processor) Wait() []error {
	waitFor := p.maxJobs
	if waitFor < 0 {
		waitFor = len(p.files)
	}
	log.Printf("Waiting for %d files to be downloaded and processed", waitFor)

	errors := make([]error, 0, 10)
	for range waitFor {
		err := <-p.results
		if err != nil {
			errors = append(errors, err)
		}
	}
	p.endTime = time.Now()
	elapsed := p.endTime.Sub(p.startTime)
	log.Printf("All files downloaded and processed in %s (errors=%d)", elapsed, len(errors))
	return errors
}

func (p *Processor) createPath(matches []string) (string, error) {
	path := filepath.Join(p.rootDir,
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
