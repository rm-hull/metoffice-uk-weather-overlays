package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"

	"github.com/rm-hull/metoffice-uk-weather-overlays/internal"
	"github.com/rm-hull/metoffice-uk-weather-overlays/internal/png"
)

func Download(rootDir string) error {
	internal.GitVersion()
	internal.UserInfo()
	internal.EnvironmentVars()

	apiKey := os.Getenv("METOFFICE_DATAHUB_API_KEY")
	if apiKey == "" {
		return errors.New("environment variable METOFFICE_DATAHUB_API_KEY not set")
	}

	orderId := os.Getenv("METOFFICE_ORDER_ID")
	if orderId == "" {
		return errors.New("environment variable METOFFICE_ORDER_ID not set")
	}

	fileIdRegex := regexp.MustCompile(`(.*?)_ts(\d{1,2})_(\d{4})(\d{2})(\d{2})00`)
	client := internal.NewDataHubClient(apiKey)
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
        defer func() {
            if err := inFile.Close(); err != nil {
                // Log the error, but don't return it as it might mask a more important error
                fmt.Fprintf(os.Stderr, "warning: failed to close input file %s: %v\n", file.FileId, err)
            }
        }()

		tmpFile, err := os.CreateTemp(path, "download-*.tmp")
		if err != nil {
			return fmt.Errorf("failed to create temporary file: %w", err)
		}
		defer func() {
			_ = tmpFile.Close()
			_ = os.Remove(tmpFile.Name())
		}()

		var processingErr error
		if kind == "total_precipitation_rate" {
			processingErr = png.Smooth(inFile, tmpFile, 50, 1.0)
		} else {
			_, processingErr = io.Copy(tmpFile, inFile)
		}

		if processingErr != nil {
			return fmt.Errorf("failed to process data file: %w", processingErr)
		}

		// Close tmpFile before renaming to ensure all data is flushed
		if err := tmpFile.Close(); err != nil {
			return fmt.Errorf("failed to close temporary file before rename: %w", err)
		}

		if err := os.Rename(tmpFile.Name(), filename); err != nil {
			return fmt.Errorf("failed to rename temporary file: %w", err)
		}
		// If rename is successful, cancel the deferred removal of the temporary file
		// by re-deferring a no-op or setting a flag. For simplicity, we'll just let the defer run
		// and handle the error if the file is already gone (which it will be).
		// A more robust solution would involve a flag or a custom defer stack.
		// For now, the os.Remove will just fail silently if the file is already gone.
		// This is acceptable for a temporary file that has been renamed.
	}

	return nil
}
