package proxypool

import (
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/google/uuid"
)

const defaultFlagDuration = 10 * time.Minute

type Config struct {
	Proxies      []string
	Entries      []Entry
	Protocol     Protocol
	FlagDuration time.Duration
	Store        Store
}

type Pool struct {
	mu      sync.RWMutex
	proxies []Proxy
	index   map[string]int
	dead    map[string]bool
	store   Store
	flagDur time.Duration
	proto   Protocol
	dflt    *Consumer
}

func New(cfg Config) (*Pool, error) {
	flagDur := cfg.FlagDuration
	if flagDur <= 0 {
		flagDur = defaultFlagDuration
	}

	p := &Pool{
		index:   make(map[string]int),
		dead:    make(map[string]bool),
		store:   cfg.Store,
		flagDur: flagDur,
		proto:   cfg.Protocol,
	}

	if cfg.Store != nil {
		loaded, err := cfg.Store.Load()
		if err != nil {
			return nil, fmt.Errorf("proxypool: store load: %w", err)
		}
		for _, px := range loaded {
			p.proxies = append(p.proxies, px)
			p.index[px.ID] = len(p.proxies) - 1
		}
	}

	if len(cfg.Entries) > 0 {
		for _, e := range cfg.Entries {
			px := Proxy{
				ID:       uuid.NewString(),
				Host:     e.Host,
				Port:     e.Port,
				Username: e.User,
				Password: e.Pass,
				Protocol: normalizeProtocol(e.Protocol),
			}
			if !p.hasDupe(px) {
				p.proxies = append(p.proxies, px)
				p.index[px.ID] = len(p.proxies) - 1
			}
		}
	}

	if len(cfg.Proxies) > 0 {
		parsed := ParseProxies(cfg.Proxies, cfg.Protocol)
		for _, px := range parsed {
			if !p.hasDupe(px) {
				p.proxies = append(p.proxies, px)
				p.index[px.ID] = len(p.proxies) - 1
			}
		}
	}

	p.dflt = &Consumer{
		pool:    p,
		name:    "_default",
		flagged: make(map[string]time.Time),
		inUse:   make(map[string]bool),
		flagDur: flagDur,
	}

	if cfg.Store != nil && (len(cfg.Entries) > 0 || len(cfg.Proxies) > 0) {
		if err := p.save(); err != nil {
			return nil, fmt.Errorf("proxypool: initial save: %w", err)
		}
	}

	return p, nil
}

func (p *Pool) hasDupe(px Proxy) bool {
	for _, existing := range p.proxies {
		if existing.Host == px.Host && existing.Port == px.Port && existing.Protocol == px.Protocol {
			return true
		}
	}
	return false
}

func (p *Pool) Add(raw ...string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	parsed := ParseProxies(raw, p.proto)
	added := false
	for _, px := range parsed {
		if !p.hasDupe(px) {
			p.proxies = append(p.proxies, px)
			p.index[px.ID] = len(p.proxies) - 1
			added = true
		}
	}

	if added && p.store != nil {
		return p.save()
	}
	return nil
}

func (p *Pool) AddEntries(entries ...Entry) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	added := false
	for _, e := range entries {
		px := Proxy{
			ID:       uuid.NewString(),
			Host:     e.Host,
			Port:     e.Port,
			Username: e.User,
			Password: e.Pass,
			Protocol: normalizeProtocol(e.Protocol),
		}
		if !p.hasDupe(px) {
			p.proxies = append(p.proxies, px)
			p.index[px.ID] = len(p.proxies) - 1
			added = true
		}
	}

	if added && p.store != nil {
		return p.save()
	}
	return nil
}

func (p *Pool) Remove(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	idx, ok := p.index[id]
	if !ok {
		return nil
	}

	last := len(p.proxies) - 1
	if idx != last {
		p.proxies[idx] = p.proxies[last]
		p.index[p.proxies[idx].ID] = idx
	}
	p.proxies = p.proxies[:last]
	delete(p.index, id)
	delete(p.dead, id)

	if p.store != nil {
		return p.save()
	}
	return nil
}

func (p *Pool) Len() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.proxies)
}

func (p *Pool) All() []Proxy {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]Proxy, len(p.proxies))
	copy(out, p.proxies)
	return out
}

func (p *Pool) Next() (Proxy, error)    { return p.dflt.Next() }
func (p *Pool) Flag(id string)          { p.dflt.Flag(id) }
func (p *Pool) Release(id string)       { p.dflt.Release(id) }

func (p *Pool) Random() (Proxy, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var alive []Proxy
	for _, px := range p.proxies {
		if !p.dead[px.ID] {
			alive = append(alive, px)
		}
	}
	if len(alive) == 0 {
		return Proxy{}, fmt.Errorf("proxypool: no alive proxies")
	}
	return alive[rand.IntN(len(alive))], nil
}

func (p *Pool) MarkDead(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.dead[id] = true
}

func (p *Pool) Revive(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.dead, id)
}

func (p *Pool) Consumer(name string) *Consumer {
	return &Consumer{
		pool:    p,
		name:    name,
		flagged: make(map[string]time.Time),
		inUse:   make(map[string]bool),
		flagDur: p.flagDur,
	}
}

func (p *Pool) Close() error {
	if p.store != nil {
		p.mu.RLock()
		defer p.mu.RUnlock()
		return p.save()
	}
	return nil
}

func (p *Pool) alive() []Proxy {
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := make([]Proxy, 0, len(p.proxies))
	for _, px := range p.proxies {
		if !p.dead[px.ID] {
			result = append(result, px)
		}
	}
	return result
}

func (p *Pool) save() error {
	if p.store == nil {
		return nil
	}
	out := make([]Proxy, len(p.proxies))
	copy(out, p.proxies)
	return p.store.Save(out)
}
