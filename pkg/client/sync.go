package client

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// SyncInterval is the default interval for background sync.
const SyncInterval = 5 * time.Minute

type syncResponse struct {
	Status     string   `json:"status"`
	RevokeKeys []string `json:"revoke_keys"`
}

type revocationsResponse struct {
	RevokedKeys []string `json:"revoked_keys"`
}

// StartBackgroundSync starts a goroutine that periodically syncs with the server.
func (c *Client) StartBackgroundSync(ctx context.Context, logger *slog.Logger) {
	if c.config.ServerURL == "" {
		if logger != nil {
			logger.Info("Background sync disabled (no ServerURL)")
		}
		return
	}

	httpClient := &http.Client{Timeout: 10 * time.Second}

	ticker := time.NewTicker(SyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.performSync(httpClient, logger)
			c.fetchRevocations(httpClient, logger)
		}
	}
}

func (c *Client) performSync(httpClient *http.Client, logger *slog.Logger) {
	payloads, err := c.store.DequeueAllSync()
	if err != nil || len(payloads) == 0 {
		return
	}

	for _, payload := range payloads {
		body, err := json.Marshal(payload)
		if err != nil {
			continue
		}

		req, err := http.NewRequest("POST", c.config.ServerURL+"/api/v1/activate", bytes.NewBuffer(body))
		if err != nil {
			c.store.EnqueueSync(payload)
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := httpClient.Do(req)
		if err != nil {
			c.store.EnqueueSync(payload)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			c.store.EnqueueSync(payload)
			resp.Body.Close()
			continue
		}

		var syncResp syncResponse
		if err := json.NewDecoder(resp.Body).Decode(&syncResp); err == nil {
			if len(syncResp.RevokeKeys) > 0 {
				c.store.AddToBlacklist(syncResp.RevokeKeys)
			}
		}
		resp.Body.Close()
	}
}

func (c *Client) fetchRevocations(httpClient *http.Client, logger *slog.Logger) {
	resp, err := httpClient.Get(c.config.ServerURL + "/api/v1/revocations")
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var revResp revocationsResponse
		if err := json.NewDecoder(resp.Body).Decode(&revResp); err == nil {
			if len(revResp.RevokedKeys) > 0 {
				c.store.AddToBlacklist(revResp.RevokedKeys)
			}
		}
	}
}
