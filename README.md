# proxypool-go

High-performance, in-memory proxy pool manager for Go. Parse, rotate, flag, ping, and manage outbound HTTP/SOCKS proxies with optional persistence.

## Install

```bash
go get github.com/status403com/proxypool-go
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    proxypool "github.com/status403com/proxypool-go"
)

func main() {
    pool, err := proxypool.New(proxypool.Config{
        Proxies: []string{
            "1.2.3.4:8080",
            "socks5://user:pass@5.6.7.8:1080",
            "admin:secret@9.10.11.12:3128",
            "10.0.0.1:8080:user:pass",
        },
        Protocol: proxypool.HTTP, // default for strings without a scheme
    })
    if err != nil {
        log.Fatal(err)
    }

    proxy, err := pool.Next()
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(proxy.URL()) // http://1.2.3.4:8080
}
```

## Input Formats

All of these are auto-detected:

```
host:port
host:port:user:pass
user:pass@host:port
user:pass:host:port
http://host:port
socks5://user:pass@host:port
```

You can also pass pre-parsed entries:

```go
pool, _ := proxypool.New(proxypool.Config{
    Entries: []proxypool.Entry{
        {Host: "1.2.3.4", Port: "8080", User: "u", Pass: "p", Protocol: proxypool.SOCKS5},
    },
})
```

## Smart Selection

`Next()` picks proxies intelligently:

1. Skips dead proxies (marked via `MarkDead` or `PingAndMarkDead`)
2. Skips flagged proxies (temp-banned, expires after `FlagDuration`)
3. Prefers proxies not currently in use
4. Falls back to reusing in-use proxies when all are taken

```go
proxy, _ := pool.Next()
// ... use proxy ...
pool.Release(proxy.ID)

// Temp-ban a bad proxy
pool.Flag(proxy.ID)

// Permanently remove from rotation
pool.MarkDead(proxy.ID)

// Bring it back
pool.Revive(proxy.ID)
```

## Flag Duration

The flag/temp-ban duration defaults to 10 minutes. Override it:

```go
pool, _ := proxypool.New(proxypool.Config{
    Proxies:      []string{"1.2.3.4:8080"},
    FlagDuration: 5 * time.Minute, // custom duration
})
```

## Consumer Isolation

When multiple systems share the same proxy list but need independent flag/in-use tracking:

```go
pool, _ := proxypool.New(proxypool.Config{
    Proxies: []string{"1.2.3.4:8080", "5.6.7.8:3128", "9.10.11.12:1080"},
})

checkout := pool.Consumer("checkout")
monitor  := pool.Consumer("monitor")

// Flag on checkout doesn't affect monitor
p, _ := checkout.Next()
checkout.Flag(p.ID) // only flagged for "checkout" consumer

m, _ := monitor.Next() // can still get the same proxy
monitor.Release(m.ID)
```

## Ping Testing

```go
// Synchronous — test all proxies
stats := pool.Ping("https://httpbin.org/ip", 20) // 20 concurrent
fmt.Printf("Alive: %d, Dead: %d, Avg: %v\n", stats.Alive, stats.Dead, stats.AvgPing)

// Streaming — get results as they come in
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

pool.PingLive(ctx, "https://httpbin.org/ip", 20, func(r proxypool.PingResult, done, total int) {
    fmt.Printf("[%d/%d] %s — %v\n", done, total, r.ProxyID, r.Latency)
})

// Ping and auto-remove dead proxies from rotation
pool.PingAndMarkDead("https://httpbin.org/ip", 20)
```

## Persistence

The pool works entirely in-memory by default. Optionally plug in a store for persistence.

### JSON File

```go
pool, _ := proxypool.New(proxypool.Config{
    Proxies: []string{"1.2.3.4:8080", "5.6.7.8:3128"},
    Store:   proxypool.NewJSONStore("proxies.json"),
})
defer pool.Close() // final save

// Proxies persist across restarts.
// Add/Remove auto-save to the file.
```

### SQLite

```go
import "github.com/status403com/proxypool-go/sqlitestore"

store, err := sqlitestore.New("proxies.db")
if err != nil {
    log.Fatal(err)
}
defer store.Close()

pool, _ := proxypool.New(proxypool.Config{
    Proxies: []string{"1.2.3.4:8080"},
    Store:   store,
})
defer pool.Close()
```

### Custom Store

Implement the `Store` interface for any backend:

```go
type Store interface {
    Load() ([]proxypool.Proxy, error)
    Save(proxies []proxypool.Proxy) error
}
```

## Dynamic Management

```go
// Add proxies at runtime
pool.Add("10.0.0.1:8080", "10.0.0.2:8080")

// Add pre-parsed entries
pool.AddEntries(proxypool.Entry{Host: "10.0.0.3", Port: "8080", Protocol: proxypool.HTTP})

// Remove by ID
pool.Remove(proxy.ID)

// Get all proxies
all := pool.All()

// Count
fmt.Println(pool.Len())
```

## API Reference

### Pool

| Method | Description |
|--------|-------------|
| `New(Config) (*Pool, error)` | Create a new pool |
| `Add(raw ...string) error` | Parse and add proxies |
| `AddEntries(entries ...Entry) error` | Add pre-parsed entries |
| `Remove(id string) error` | Remove a proxy by ID |
| `Len() int` | Number of proxies in pool |
| `All() []Proxy` | Snapshot of all proxies |
| `Next() (Proxy, error)` | Smart pick (via default consumer) |
| `Random() (Proxy, error)` | Random pick, no exclusions |
| `Flag(id string)` | Temp-ban (via default consumer) |
| `Release(id string)` | Mark not in use (via default consumer) |
| `MarkDead(id string)` | Permanently exclude from rotation |
| `Revive(id string)` | Undo MarkDead |
| `Consumer(name string) *Consumer` | Create isolated consumer |
| `Ping(url string, concurrency int) PingStats` | Test all proxies |
| `PingLive(ctx, url, concurrency, callback) error` | Streaming ping |
| `PingAndMarkDead(url string, concurrency int) PingStats` | Ping + auto-mark dead |
| `Close() error` | Final save (if store set) |

### Consumer

| Method | Description |
|--------|-------------|
| `Next() (Proxy, error)` | Smart pick with isolated state |
| `Flag(id string)` | Temp-ban on this consumer only |
| `Release(id string)` | Mark not in use |
| `ReleaseAll()` | Release all in-use proxies |
| `InUse() []Proxy` | Currently assigned proxies |

## License

MIT
