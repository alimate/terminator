# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
go build -o terminator .

# Run (default 1-minute interval)
./terminator

# Run with custom interval
./terminator --interval 30s

# Run directly without building
go run main.go --interval 30s
```

There are no tests in this project.

## Architecture

Single-file Go application (`main.go`) that uses a headless Chrome browser (via chromedp) to check the Berlin city appointment service (`service.berlin.de`) at a configurable interval, looking for available appointment slots at the Mitte location.

- **`snipe()`** navigates to the service page then the Mitte booking URL, reads `body.id`, the HTTP status, and the page headline, then logs the result. If `body.id="taken"` it waits and retries; if `body.id="day"` it alerts.
- **`main()`** parses the `--interval` flag, launches a headless Chrome context, and runs `snipe()` in a loop until SIGINT/SIGTERM.
