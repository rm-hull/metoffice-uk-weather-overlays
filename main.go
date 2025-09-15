package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/rm-hull/metoffice-uk-weather-overlays/cmd"
	"github.com/spf13/cobra"
)

func main() {
	var err error
	var rootPath string
	var port int
	var debug bool

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	apiKey := os.Getenv("METOFFICE_DATAHUB_API_KEY")
	if apiKey == "" {
		log.Fatal("Error: METOFFICE_DATAHUB_API_KEY environment variable not set.")
	}

	rootCmd := &cobra.Command{
		Use:  "uk-weather-overlays",
		Long: `Met Office UK weather overlays`,
	}

	apiServerCmd := &cobra.Command{
		Use:   "api-server [--root <path>] [--port <port>] [--debug]",
		Short: "Start HTTP API server",
		Run: func(_ *cobra.Command, _ []string) {
			cmd.ApiServer(rootPath, port, debug)
		},
	}

	apiServerCmd.Flags().StringVar(&rootPath, "root", "./data/datahub", "Path to root folder")
	apiServerCmd.Flags().IntVar(&port, "port", 8080, "Port to run HTTP server on")
	apiServerCmd.Flags().BoolVar(&debug, "debug", false, "Enable debugging (pprof) - WARING: do not enable in production")

	rootCmd.AddCommand(apiServerCmd)
	if err = rootCmd.Execute(); err != nil {
		panic(err)
	}
}
