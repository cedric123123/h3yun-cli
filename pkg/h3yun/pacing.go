package h3yun

import (
	"context"
	"time"
)

// waitRequest reserves a request start time within one Client. Reservations are
// made before waiting so a canceled caller cannot let concurrent calls bunch up.
// It does not coordinate separate clients or CLI processes.
func (c *Client) waitRequest(ctx context.Context) error {
	now := time.Now()
	c.paceMu.Lock()
	scheduled := now
	if next := c.lastRequest.Add(c.cfg.MinInterval); next.After(scheduled) {
		scheduled = next
	}
	c.lastRequest = scheduled
	c.paceMu.Unlock()
	if delay := time.Until(scheduled); delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return Fail("CANCELED", "request canceled before sending", 4)
		case <-timer.C:
		}
	}
	if ctx.Err() != nil {
		return Fail("CANCELED", "request canceled before sending", 4)
	}
	return nil
}
