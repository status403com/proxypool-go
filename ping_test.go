package proxypool

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPing_DeadProxy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer server.Close()

	pool, _ := New(Config{
		Proxies: []string{"127.0.0.1:59999"},
	})

	stats := pool.Ping(server.URL, 5)
	if stats.Dead != 1 {
		t.Errorf("expected 1 dead, got %d dead / %d alive", stats.Dead, stats.Alive)
	}
}

func TestPingLive_Cancel(t *testing.T) {
	pool, _ := New(Config{
		Proxies: []string{"127.0.0.1:59998", "127.0.0.1:59997"},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	pool.PingLive(ctx, "http://127.0.0.1:1", 1, func(r PingResult, done, total int) {})
}

func TestPingAndMarkDead(t *testing.T) {
	pool, _ := New(Config{
		Proxies: []string{"127.0.0.1:59996"},
	})

	pool.PingAndMarkDead("http://127.0.0.1:1", 5)

	_, err := pool.Next()
	if err == nil {
		t.Error("expected error: proxy should be marked dead")
	}
}

func TestPingResult_Alive(t *testing.T) {
	alive := PingResult{Latency: 100 * time.Millisecond}
	if !alive.Alive() {
		t.Error("expected alive")
	}

	dead := PingResult{Latency: -1}
	if dead.Alive() {
		t.Error("expected dead")
	}
}

func TestPingStats_Empty(t *testing.T) {
	pool, _ := New(Config{})
	stats := pool.Ping("http://localhost:1", 5)
	if len(stats.Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(stats.Results))
	}
}
