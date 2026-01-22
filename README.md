# GAA File Organizer

**An intelligent, real-time file organization daemon for automated file management**

---

## Overview

GAA File Organizer is a lightweight, concurrent file organization daemon written in Go. It monitors specified directories for new files and automatically moves them into organized destination folders based on configurable matching rules. Perfect for automating file organization workflows, managing downloads folders, organizing project files, or implementing custom file management pipelines.

**Key Benefits:**
- Automated, real-time file organization
- Zero manual intervention required
- Flexible, rule-based configuration
- Production-ready with comprehensive logging
- Cross-platform support (Linux, macOS, Windows)

---

## Features

- **Real-time File Monitoring** — Uses fsnotify for instant file system event detection
- **Recursive Directory Watching** — Monitor directories and all their subdirectories automatically
- **Advanced File Filtering**:
  - Match by file extension (`.pdf`, `.docx`, `.xlsx`, etc.)
  - Match by filename patterns (`contains`, `starts_with`)
  - Automatic exclusion of hidden files (`.hidden`)
  - Automatic exclusion of temporary files (`.tmp`, `.crdownload`, `.part`)
- **Concurrent Processing** — Configurable worker pool for parallel file operations
- **Conflict Resolution** — Three strategies: `rename`, `overwrite`, or `skip`
- **Intelligent File Handling**:
  - Cross-device move support (copy + delete fallback)
  - File readiness checking (ensures files are fully written)
  - Retry logic for locked/busy files
- **Comprehensive Logging** — Multiple log levels (debug, info, warn, error) with file and console output
- **Graceful Shutdown** — Clean exit handling with pending job completion
- **Robust Error Handling** — Panic recovery and detailed error reporting

---

## Installation

### Prerequisites

- **Go 1.25.5 or later**
- Unix-like OS (Linux, macOS) or Windows

### Setup

1. **Clone or download the project:**
   ```bash
   git clone <repository-url>
   cd GAA
   ```

2. **Install dependencies:**
   ```bash
   go mod download
   ```

3. **Build the application:**
   ```bash
   go build -o gaa-organizer main.go
   ```

4. **Verify the build:**
   ```bash
   ./gaa-organizer -version  # If version flag is implemented
   # or simply run it to see usage
   ./gaa-organizer
   ```

---

## Quick Start

1. **Create a configuration file** (`config.yaml`):
   ```yaml
   settings:
     log_level: info
     delay_before_move: 2s
     max_workers: 4

   monitors:
     - name: downloads_organizer
       source_path: ~/Downloads
       recursive: true
       rules:
         - name: pdf_documents
           extensions: [pdf]
           destination: ~/Documents/PDFs
           conflict_strategy: rename

         - name: images
           extensions: [jpg, jpeg, png, gif]
           destination: ~/Pictures
           conflict_strategy: rename
   ```

2. **Run the organizer:**
   ```bash
   ./gaa-organizer
   ```

3. **Monitor the logs:**
   ```bash
   tail -f logs/organizer.log
   ```

---

## Configuration Guide

The `config.yaml` file controls all aspects of the GAA File Organizer.

### Settings Section (Optional)

Global settings with sensible defaults:

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `log_level` | string | `info` | Log verbosity: `debug`, `info`, `warn`, `error` |
| `delay_before_move` | duration | `2s` | Time to wait before moving a file (ensures file is fully written) |
| `max_workers` | integer | `4` | Number of concurrent file processing workers |

**Example:**
```yaml
settings:
  log_level: debug
  delay_before_move: 3s
  max_workers: 8
```

### Monitors Section (Required)

Array of directory monitors to watch:

| Option | Type | Required | Description |
|--------|------|----------|-------------|
| `name` | string | ✓ | Unique identifier for this monitor |
| `source_path` | string | ✓ | Directory path to monitor |
| `recursive` | boolean | ✗ | Watch subdirectories (default: false) |
| `rules` | array | ✓ | Array of matching rules |

### Rules Section (Required per Monitor)

Each rule defines matching criteria and a destination:

| Option | Type | Required | Description |
|--------|------|----------|-------------|
| `name` | string | ✓ | Unique identifier for this rule |
| `extensions` | array | ✗ | File extensions to match (e.g., `[pdf, doc, docx]`) |
| `name_contains` | array | ✗ | Strings that must appear in filename |
| `name_starts_with` | array | ✗ | Strings the filename must start with |
| `destination` | string | ✓ | Target directory for matched files |
| `conflict_strategy` | string | ✓ | How to handle existing files: `rename`, `overwrite`, `skip` |

**Rule Matching Logic:**
- Rules are evaluated in order; the first matching rule is applied
- ALL specified criteria within a rule must match (AND logic)
- Matching is case-insensitive
- If multiple arrays are defined, file must match at least one item from each array

---

## Configuration Examples

### Example 1: Simple Downloads Organization

Organize a downloads folder by file type:

