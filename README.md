# Met Office UK Weather Overlays

This project provides tools to download weather overlay data from the Met Office DataHub API, process it, and serve it via an HTTP API. It's designed to fetch specific weather imagery, apply image processing (like smoothing), and make it available for consumption, potentially for mapping applications or visualizations.

## Features

*   **Data Download:** Fetches the latest weather overlay data from the Met Office DataHub API.
*   **Image Processing:** Includes functionality to smooth certain types of weather images (e.g., `total_precipitation_rate`).
*   **HTTP API Server:** Serves the processed weather overlay images as static files.
*   **Monitoring:** Integrates Prometheus metrics and pprof for performance profiling.

## Getting Started

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes.

### Prerequisites

*   Go (version 1.25.1 or higher)
*   A Met Office DataHub API Key
*   A Met Office DataHub Order ID

### Installation

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/rm-hull/metoffice-uk-weather-overlays.git
    cd metoffice-uk-weather-overlays
    ```

2.  **Set up environment variables:**
    Create a `.env` file in the project root directory based on `.env.example`:
    ```sh
    METOFFICE_DATAHUB_API_KEY=your_api_key_here
    METOFFICE_ORDER_ID=your_order_id_here
    ```
    Replace `your_api_key_here` and `your_order_id_here` with your actual Met Office DataHub credentials.

3.  **Download Go modules:**
    ```bash
    go mod tidy
    ```

## Usage

The project provides two main commands: `api-server` and `download`.

### 1. `download` command

This command fetches the latest weather overlay data from the Met Office DataHub API and saves it to your local filesystem.

```bash
go run main.go download
```

**Options:**
*   `--root <path>`: Specifies the root directory where data will be stored. Defaults to `./data/datahub`.

**Example:**
```bash
go run main.go download --root /var/weather_data
```

### 2. `api-server` command

This command starts an HTTP server that serves the downloaded weather overlay images. It also exposes Prometheus metrics and pprof endpoints (if debug is enabled).

```bash
go run main.go api-server
```

**Options:**
*   `--port <port>`: Specifies the port to run the HTTP server on. Defaults to `8080`.
*   `--debug`: Enables pprof debugging endpoints. **WARNING: Do not enable in production.**

**Example:**
```bash
go run main.go api-server --port 8000 --debug
```

Once the server is running, you can access the static files at `/v1/metoffice/datahub`. For example, if your `--root` is `./data/datahub` and you've downloaded data, you might access an image at `http://localhost:8080/v1/metoffice/datahub/total_precipitation_rate/2025/09/25/00.png`.

## Project Structure

*   `cmd/`: Contains the main logic for the `api-server` and `download` commands.
*   `internal/`: Houses internal packages for core functionalities:
    *   `datahub/`: Met Office DataHub API client.
    *   `debug/`: Debugging utilities (version info, environment vars).
    *   `models/met_office/`: Go structs for Met Office API responses.
    *   `png/`: Image processing utilities (animate, smooth).
*   `data/`: Default directory for downloaded weather data.

## Dependencies

The project uses the following key Go modules:

*   `github.com/gin-gonic/gin`: HTTP web framework.
*   `github.com/joho/godotenv`: Loads environment variables from a `.env` file.
*   `github.com/spf13/cobra`: Commander for modern CLI interactions.
*   `github.com/anthonynsimon/bild`: Image processing library.
*   `github.com/kettek/apng`: APNG (Animated PNG) encoder/decoder.
*   `github.com/Depado/ginprom`: Prometheus metrics for Gin.
*   `github.com/gin-contrib/pprof`: pprof integration for Gin.
*   `github.com/tavsec/gin-healthcheck`: Health check endpoints for Gin.

## Building

To build the executable:

```bash
go build -o uk-weather-overlays main.go
```

You can then run the commands directly:

```bash
./uk-weather-overlays download
./uk-weather-overlays api-server
```

## Testing

Currently, there are no automated tests implemented. (This is an area for future improvement.)

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details.
