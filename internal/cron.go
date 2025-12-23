package internal

import (
	"log"

	"github.com/robfig/cron/v3"
)

func StartCron(rootDir, apiKey, orderId string) (*cron.Cron, error) {
	poolSize := 1
	schedule := "30 4,5,6 * * *"

	c := cron.New()

	log.Printf("Starting CRON job to download files (schedule=%s)", schedule)
	_, err := c.AddFunc(schedule, func() {
		downloader, err := NewDownloader(rootDir, poolSize, apiKey, orderId)
		if err != nil {
			log.Printf("Failed to create downloader: %v", err)
		}

		downloader.StartWorkers()
		downloader.DispatchJobs()
		errors := downloader.Wait()
		if len(errors) > 0 {
			log.Printf("Errors occurred: %v", errors)
		}
	})

	if err != nil {
		return nil, err
	}

	c.Start()
	return c, nil
}
