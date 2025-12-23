package cmd

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/rm-hull/godx"
	"github.com/rm-hull/metoffice-uk-weather-overlays/internal"
)

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

	downloader, err := internal.NewDownloader(rootDir, poolSize, apiKey, orderId)
	if err != nil {
		return err
	}

	downloader.StartWorkers()
	downloader.DispatchJobs()
	errors := downloader.Wait()

	if len(errors) > 0 {
		for _, err := range errors {
			log.Printf("Error: %v", err)
		}
		return fmt.Errorf("%d error(s) occurred", len(errors))
	}

	return nil
}
