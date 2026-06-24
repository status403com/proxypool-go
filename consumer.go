package proxypool

import (
	"fmt"
	"math/rand/v2"
	"sync"
	"time"
)

type Consumer struct {
	pool    *Pool
	name    string
	mu      sync.Mutex
	flagged map[string]time.Time
	inUse   map[string]bool
	flagDur time.Duration
}

func (c *Consumer) Next() (Proxy, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cleanExpiredFlags()

	alive := c.pool.alive()
	if len(alive) == 0 {
		return Proxy{}, fmt.Errorf("proxypool: no alive proxies")
	}

	var viable []Proxy
	var unused []Proxy
	for _, px := range alive {
		if _, flagged := c.flagged[px.ID]; flagged {
			continue
		}
		viable = append(viable, px)
		if !c.inUse[px.ID] {
			unused = append(unused, px)
		}
	}

	if len(viable) == 0 {
		return Proxy{}, fmt.Errorf("proxypool: no viable proxies (all dead or flagged)")
	}

	pool := unused
	if len(pool) == 0 {
		pool = viable
	}

	chosen := pool[rand.IntN(len(pool))]
	c.inUse[chosen.ID] = true
	return chosen, nil
}

func (c *Consumer) Flag(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.flagged[id] = time.Now().Add(c.flagDur)
}

func (c *Consumer) Release(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.inUse, id)
}

func (c *Consumer) ReleaseAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	clear(c.inUse)
}

func (c *Consumer) InUse() []Proxy {
	c.mu.Lock()
	defer c.mu.Unlock()

	alive := c.pool.alive()
	var result []Proxy
	for _, px := range alive {
		if c.inUse[px.ID] {
			result = append(result, px)
		}
	}
	return result
}

func (c *Consumer) cleanExpiredFlags() {
	now := time.Now()
	for id, unban := range c.flagged {
		if now.After(unban) {
			delete(c.flagged, id)
		}
	}
}
