# terminator

Polls the Berlin city appointment service ([service.berlin.de](https://service.berlin.de)) for available slots at the Mitte location. Alerts via terminal bell and an optional webhook when a slot appears.

## Dependencies

- **Go 1.21+** — [install](https://go.dev/dl/)
- **Google Chrome or Chromium** — used for headless browser automation

### Install Chrome on Ubuntu/Debian

```bash
wget https://dl.google.com/linux/direct/google-chrome-stable_current_amd64.deb
sudo dpkg -i google-chrome-stable_current_amd64.deb
sudo apt --fix-broken install
```

Or Chromium:

```bash
sudo apt install chromium-browser
```

## Build

```bash
go build -o terminator .
```

For ARM (e.g. AWS t4g):

```bash
GOARCH=arm64 GOOS=linux go build -o terminator .
```

## Configuration

Edit `config.yaml`:

```yaml
webhook_url: "https://your-webhook-url"
```

The webhook receives a plain-text `POST` with body:

```
Found an Appointment, check https://service.berlin.de/dienstleistung/351180/
```

Leave `webhook_url` empty or omit the file to disable the webhook.

### Recommended: ntfy.sh for phone notifications

[ntfy.sh](https://ntfy.sh) is a free, no-signup push notification service. When terminator fires the webhook, you get an instant notification on your phone.

1. Install the [ntfy app](https://ntfy.sh/#subscribe) on your phone (Android/iOS)
2. Subscribe to a topic — pick any unique name, e.g. `myname-berlin-termin`
3. Set the webhook URL in `config.yaml`:

```yaml
webhook_url: "https://ntfy.sh/myname-berlin-termin"
```

4. Test it:

```bash
./terminator --always-call-webhook --interval 5s
```

Your phone should buzz within seconds.

## Usage

```bash
# default 1-minute interval
./terminator

# custom interval
./terminator --interval 30s

# custom config file
./terminator --config /path/to/config.yaml

# test your webhook on every check (useful for verifying ntfy.sh setup)
./terminator --always-call-webhook
```

## How it works

On each check, the tool:

1. Opens a headless Chrome browser and navigates to the Berlin appointment service
2. Reads the page state (`body.id`, HTTP status, headline)
3. If the page is the "no slots" page (`body.id="taken"`) or returns 429 — waits and retries
4. Anything else (2xx status, unknown page) is treated as a potential slot: logs loudly, rings the terminal bell, and calls the webhook

## Running on a server (tmux)

```bash
tmux new -s terminator
./terminator --interval 30s
# detach: Ctrl+B then D
# reattach: tmux attach -t terminator
```
