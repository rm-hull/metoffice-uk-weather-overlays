package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/rm-hull/metoffice-uk-weather-overlays/internal"
	"github.com/rm-hull/metoffice-uk-weather-overlays/internal/png"
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

		data, err := client.GetLatestDataFile(resp.OrderDetails.Order.OrderId, file.FileId)
		if err != nil {
			panic(err)
		}

		outFile, err := os.Create(filename)
		if err != nil {
			panic(err)
		}

		if kind == "total_precipitation_rate" {
			err = png.Smooth(data, outFile, 50, 1.0)
		} else {
			_, err = io.Copy(outFile, data)
		}
		if err != nil {
			panic(err)
		}
		data.Close()
		outFile.Close()
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

func Router() {
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	r.Static("/v1/metoffice/datahub", "./data/datahub")

	_ = r.Run()
}

func main() {
	TestFetch()
}
