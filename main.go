package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"gopkg.in/yaml.v3"
)

const (
	serviceURL = "https://service.berlin.de/dienstleistung/351180/"
	mitteURL   = "https://service.berlin.de/terminvereinbarung/termin/tag.php?id=4126&anliegen[]=351180&termin=1&dienstleister=351636&anliegen[]=351180"
)

type Config struct {
	WebhookURL string `yaml:"webhook_url"`
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if u := cfg.WebhookURL; u != "" {
		parsed, err := url.Parse(u)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			log.Printf("config: webhook_url %q is not a valid http/https URL — webhook disabled", u)
			cfg.WebhookURL = ""
		}
	}
	return &cfg, nil
}

func callWebhook(webhookURL string) {
	msg := "Found an Appointment, check " + serviceURL
	resp, err := http.Post(webhookURL, "text/plain", strings.NewReader(msg))
	if err != nil {
		log.Printf("webhook: request failed: %v", err)
		return
	}
	defer resp.Body.Close()
	log.Printf("webhook: called %s → %d", webhookURL, resp.StatusCode)
}

func main() {
	interval          := flag.Duration("interval", 1*time.Minute, "retry interval (e.g. 20s, 1m, 2m30s)")
	configFile        := flag.String("config", "config.yaml", "path to config file")
	alwaysCallWebhook := flag.Bool("always-call-webhook", false, "call webhook on every check (useful for testing)")
	flag.Parse()

	var cfg *Config
	if c, err := loadConfig(*configFile); err != nil {
		log.Printf("config: not loaded (%v) — webhook disabled", err)
	} else {
		cfg = c
		if cfg.WebhookURL != "" {
			log.Printf("config: webhook → %s", cfg.WebhookURL)
		}
	}

	opts := chromedp.DefaultExecAllocatorOptions[:]
	opts = append(opts,
		chromedp.Flag("headless", true),
		chromedp.UserAgent("Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	ctx, browserCancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer browserCancel()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		s := <-sig
		log.Printf("received %s, shutting down", s)
		browserCancel()
	}()

	log.Printf("retry interval: %s", *interval)
	snipe(ctx, *interval, cfg, *alwaysCallWebhook)
}

func snipe(ctx context.Context, retryEvery time.Duration, cfg *Config, alwaysCallWebhook bool) {
	for {
		log.Printf("--- checking appointments ---")

		var lastStatus atomic.Int64
		chromedp.ListenTarget(ctx, func(ev interface{}) {
			if e, ok := ev.(*network.EventResponseReceived); ok {
				if e.Type == network.ResourceTypeDocument {
					lastStatus.Store(e.Response.Status)
				}
			}
		})

		var bodyID, currentURL, headline string
		err := chromedp.Run(ctx,
			network.Enable(),
			chromedp.Navigate(serviceURL),
			chromedp.Navigate(mitteURL),
			chromedp.Evaluate("document.body.id", &bodyID),
			chromedp.Evaluate("window.location.href", &currentURL),
			chromedp.ActionFunc(func(ctx context.Context) error {
				_ = chromedp.Text("h2", &headline).Do(ctx)
				if headline == "" {
					_ = chromedp.Text("h1", &headline).Do(ctx)
				}
				return nil
			}),
		)

		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("error: %v — retrying in %s", err, retryEvery)
		} else {
			status := lastStatus.Load()
			headline = strings.TrimSpace(headline)
			log.Printf("status=%d body.id=%q url=%s", status, bodyID, currentURL)
			if headline != "" {
				log.Printf("headline: %q", headline)
			}

			is2xx := status >= 200 && status < 300
			known := status == 429 || bodyID == "taken"
			if !is2xx || known {
				log.Printf("no slots available, retrying in %s", retryEvery)
				if alwaysCallWebhook && cfg != nil && cfg.WebhookURL != "" {
					callWebhook(cfg.WebhookURL)
				}
			} else {
				log.Printf("!!! APPOINTMENT FOUND — slots may be available !!!")
				fmt.Print("\a")
				if cfg != nil && cfg.WebhookURL != "" {
					callWebhook(cfg.WebhookURL)
				}
				// keep retrying — slot might still be there on next check
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(retryEvery):
		}
	}
}
