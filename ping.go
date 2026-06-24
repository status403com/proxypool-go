package proxypool

import (
	"context"
	"fmt"
	"math"
	"net"
	"net/http"
	"net/url"
	"sort"
	"sync"
	"time"
)

type PingResult struct {
	ProxyID string        `json:"proxyId"`
	Latency time.Duration `json:"latency"`
	Error   string        `json:"error,omitempty"`
}

func (r PingResult) Alive() bool { return r.Latency >= 0 }

type PingStats struct {
	Results    []PingResult  `json:"results"`
	Alive      int           `json:"alive"`
	Dead       int           `json:"dead"`
	AvgPing    time.Duration `json:"avgPing"`
	MedianPing time.Duration `json:"medianPing"`
}

func (p *Pool) Ping(targetURL string, concurrency int) PingStats {
	if targetURL == "" {
		targetURL = "https://httpbin.org/ip"
	}
	if concurrency <= 0 {
		concurrency = 20
	}

	proxies := p.alive()
	results := make([]PingResult, len(proxies))

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for i, px := range proxies {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, proxy Proxy) {
			defer wg.Done()
			defer func() { <-sem }()
			results[idx] = pingProxy(context.Background(), proxy, targetURL)
		}(i, px)
	}
	wg.Wait()

	return computeStats(results)
}

func (p *Pool) PingLive(ctx context.Context, targetURL string, concurrency int, onResult func(PingResult, int, int)) error {
	if targetURL == "" {
		targetURL = "https://httpbin.org/ip"
	}
	if concurrency <= 0 {
		concurrency = 20
	}

	proxies := p.alive()
	total := len(proxies)
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	completed := 0

	for _, px := range proxies {
		if ctx.Err() != nil {
			break
		}

		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
		}
		if ctx.Err() != nil {
			break
		}

		wg.Add(1)
		go func(proxy Proxy) {
			defer wg.Done()
			defer func() { <-sem }()

			if ctx.Err() != nil {
				return
			}

			result := pingProxy(ctx, proxy, targetURL)

			mu.Lock()
			completed++
			n := completed
			mu.Unlock()

			onResult(result, n, total)
		}(px)
	}

	wg.Wait()
	return ctx.Err()
}

func (p *Pool) PingAndMarkDead(targetURL string, concurrency int) PingStats {
	stats := p.Ping(targetURL, concurrency)
	for _, r := range stats.Results {
		if !r.Alive() {
			p.MarkDead(r.ProxyID)
		}
	}
	return stats
}

func pingProxy(ctx context.Context, p Proxy, targetURL string) PingResult {
	proxyURL, err := url.Parse(p.URL())
	if err != nil {
		return PingResult{ProxyID: p.ID, Latency: -1, Error: err.Error()}
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
			DialContext: (&net.Dialer{
				Timeout: 10 * time.Second,
			}).DialContext,
		},
		Timeout: 15 * time.Second,
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return PingResult{ProxyID: p.ID, Latency: -1, Error: err.Error()}
	}

	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)

	if err != nil {
		return PingResult{ProxyID: p.ID, Latency: -1, Error: err.Error()}
	}
	resp.Body.Close()

	if resp.StatusCode >= 400 {
		return PingResult{ProxyID: p.ID, Latency: -1, Error: fmt.Sprintf("HTTP %d", resp.StatusCode)}
	}

	return PingResult{ProxyID: p.ID, Latency: elapsed}
}

func computeStats(results []PingResult) PingStats {
	stats := PingStats{Results: results}
	var pings []time.Duration

	for _, r := range results {
		if r.Alive() {
			stats.Alive++
			pings = append(pings, r.Latency)
		} else {
			stats.Dead++
		}
	}

	if len(pings) > 0 {
		var sum time.Duration
		for _, d := range pings {
			sum += d
		}
		stats.AvgPing = time.Duration(math.Round(float64(sum) / float64(len(pings))))

		sort.Slice(pings, func(i, j int) bool { return pings[i] < pings[j] })
		mid := len(pings) / 2
		if len(pings)%2 == 0 {
			stats.MedianPing = (pings[mid-1] + pings[mid]) / 2
		} else {
			stats.MedianPing = pings[mid]
		}
	}

	return stats
}
