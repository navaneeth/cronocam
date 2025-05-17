# CronoCam

CronoCam is a command-line tool to automatically backup images to Google Photos. It supports all standard image formats including HEIC and can be run on a schedule to continuously backup new photos from a directory.

## Features

- Recursive directory scanning for images
- Supports all standard image formats (jpg, png, gif, heic, etc.)
- SQLite database to track uploaded files
- OAuth2 authentication with Google Photos API
- Resumable uploads support
- Headless operation support (ideal for Raspberry Pi)
- Configurable via YAML file or environment variables

## Setup

1. Create a Google Cloud Project and enable Google Photos API
2. Create OAuth 2.0 credentials and download the client configuration
3. Place the downloaded credentials in `config/credentials.json`
4. (Optional) Configure settings in `config.yaml`

## Configuration

The program can be configured in multiple ways:

1. Using a YAML config file (`config.yaml`):
   ```yaml
   credentials_path: "config/credentials.json"
   database_path: "data/uploads.db"
   chunk_size: 5242880  # 5MB
   max_retries: 3
   rate_limit:
     requests_per_second: 5
     max_burst: 10
   ```

2. Using environment variables:
   ```bash
   PHOTOS_CREDENTIALS_PATH="config/credentials.json"
   PHOTOS_DATABASE_PATH="data/uploads.db"
   PHOTOS_CHUNK_SIZE=5242880
   PHOTOS_MAX_RETRIES=3
   PHOTOS_RATE_LIMIT_REQUESTS_PER_SECOND=5
   PHOTOS_RATE_LIMIT_MAX_BURST=10
   ```

3. Default values will be used if no configuration is provided

Configuration files can be placed in:
- Current directory: `./config.yaml`
- User config directory: `~/.config/photos-uploader/config.yaml`

## Usage

```bash
# First time setup (will open browser for OAuth)
./cronocam setup

# Upload images from a directory
./cronocam upload /path/to/photos/directory
```

## Building

```bash
go build -o cronocam
```

## Advanced Configuration

- `chunk_size`: Size of upload chunks in bytes. Increase for faster uploads on good connections.
- `max_retries`: Number of retry attempts for failed uploads.
- `rate_limit.requests_per_second`: Maximum API requests per second to avoid quota issues.
- `rate_limit.max_burst`: Maximum number of requests allowed in a burst.
