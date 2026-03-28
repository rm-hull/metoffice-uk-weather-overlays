package internal

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/robfig/cron/v3"
)

var forecastPathRegexp = regexp.MustCompile(`^([^/]+)/(\d{4}/\d{2}/\d{2})/(\d{2})\.webp$`)

func StartCron(rootDir, apiKey, orderId string) (*cron.Cron, error) {
	c := cron.New()

	if err := ScheduleDownloadJob(c, rootDir, apiKey, orderId); err != nil {
		return nil, err
	}

	if err := ScheduleCleanupJob(c, rootDir); err != nil {
		return nil, err
	}

	c.Start()
	return c, nil
}

func ScheduleDownloadJob(c *cron.Cron, rootDir, apiKey, orderId string) error {
	poolSize := 1
	schedule := "30 4,5,6 * * *"

	log.Printf("Starting CRON job to download files (schedule=%s)", schedule)
	_, err := c.AddFunc(schedule, func() {
		downloader, err := NewDownloader(rootDir, poolSize, apiKey, orderId)
		if err != nil {
			log.Printf("Failed to create downloader: %v", err)
			return
		}

		downloader.StartWorkers()
		downloader.DispatchJobs()
		errors := downloader.Wait()
		if len(errors) > 0 {
			log.Printf("Errors occurred: %v", errors)
		}
	})

	return err
}

func ScheduleCleanupJob(c *cron.Cron, rootDir string) error {
	schedule := "0 1 * * *"
	log.Printf("Starting CRON job to cleanup old overflow forecasts (schedule=%s)", schedule)
	_, err := c.AddFunc(schedule, func() {
		cleanupOldOverflowForecasts(rootDir)
	})
	return err
}

func cleanupOldOverflowForecasts(rootDir string) {
	now := time.Now()
	cutoff := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -7)

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("cleanup: error accessing %q: %v", path, err)
			return nil
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(rootDir, path)
		if err != nil {
			log.Printf("cleanup: could not get relative path for %q: %v", path, err)
			return nil
		}

		// Convert to slash to ensure regex works on all platforms
		matches := forecastPathRegexp.FindStringSubmatch(filepath.ToSlash(rel))
		if len(matches) != 4 {
			return nil
		}

		dateStr := matches[2]
		hourStr := matches[3]

		forecastDate, err := time.ParseInLocation("2006/01/02", dateStr, now.Location())
		if err != nil {
			log.Printf("cleanup: could not parse date from path %q: %v", path, err)
			return nil
		}

		if !forecastDate.Before(cutoff) {
			return nil
		}

		hour, err := strconv.Atoi(hourStr)
		if err != nil {
			log.Printf("cleanup: could not parse hour from path %q: %v", path, err)
			return nil
		}

		if hour >= 24 {
			log.Printf("Deleting old overflow forecast: %s", path)
			if err := os.Remove(path); err != nil {
				log.Printf("Failed to delete %s: %v", path, err)
			}
		}

		return nil
	})

	if err != nil {
		log.Printf("Cleanup job failed to walk directory: %v", err)
	}
}
