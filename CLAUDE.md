# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
go build -o terminator .

# Build for ARM (e.g. AWS t4g)
GOARCH=arm64 GOOS=linux go build -o terminator .

# Run (default 1-minute interval)
./terminator

# Run with custom interval
./terminator --interval 30s

# Run with custom config file
./terminator --config /path/to/config.yaml

# Test webhook is reachable (calls it on every check, not just on success)
./terminator --always-call-webhook

# Run directly without building
go run main.go --interval 30s
```

There are no tests in this project.

## Architecture

Single-file Go application (`main.go`). One persistent headless Chrome instance is shared across all checks via a long-lived chromedp browser context.

**Flow per check (`snipe` loop):**
1. Navigate to the service page (`serviceURL`) to establish session/cookies, then navigate directly to the Mitte booking URL (`mitteURL`)
2. Capture the HTTP status of the document response via a `chromedp.ListenTarget` network event listener
3. Read `document.body.id`, `window.location.href`, and the page `h2`/`h1` headline
4. **Known failures:** `body.id="taken"` (no slots page) or HTTP 429 → log and wait `--interval`
5. **Success:** 2xx status and not a known failure → log, ring terminal bell, call webhook if configured

**Webhook** is configured in `config.yaml` (`webhook_url` field). On success it sends a plain-text POST: `"Found an Appointment, check <serviceURL>"`. URL is validated to be http/https at startup; invalid URLs disable the webhook silently.

**Signals:** SIGINT/SIGTERM cancel the browser context, which unblocks the `select` in the loop and exits cleanly.
