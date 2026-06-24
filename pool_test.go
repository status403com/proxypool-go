package proxypool

import (
	"sync"
	"testing"
)

func TestNew_InMemory(t *testing.T) {
	pool, err := New(Config{
		Proxies: []string{"1.2.3.4:8080", "5.6.7.8:3128"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if pool.Len() != 2 {
		t.Errorf("expected 2, got %d", pool.Len())
	}
}

func TestNew_WithEntries(t *testing.T) {
	pool, err := New(Config{
		Entries: []Entry{
			{Host: "1.2.3.4", Port: "8080", Protocol: HTTP},
			{Host: "5.6.7.8", Port: "3128", User: "u", Pass: "p", Protocol: SOCKS5},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if pool.Len() != 2 {
		t.Errorf("expected 2, got %d", pool.Len())
	}
	all := pool.All()
	if all[1].Username != "u" {
		t.Errorf("expected user u, got %s", all[1].Username)
	}
}

func TestNew_Dedup(t *testing.T) {
	pool, err := New(Config{
		Proxies: []string{"1.2.3.4:8080", "1.2.3.4:8080", "5.6.7.8:3128"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if pool.Len() != 2 {
		t.Errorf("expected 2 (deduped), got %d", pool.Len())
	}
}

func TestNew_MergeEntriesAndStrings(t *testing.T) {
	pool, err := New(Config{
		Entries: []Entry{{Host: "1.2.3.4", Port: "8080", Protocol: HTTP}},
		Proxies: []string{"5.6.7.8:3128", "1.2.3.4:8080"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if pool.Len() != 2 {
		t.Errorf("expected 2 (deduped across entries+strings), got %d", pool.Len())
	}
}

func TestPool_Add(t *testing.T) {
	pool, _ := New(Config{Proxies: []string{"1.2.3.4:8080"}})
	if err := pool.Add("5.6.7.8:3128"); err != nil {
		t.Fatal(err)
	}
	if pool.Len() != 2 {
		t.Errorf("expected 2, got %d", pool.Len())
	}
}

func TestPool_Add_Dedup(t *testing.T) {
	pool, _ := New(Config{Proxies: []string{"1.2.3.4:8080"}})
	pool.Add("1.2.3.4:8080")
	if pool.Len() != 1 {
		t.Errorf("expected 1 (no dup), got %d", pool.Len())
	}
}

func TestPool_AddEntries(t *testing.T) {
	pool, _ := New(Config{})
	pool.AddEntries(Entry{Host: "1.2.3.4", Port: "8080", Protocol: HTTP})
	if pool.Len() != 1 {
		t.Errorf("expected 1, got %d", pool.Len())
	}
}

func TestPool_Remove(t *testing.T) {
	pool, _ := New(Config{Proxies: []string{"1.2.3.4:8080", "5.6.7.8:3128"}})
	all := pool.All()
	pool.Remove(all[0].ID)
	if pool.Len() != 1 {
		t.Errorf("expected 1, got %d", pool.Len())
	}
}

func TestPool_Remove_NonExistent(t *testing.T) {
	pool, _ := New(Config{Proxies: []string{"1.2.3.4:8080"}})
	if err := pool.Remove("nonexistent"); err != nil {
		t.Errorf("remove nonexistent should not error, got %v", err)
	}
}

func TestPool_MarkDead(t *testing.T) {
	pool, _ := New(Config{Proxies: []string{"1.2.3.4:8080"}})
	all := pool.All()
	pool.MarkDead(all[0].ID)

	_, err := pool.Next()
	if err == nil {
		t.Error("expected error: only proxy is dead")
	}
}

func TestPool_Revive(t *testing.T) {
	pool, _ := New(Config{Proxies: []string{"1.2.3.4:8080"}})
	all := pool.All()
	pool.MarkDead(all[0].ID)
	pool.Revive(all[0].ID)

	px, err := pool.Next()
	if err != nil {
		t.Fatal(err)
	}
	if px.Host != "1.2.3.4" {
		t.Errorf("expected 1.2.3.4, got %s", px.Host)
	}
}

func TestPool_Random(t *testing.T) {
	pool, _ := New(Config{Proxies: []string{"1.2.3.4:8080", "5.6.7.8:3128"}})
	for range 10 {
		_, err := pool.Random()
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestPool_Random_Empty(t *testing.T) {
	pool, _ := New(Config{})
	_, err := pool.Random()
	if err == nil {
		t.Error("expected error on empty pool")
	}
}

func TestPool_Random_SkipsDead(t *testing.T) {
	pool, _ := New(Config{Proxies: []string{"1.2.3.4:8080", "5.6.7.8:3128"}})
	all := pool.All()
	pool.MarkDead(all[0].ID)

	for range 20 {
		px, err := pool.Random()
		if err != nil {
			t.Fatal(err)
		}
		if px.ID == all[0].ID {
			t.Error("Random() returned dead proxy")
		}
	}
}

func TestPool_All_Snapshot(t *testing.T) {
	pool, _ := New(Config{Proxies: []string{"1.2.3.4:8080"}})
	all := pool.All()
	all[0].Host = "modified"

	fresh := pool.All()
	if fresh[0].Host != "1.2.3.4" {
		t.Error("All() should return a copy, not a reference")
	}
}

func TestPool_ThreadSafety(t *testing.T) {
	pool, _ := New(Config{
		Proxies: []string{"1.2.3.4:8080", "5.6.7.8:3128", "9.10.11.12:1080"},
	})

	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pool.Next()
			pool.Random()
			pool.All()
			pool.Len()
		}()
	}
	wg.Wait()
}

func TestPool_Empty(t *testing.T) {
	pool, err := New(Config{})
	if err != nil {
		t.Fatal(err)
	}
	if pool.Len() != 0 {
		t.Errorf("expected 0, got %d", pool.Len())
	}
	_, err = pool.Next()
	if err == nil {
		t.Error("expected error on empty pool")
	}
}
