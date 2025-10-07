package main

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/rm-hull/metoffice-uk-weather-overlays/cmd"
	"github.com/spf13/cobra"
)

func main() {
	var err error
	var rootPath string
	var port int
	var debug bool
	var poolSize int

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	rootCmd := &cobra.Command{
		Use:  "uk-weather-overlays",
		Long: `Met Office UK weather overlays`,
	}

	apiServerCmd := &cobra.Command{
		Use:   "api-server [--port <port>] [--debug]",
		Short: "Start HTTP API server",
		Run: func(_ *cobra.Command, _ []string) {
			cmd.ApiServer(rootPath, port, debug)
		},
	}

	apiServerCmd.Flags().IntVar(&port, "port", 8080, "Port to run HTTP server on")
	apiServerCmd.Flags().BoolVar(&debug, "debug", false, "Enable debugging (pprof) - WARING: do not enable in production")

	downloadCmd := &cobra.Command{
		Use:   "download",
		Short: "Initiate download",
		Run: func(_ *cobra.Command, _ []string) {
			if err := cmd.Download(rootPath, poolSize); err != nil {
				log.Fatalf("failed to download: %v", err)
			}
		},
	}
	downloadCmd.Flags().IntVar(&poolSize, "pool-size", 4, "Number of parallel downloads")

	rootCmd.PersistentFlags().StringVar(&rootPath, "root", "./data/datahub", "Path to root folder")
	rootCmd.AddCommand(apiServerCmd)
	rootCmd.AddCommand(downloadCmd)
	if err = rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
