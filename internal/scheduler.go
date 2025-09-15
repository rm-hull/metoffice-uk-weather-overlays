package internal

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"

	"github.com/go-co-op/gocron/v2"
	"github.com/rm-hull/metoffice-uk-weather-overlays/internal/png"
)

var fileIdRegex = regexp.MustCompile(`(.*?)_ts(\d{1,2})_(\d{4})(\d{2})(\d{2})00`)

func NewScheduler(apiKey, orderId, rootDir string) (gocron.Scheduler, error) {

	if err := testFetch(apiKey, orderId, rootDir); err != nil {
		return nil, fmt.Errorf("initial run of job failed: %w", err)
	}

	scheduler, err := gocron.NewScheduler()
	if err != nil {
		return nil, fmt.Errorf("failed to create scheduler: %w", err)
	}

	_, err = scheduler.NewJob(
		gocron.DailyJob(1, gocron.NewAtTimes(
			gocron.NewAtTime(2, 00, 00),
			gocron.NewAtTime(3, 00, 00),
			gocron.NewAtTime(4, 00, 00),
			gocron.NewAtTime(5, 00, 00),
		)),
		gocron.NewTask(testFetch, apiKey, orderId, rootDir),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	scheduler.Start()
	return scheduler, nil
}

func testFetch(apiKey, orderId, rootDir string) error {

	client := NewDataHubClient(apiKey)
	resp, err := client.GetLatest(orderId)
	if err != nil {
		return fmt.Errorf("failed to retrieve order %s: %w", orderId, err)
	}

	createPath := func(matches []string) (string, error) {
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

	for _, file := range resp.OrderDetails.Files {
		matches := fileIdRegex.FindStringSubmatch(file.FileId)
		if matches == nil {
			continue
		}

		path, err := createPath(matches)
		if err != nil {
			return fmt.Errorf("failed to create path: %w", err)
		}

		hour, err := strconv.Atoi(matches[2])
		if err != nil {
			return fmt.Errorf("failed to convert %s to integer: %w", matches[2], err)
		}

		kind := matches[1]
		filename := fmt.Sprintf("%s/%02d.png", path, hour)

		if _, err := os.Stat(filename); err == nil {
			// File already exists, skip.
			continue
		} else if !os.IsNotExist(err) {
			// An unexpected error occurred (e.g., permissions).
			return err
		}

		inFile, err := client.GetLatestDataFile(resp.OrderDetails.Order.OrderId, file.FileId)
		if err != nil {
			return fmt.Errorf("failed to retrieve datafile %s for order %s: %w", file.FileId, orderId, err)
		}

		tmpFile, err := os.CreateTemp(path, "download-*.tmp")
		if err != nil {
			_ = inFile.Close()
			return fmt.Errorf("failed to create temporary file: %w", err)
		}

		var processingErr error
		if kind == "total_precipitation_rate" {
			processingErr = png.Smooth(inFile, tmpFile, 50, 1.0)
		} else {
			_, processingErr = io.Copy(tmpFile, inFile)
		}

		// Always close files and check for errors
		inFileCloseErr := inFile.Close()
		tmpFileCloseErr := tmpFile.Close()

		if processingErr != nil {
			_ = os.Remove(tmpFile.Name())
			return fmt.Errorf("failed to process data file: %w", processingErr)
		}
		if inFileCloseErr != nil {
			_ = os.Remove(tmpFile.Name())
			return fmt.Errorf("failed to close input file: %w", inFileCloseErr)
		}
		if tmpFileCloseErr != nil {
			_ = os.Remove(tmpFile.Name())
			return fmt.Errorf("failed to close temporary file: %w", tmpFileCloseErr)
		}

		if err := os.Rename(tmpFile.Name(), filename); err != nil {
			_ = os.Remove(tmpFile.Name())
			return fmt.Errorf("failed to rename temporary file: %w", err)
		}
	}

	return nil
}
