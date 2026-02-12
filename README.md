# Asana Extractor

A production-ready Go application that extracts users and projects from the Asana API. Built with a focus on resiliency, it features a **concurrent actor-based architecture**, custom token-bucket rate limiting, and a high-precision cron-based scheduler.

## 🚀 Step-by-Step Quick Start

1.  **Clone & Navigate**:
    ```bash
    git clone <your-repo-url>
    cd asana-extractor
    ```

2.  **Environment Setup**:
    ```bash
    cp .env.example .env
    ```
    Open `.env` and provide your `ASANA_TOKEN` and `ASANA_WORKSPACE`.

3.  **Install & Test**:
    ```bash
    make deps
    make test
    ```

4.  **Build & Run**:
    ```bash
    make build
    ./bin/asana-extractor
    ```

---

## 🐳 Docker & Makefile Usage

This project includes a **multi-stage Dockerfile** for optimized, small-footprint production images and a **Makefile** to simplify common tasks.

### Prerequisites
* [Docker](https://docs.docker.com/get-docker/) installed and running.
* A valid `.env` file in the root directory.

### Quick Commands

| Command | Description |
| :--- | :--- |
| `make docker-build` | Builds the extractor image using Go 1.25. |
| `make docker-run` | Starts the container, loads `.env`, and mounts `./output`. |
| `make docker-stop` | Stops and removes the running container. |
| `make docker-logs` | Tails the logs of the running extractor. |
| `make docker-clean` | Deletes the local Docker image. |

### Running with Persistent Storage
To ensure your extracted data is saved on your host machine, `make docker-run` automatically mounts your local `./output` directory to the container.

```bash
# 1. Build the image
make docker-build
```

```bash
# 2. Start the background service
make docker-run
```

```bash
# 3. Stop the background service
make docker-stop
```

```bash
# 4. View the logs
make docker-logs
```

```bash
# 5. Remove the Docker image
make docker-clean
```

---

## 🏗 Architecture: The Concurrent Actor Pattern

To ensure high-performance throughput while maintaining strict thread safety, the application utilizes a **Concurrent Actor Pattern**.



### How it Works:
* **The Workers (Fetchers/Writers)**: Separate goroutines are spawned for User and Project categories. These workers handle fetching paginated data from the API and performing atomic writes to the filesystem.
* **The Channel (Communication)**: Workers communicate with the state manager using a buffered channel. They send "update functions" across the channel rather than modifying shared memory.
* **The Actor (State Manager)**: A single dedicated goroutine acts as the "Actor." It is the **only** entity authorized to modify the internal `Stats` struct, eliminating data races and the need for Mutex locks.
* **Orchestration**: A coordination layer separates fatal API errors from non-fatal storage errors, ensuring the scheduler can report accurately on the status of each run.

---

## 🛠 Main Functionality

The application operates as a scheduled service that performs the following:

1.  **Identity Discovery**: Fetches all accessible users and projects within the configured workspace.
2.  **Concurrent Extraction**: Processes resources using an internal queue that respects Asana's rate limits.
3.  **Atomic Persistence**: Saves each resource as an individual JSON file. It uses a **Write-and-Rename** strategy to ensure files are never corrupted if the process is interrupted.

---

## 🌟 Key Features

* **Actor-Based Concurrency**: Thread-safe state management without Mutex contention.
* **6-Field Cron Scheduling**: High-precision scheduling with second-level granularity (e.g., `0 */5 * * * *`).
* **Cursor-Based Pagination**: Automatically handles large datasets by following Asana's `next_page` tokens.
* **Smart Retries**: Exponential backoff with jitter and full support for the `Retry-After` header.
* **Graceful Shutdown**: Listens for OS signals (`SIGINT`, `SIGTERM`) to stop the scheduler cleanly after current file writes complete.

---

## ⚠️ Limits & Constraints

| Category | Limit | Description |
| :--- | :--- | :--- |
| **Throughput** | 150 RPM | Optimized for Asana Free Tier; adjustable via `REQUESTS_PER_MINUTE`. |
| **Concurrency** | 50 Read / 15 Write | Enforced via internal semaphores to prevent API-side connection rejection. |
| **Pagination** | 100 items/page | Set to Asana's maximum page size to minimize network round-trips. |
| **Scheduling** | 1 Second | Minimum supported interval between extractions due to 6-field cron parser. |
| **Storage** | File System | Extraction speed is bounded by disk IOPS when writing thousands of small JSON files. |

---

## ⚙️ Full Configuration Guide

Available variables in your `.env` file:

### Required Variables
| Variable | Example | Description |
| :--- | :--- | :--- |
| `ASANA_TOKEN` | `1/123...` | Your Personal Access Token (PAT). |
| `ASANA_WORKSPACE` | `123456789` | The GID of the target workspace. |

### Scheduling (6-Field Cron)
*Format: [Sec] [Min] [Hour] [Dom] [Mon] [Dow]*

| Frequency | Expression |
| :--- | :--- |
| **Every 5 Minutes** | `0 */5 * * * *` |
| **Every 30 Minutes** | `0 */30 * * * *` |
| **Every Hour** | `0 0 * * * *` |
| **Every Day (Midnight)** | `0 0 0 * * *` |

### Fine-Tuning
| Variable | Default | Description |
| :--- | :--- | :--- |
| `REQUESTS_PER_MINUTE` | `150` | Global token bucket refill rate. |
| `MAX_CONCURRENT_READ` | `50` | Simultaneous GET requests allowed. |
| `MAX_CONCURRENT_WRITE` | `15` | Simultaneous POST/PUT/DELETE requests allowed. |
| `MAX_RETRIES` | `5` | Attempts per request before failing. |
| `INITIAL_BACKOFF` | `1s` | Starting wait time for exponential backoff. |
| `MAX_BACKOFF` | `60s` | Maximum duration to wait between retries. |
| `HTTP_TIMEOUT` | `30s` | Maximum duration for a single network request. |
| `USER_PAGE_SIZE` | `100` | Results per page for User queries. |
| `PROJECT_PAGE_SIZE` | `100` | Results per page for Project queries. |
| `OUTPUT_DIR` | `./output` | Destination path for JSON data storage. |
| `BASE_URL` | `https://app...` | Asana API base endpoint. |

---

## 📂 Output Structure

Resources are stored in subdirectories named by the Asana GID.

```text
output/
├── users/
│   ├── 11002233.json
│   └── 11002234.json
└── projects/
    ├── 44556677.json
    └── 44556678.json