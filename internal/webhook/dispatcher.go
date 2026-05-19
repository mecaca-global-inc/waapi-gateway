package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/mecaca/waapi-gateway/internal/store"
	"github.com/rs/zerolog/log"
)

type Dispatcher struct {
	repo   *store.Repo
	client *http.Client
	bus    chan Envelope
}

func New(repo *store.Repo) *Dispatcher {
	return &Dispatcher{
		repo:   repo,
		client: &http.Client{Timeout: 15 * time.Second},
		bus:    make(chan Envelope, 256),
	}
}

func (d *Dispatcher) Start(ctx context.Context, workers int) {
	if workers <= 0 {
		workers = 4
	}
	for i := 0; i < workers; i++ {
		go d.worker(ctx)
	}
}

func (d *Dispatcher) Emit(ev Envelope) {
	if ev.Time == 0 {
		ev.Time = time.Now().Unix()
	}
	select {
	case d.bus <- ev:
	default:
		log.Warn().Str("event", ev.Event).Msg("webhook bus full, dropping event")
	}
}

func (d *Dispatcher) Bus() <-chan Envelope { return d.bus }

func (d *Dispatcher) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev := <-d.bus:
			d.deliver(ctx, ev)
		}
	}
}

func (d *Dispatcher) deliver(ctx context.Context, ev Envelope) {
	hooks, err := d.repo.ListWebhooksBySession(ctx, ev.Session)
	if err != nil {
		log.Error().Err(err).Msg("webhook lookup failed")
		return
	}
	body, err := json.Marshal(ev)
	if err != nil {
		log.Error().Err(err).Msg("webhook marshal failed")
		return
	}
	for _, h := range hooks {
		if !h.Enabled {
			continue
		}
		if len(h.Events) > 0 && !contains(h.Events, ev.Event) {
			continue
		}
		go d.send(h, body)
	}
}

func (d *Dispatcher) send(h store.Webhook, body []byte) {
	backoff := []time.Duration{0, 2 * time.Second, 5 * time.Second}
	for attempt, wait := range backoff {
		if wait > 0 {
			time.Sleep(wait)
		}
		req, err := http.NewRequest(http.MethodPost, h.URL, bytes.NewReader(body))
		if err != nil {
			log.Error().Err(err).Msg("webhook req build failed")
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "whatsapp-gateway/1.0")
		if h.Secret != "" {
			mac := hmac.New(sha256.New, []byte(h.Secret))
			mac.Write(body)
			req.Header.Set("X-Webhook-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
		}
		resp, err := d.client.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < 500 {
				return
			}
		}
		log.Warn().Int("attempt", attempt+1).Str("url", h.URL).Err(err).Msg("webhook delivery retry")
	}
	log.Error().Str("url", h.URL).Msg("webhook delivery failed after retries")
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
