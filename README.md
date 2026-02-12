# Asana Extractor

A production-ready Go application that extracts users and projects from the Asana API with robust rate limiting, retry logic, and configurable scheduling.

## Features

- ✅ **Modular rate limiting** - Token bucket algorithm enforcing 150 requests/minute (free tier)
- ✅ **Smart retry logic** - Exponential backoff with jitter and `Retry-After` header support
- ✅ **Pagination support** - Handles high-volume workspaces with automatic pagination
- ✅ **Configurable scheduling** - Cron-based scheduling for periodic extraction
- ✅ **Individual JSON output** - Each user/project saved as separate JSON file
- ✅ **Graceful error handling** - Comprehensive logging and error recovery
- ✅ **Thread-safe** - Concurrent request management
- ✅ **Extensible design** - Interface-based architecture for easy customization

## Installation

### Prerequisites

- Go 1.21 or higher
- Asana API token (Personal Access Token)
- Asana workspace ID or name

### Setup

1. Clone or navigate to the repository:
```bash
cd /asana-extractor
```

2. Install dependencies:
```bash
go mod download
```

3. Configure environment variables:
```bash
cp .env.example .env
# Edit .env with your Asana credentials
```

## Configuration

The application is configured via environment variables. See [.env.example](.env.example) for all available options.

### Required Variables

```bash
ASANA_TOKEN=your-personal-access-token
ASANA_WORKSPACE=your-workspace-id-or-name
```

### Optional Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SCHEDULE_CRON` | `*/5 * * * *` | Cron expression for scheduling |
| `OUTPUT_DIR` | `./output` | Directory for JSON output files |
| `REQUESTS_PER_MINUTE` | `150` | Rate limit (free tier) |
| `MAX_CONCURRENT_READ` | `50` | Max concurrent GET requests |
| `MAX_CONCURRENT_WRITE` | `15` | Max concurrent POST/PUT/PATCH/DELETE |
| `HTTP_TIMEOUT` | `30s` | HTTP client timeout |
| `MAX_RETRIES` | `5` | Maximum retry attempts |
| `INITIAL_BACKOFF` | `1s` | Initial retry backoff |
| `MAX_BACKOFF` | `60s` | Maximum retry backoff |

### Scheduling Examples

```bash
# Every 5 minutes (default)
SCHEDULE_CRON="*/5 * * * *"

# Every 30 minutes
SCHEDULE_CRON="*/30 * * * *"

# Every hour
SCHEDULE_CRON="0 * * * *"

# Every day at 2 AM
SCHEDULE_CRON="0 2 * * *"
```

**Note**: For intervals less than 1 minute (e.g., every 30 seconds), you'll need to implement a custom scheduler. The current cron-based implementation supports minute-level granularity.

## Usage

### Build

```bash
make build
# or
go build -o bin/asana-extractor ./cmd/extractor
```

### Run

```bash
# Using make
make run

# Or directly
go run ./cmd/extractor/main.go

# Or using the binary
./bin/asana-extractor
```

### Output

The extractor creates the following directory structure:

```
output/
├── users/
│   ├── 1234567890.json
│   ├── 9876543210.json
│   └── ...
└── projects/
    ├── 1111111111.json
    ├── 2222222222.json
    └── ...
```

Each file contains the complete JSON representation of a user or project.

## Rate Limiting

The application implements Asana's rate limiting requirements:

- **Standard rate limit**: 150 requests/minute for free tier
- **Concurrent requests**: 50 GET, 15 POST/PUT/PATCH/DELETE
- **Retry-After header**: Automatically respected on 429 responses
- **Exponential backoff**: With jitter to avoid thundering herd

### Handling 429 Errors

When the API returns a `429 Too Many Requests` response, the application:

1. Reads the `Retry-After` header
2. Waits for the specified duration
3. Retries the request (up to `MAX_RETRIES` times)
4. Uses exponential backoff if `Retry-After` is not specified

## Testing

```bash
# Run all tests
make test
# or
go test ./...

# Run with race detection
go test -race ./...

# Run with coverage
go test -cover ./...
```

## Architecture

```
cmd/extractor/          - Main application entry point
pkg/
  ├── ratelimit/        - Rate limiting with token bucket
  ├── retry/            - Retry logic with exponential backoff
  ├── client/           - HTTP client with rate limiting & retry
  ├── asana/            - Asana API client
  ├── config/           - Configuration management
  ├── scheduler/        - Cron-based scheduler
  ├── storage/          - JSON file storage
  └── extractor/        - Extraction orchestration
```

## Troubleshooting

### Rate Limit Errors

If you're hitting rate limits frequently:

1. Reduce `REQUESTS_PER_MINUTE` to be more conservative
2. Increase `INITIAL_BACKOFF` and `MAX_BACKOFF`
3. Check if you have a paid Asana plan with higher limits

### Large Workspaces

For workspaces with >1000 users/projects:

- The pagination is automatic and handles large datasets
- Extraction may take longer due to rate limiting
- Consider adjusting the schedule to run less frequently

### Network Issues

The application automatically retries on network errors with exponential backoff. If issues persist:

1. Check your internet connection
2. Verify the Asana API is accessible
3. Increase `HTTP_TIMEOUT` if requests are timing out

## License

MIT
