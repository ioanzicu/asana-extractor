# Asana Extractor

A production-ready Go application that extracts users and projects from the Asana API. Built with a focus on resiliency, it features a custom token-bucket rate limiter, exponential backoff, and a high-precision cron-based scheduler.

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

## 🛠 Main Functionality

The application operates as a scheduled service that performs the following:

1.  **Identity Discovery**: Connects to Asana and fetches all accessible users and projects within the configured workspace.
2.  **Resilient Extraction**: Processes resources using an internal queue that respects Asana's strict rate limits (Requests Per Minute and Concurrent Request limits).
3.  **Atomic Persistence**: Saves each resource as an individual JSON file, using the Asana GID as the unique identifier to ensure data consistency and prevent overwrites.



---

## 🌟 Key Features

* **6-Field Cron Scheduling**: High-precision scheduling allowing for second-level granularity (e.g., `0 */5 * * * *`).
* **Cursor-Based Pagination**: Automatically handles large datasets by following Asana's `next_page` tokens, ensuring no data is missed in high-volume environments.
* **Smart Retries**: Implements exponential backoff with jitter and full support for the `Retry-After` header sent by the Asana API.
* **Concurrency Control**: Separates Read (GET) and Write (POST/PUT) limits to maximize throughput without triggering account "Concurrent Request" bans.
* **Graceful Shutdown**: Listens for OS signals (SIGINT, SIGTERM) to stop the scheduler cleanly, ensuring current file writes complete before exiting.

---

## ⚠️ Limits & Constraints

| Category | Limit | Description |
| :--- | :--- | :--- |
| **Throughput** | 150 RPM | Optimized for Asana Free Tier; adjustable via `REQUESTS_PER_MINUTE`. |
| **Concurrency** | 50 Read / 15 Write | Enforced via internal semaphores to prevent API-side connection rejection. |
| **Pagination** | 100 items/page | Set to Asana's maximum page size to minimize network round-trips. |
| **Scheduling** | 1 Second | Minimum supported interval between extractions due to 6-field cron parser. |
| **Storage** | File System | Extraction speed is ultimately bounded by disk IOPS when writing thousands of small JSON files. |

---

## ⚙️ Full Configuration Guide

The following variables are available in your `.env` file:

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

Resources are stored in subdirectories based on type. Files are named using the Asana GID.

```text
output/
├── users/
│   ├── 11002233.json
│   └── 11002234.json
└── projects/
    ├── 44556677.json
    └── 44556678.json