```yaml
settings:
  log_level: info
  delay_before_move: 2s
  max_workers: 4

monitors:
  - name: downloads_organizer
    source_path: ~/Downloads
    recursive: false
    rules:
      - name: documents
        extensions: [pdf, doc, docx, txt, xls, xlsx]
        destination: ~/Downloads/Documents
        conflict_strategy: rename

      - name: images
        extensions: [jpg, jpeg, png, gif, bmp, svg]
        destination: ~/Downloads/Images
        conflict_strategy: rename

      - name: videos
        extensions: [mp4, mkv, avi, mov, flv, wmv]
        destination: ~/Downloads/Videos
        conflict_strategy: rename

      - name: archives
        extensions: [zip, rar, 7z, tar, gz]
        destination: ~/Downloads/Archives
        conflict_strategy: rename
```

### Example 2: Advanced Multi-Monitor Setup

Organization by content with naming patterns:

```yaml
settings:
  log_level: debug
  delay_before_move: 3s
  max_workers: 6

monitors:
  - name: project_documents
    source_path: ~/Projects/Incoming
    recursive: true
    rules:
      - name: project_reports
        name_starts_with: [Report, Relatorio]
        extensions: [pdf, docx]
        destination: ~/Projects/Reports
        conflict_strategy: rename

      - name: project_invoices
        name_contains: [invoice, invoice, fattura]
        extensions: [pdf, xlsx]
        destination: ~/Projects/Invoices
        conflict_strategy: rename

  - name: email_attachments
    source_path: ~/Downloads/Email_Attachments
    recursive: false
    rules:
      - name: work_documents
        name_contains: [work, project, client]
        destination: ~/Work/Documents
        conflict_strategy: rename

      - name: personal_photos
        extensions: [jpg, jpeg, png]
        name_contains: [family, vacation, holiday]
        destination: ~/Pictures/Personal
        conflict_strategy: rename
```

---

## File Filtering

### Automatic Exclusions

The following files are **automatically excluded** and never processed:

- **Hidden files**: Files starting with `.` (e.g., `.DS_Store`, `.gitignore`)
- **Temporary files**:
  - `.tmp`
  - `.crdownload` (Chrome downloads in progress)
  - `.part` (partial downloads)
  - `.download`
- **Destination folders**: To prevent infinite loops, the source path and destination paths are excluded from monitoring

### Extension Matching

- Extensions are matched case-insensitively: `PDF`, `Pdf`, `pdf` all match
- Include extensions without the dot in config: `extensions: [pdf, doc, docx]`
- A file matches if its extension matches ANY extension in the list (OR logic)

### Filename Pattern Matching

**`name_contains`:** Match files that contain a substring anywhere in the filename
```yaml
name_contains: [invoice, fattura, bill]  # Matches: invoice_2024.pdf, my-fattura.pdf
```

**`name_starts_with`:** Match files that start with a specific prefix
```yaml
name_starts_with: [Report, Rel]  # Matches: Report_2024.pdf, Rel_001.xlsx
```

---

## Conflict Resolution

When a file exists at the destination, the `conflict_strategy` determines what happens:

### Strategy 1: `rename` (Recommended)

Appends a counter to the filename:
```
original.pdf → original_1.pdf
original_1.pdf → original_2.pdf
...
original_1000.pdf → original_2026-01-22-14-30-45.pdf  # Switches to timestamp if >1000 conflicts
```

**Best for:** Preserving all files without data loss

### Strategy 2: `overwrite`

Replaces the existing file with the new one:
```
existing.pdf → (replaced with new file)
```

**Best for:** You want the latest version only

### Strategy 3: `skip`

Leaves the file in the source directory, does not move it:
```
file.pdf → (remains in source directory)
```

**Best for:** Safety-first approach, manual review required

---

## Usage

### Running the Organizer

Start the daemon with the default configuration:
```bash
./gaa-organizer
```

Start with a custom configuration file:
```bash
./gaa-organizer -config /path/to/custom-config.yaml
```

### Stopping the Organizer

Press `Ctrl+C` to gracefully shut down. The application will:
1. Stop accepting new file events
2. Complete processing of queued jobs
3. Close all file handles
4. Exit cleanly

### Viewing Logs

View real-time logs:
```bash
tail -f logs/organizer.log
```

View logs with specific level:
```bash
grep "ERROR" logs/organizer.log
grep "WARN" logs/organizer.log
```

---

## Logging

### Log Levels

| Level | Use Case | Typical Output |
|-------|----------|----------------|
| `debug` | Development & troubleshooting | All operations, detailed file checks, worker states |
| `info` | Normal operation | File movements, rule matches, startup/shutdown |
| `warn` | Important but non-critical | Skipped files, renamed conflicts, retry attempts |
| `error` | Failures that need attention | Failed moves, permission errors, configuration issues |

### Log Output

Logs are written to:
- **Console** (`stdout`/`stderr`) — Real-time feedback
- **File** (`logs/organizer.log`) — Persistent record for later analysis

### Example Log Entries

