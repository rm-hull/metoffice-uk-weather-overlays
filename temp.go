package main

import (
	"fmt"
	"log"
	"os"

	imageprocessing "github.com/rm-hull/metoffice-uk-weather-overlays/internal/imageprocessing"
)

func CreateAnimation() {

	dirPath := "data/datahub/temperature_at_surface/2025/09/15/"
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		log.Fatal(err)
	}

	files := make([]string, len(entries))
	for i, entry := range entries {
		fmt.Println(entry.Name()) // just the filename
		files[i] = dirPath + entry.Name()
	}

	apngBytes, err := imageprocessing.Animate(files, 1.0)
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile("data/temp.png", apngBytes, 0644)
	if err != nil {
		log.Fatal(err)
	}
}
