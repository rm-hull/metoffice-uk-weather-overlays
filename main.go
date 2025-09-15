package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/rm-hull/metoffice-uk-weather-overlays/internal"
)

func main() {

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	apiKey := os.Getenv("METOFFICE_DATAHUB_API_KEY")

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

		// Skip if the file exists
		if _, err := os.Stat(filename); err == nil || !os.IsNotExist(err) {
			continue
		}

		data, err := client.GetLatestDataFile(resp.OrderDetails.Order.OrderId, file.FileId)
		if err != nil {
			panic(err)
		}
		defer data.Close()

		outFile, err := os.Create(filename)
		if err != nil {
			panic(err)
		}
		defer outFile.Close()

		if kind == "total_precipitation_rate" {
			err = internal.SmoothPNG(data, outFile, 50, 1.0)
		} else {
			_, err = io.Copy(outFile, data)
		}
		if err != nil {
			panic(err)
		}
	}
}

func createPath(matches []string) (string, error) {
	path := fmt.Sprintf("data/%s/%s/%s/%s", matches[1], // type
		matches[3], // year
		matches[4], // month
		matches[5], // day
	)
	if err := os.MkdirAll(path, 0755); err != nil {
		return "", err
	}
	return path, nil
}