```
[2026-01-22T10:15:30Z] INFO: Starting file organizer with 4 workers
[2026-01-22T10:15:30Z] INFO: Monitor "downloads_organizer" started watching ~/Downloads
[2026-01-22T10:15:45Z] INFO: Moving ~/Downloads/document.pdf -> ~/Downloads/Documents/document.pdf
[2026-01-22T10:15:46Z] INFO: File moved successfully, rule "pdf_documents" applied
[2026-01-22T10:16:20Z] WARN: File exists at destination, applying rename strategy: document_1.pdf
[2026-01-22T10:16:21Z] DEBUG: Worker 2 completed job for invoice.xlsx
```

---

## Project Architecture

### Directory Structure

```
GAA/
├── main.go                    # Application entry point
├── config.yaml               # Configuration file
├── README.md                 # This file
├── go.mod                    # Go module definition
├── go.sum                    # Dependency checksums
├── logs/                     # Log output directory
│   └── organizer.log        # Daily/rolling log file
└── src/
    ├── config/
    │   ├── config.go        # Config parsing and validation
    │   └── logger.go        # Logging initialization
    ├── processor/
    │   ├── rules.go         # Rule matching engine
    │   └── mover.go         # File movement operations
    └── watcher/
        ├── watcher.go       # File system monitoring (fsnotify)
        └── worker_pool.go   # Concurrent job processing
```

### Core Components

**Watcher** (`src/watcher/`): Monitors file system events using fsnotify, filters irrelevant events, checks file readiness, and submits jobs to the worker pool.

**Processor** (`src/processor/`): Implements rule matching to determine which destination a file should move to, and handles the actual file move operations with conflict resolution.

**Config** (`src/config/`): Parses and validates the YAML configuration, initializes logging, and provides configuration to all components.

**Worker Pool**: Concurrent job queue with configurable workers for parallel file processing and graceful shutdown.

---

## Troubleshooting

### Files Not Being Moved

**Check the following:**
1. Is the organizer running? (`ps aux | grep gaa-organizer`)
2. Are there any `ERROR` or `WARN` messages in `logs/organizer.log`?
3. Are file paths correct? (Use absolute paths in config)
4. Is the source directory actually being monitored? (Check log on startup)
5. Do the rule criteria match your files? (Try `debug` log level)

### Permission Denied Errors

**Solutions:**
1. Run with appropriate permissions: `sudo ./gaa-organizer` (not recommended)
2. Check file ownership and directory permissions
3. Ensure destination directories are writable: `chmod 755 destination_dir`

### Files Not Matching Rules

**Debug steps:**
1. Set `log_level: debug` in config for detailed matching information
2. Verify file extensions in config don't include the dot: `extensions: [pdf]` not `extensions: [.pdf]`
3. Check case-sensitivity in `name_contains` and `name_starts_with` (matching is case-insensitive)
4. Verify rule order: first matching rule is applied

### Conflicts Not Resolving

1. Check `conflict_strategy` is set to `rename`, `overwrite`, or `skip`
2. Verify destination directory has write permissions
3. Check logs for permission errors
4. For `rename` strategy: verify disk space for renamed files

---

## Performance Tuning

### Optimize `max_workers`

More workers = faster processing, but more system resource usage:
- **Downloads folder (typical usage)**: 4-6 workers
- **Large file volumes**: 8-16 workers
- **Low-power systems**: 2-4 workers

```yaml
settings:
  max_workers: 8
```

### Adjust `delay_before_move`

Balance between responsiveness and file stability:
- **Fast completion files**: `1s` or `500ms`
- **Large files**: `3-5s` (ensure fully written)
- **Network shares**: `5-10s` (slower write confirmation)

```yaml
settings:
  delay_before_move: 3s
```

### Recursive Monitoring

Disable if not needed to reduce resource usage:
```yaml
monitors:
  - recursive: false  # Only monitor top-level directory
```

---

## Common Use Cases

### Automated Downloads Organization

Keep your Downloads folder organized automatically:
```yaml
monitors:
  - name: downloads
    source_path: ~/Downloads
    recursive: false
    rules:
      # Documents → Documents folder
      # Images → Pictures folder
      # Videos → Videos folder
      # Archives → Compressed folder
```

### Project File Organization

Organize incoming project files by type:
```yaml
monitors:
  - name: project_inbox
    source_path: ~/Projects/Inbox
    recursive: true
    rules:
      - name: source_code
        extensions: [js, py, go, java, cpp, h]
        destination: ~/Projects/Source
      - name: documentation
        extensions: [md, pdf, doc, docx]
        destination: ~/Projects/Docs
```

### Email Attachments Management

Auto-organize attachments based on content:
```yaml
monitors:
  - name: email_attachments
    source_path: ~/Downloads/Email
    recursive: false
    rules:
      - name: invoices
        name_contains: [invoice, receipt]
        destination: ~/Accounting/Invoices
      - name: contracts
        name_contains: [contract, agreement]
        destination: ~/Legal/Contracts
```

---