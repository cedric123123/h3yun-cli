package h3yun

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPacingConcurrentStartsAndCancellation(t *testing.T) {
	c := &Client{cfg: Config{MinInterval: 30 * time.Millisecond}}
	start := time.Now()
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := c.waitRequest(context.Background()); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if time.Since(start) < 90*time.Millisecond {
		t.Fatal("concurrent callers bypassed spacing")
	}
	c.cfg.MinInterval = time.Minute
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start = time.Now()
	if err := c.waitRequest(ctx); err == nil || err.(*Error).Code != "CANCELED" {
		t.Fatalf("cancellation: %v", err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("cancellation did not interrupt wait")
	}
}

func TestPacingCanceledWriteNeverSent(t *testing.T) {
	var calls atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		fmt.Fprint(w, `{"Successful":true}`)
	}))
	defer s.Close()
	c := clientFor(t, s)
	c.cfg.MinInterval = time.Minute
	if _, err := c.Invoke(context.Background(), map[string]any{"ActionName": "test"}, false); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := c.Invoke(ctx, map[string]any{"ActionName": "write"}, true)
	e, ok := err.(*Error)
	if !ok || e.Code != "CANCELED" || e.OutcomeUnknown || calls.Load() != 1 {
		t.Fatalf("calls=%d error=%+v", calls.Load(), err)
	}
}

func TestPacingConfigurationBounds(t *testing.T) {
	for _, interval := range []time.Duration{-1, 0, time.Minute, time.Minute + 1} {
		c, err := New(Config{BaseURL: "https://example.test", EngineCode: "e", EngineSecret: "s", Timeout: time.Second, MinInterval: interval, InvokePath: "/invoke", UploadPath: "/upload", DownloadPath: "/download"})
		valid := interval >= 0 && interval <= time.Minute
		if (err == nil) != valid {
			t.Fatalf("%v: %v", interval, err)
		}
		if c != nil {
			c.Close()
		}
	}
}
