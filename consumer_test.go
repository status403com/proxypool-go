package proxypool

import (
	"testing"
	"time"
)

func TestConsumer_Next_ExcludesFlagged(t *testing.T) {
	pool, _ := New(Config{
		Proxies:      []string{"1.2.3.4:8080"},
		FlagDuration: 1 * time.Hour,
	})
	c := pool.Consumer("test")

	px, _ := c.Next()
	c.Flag(px.ID)
	c.Release(px.ID)

	_, err := c.Next()
	if err == nil {
		t.Error("expected error: only proxy is flagged")
	}
}

func TestConsumer_Next_FlagExpiry(t *testing.T) {
	pool, _ := New(Config{
		Proxies:      []string{"1.2.3.4:8080"},
		FlagDuration: 1 * time.Millisecond,
	})
	c := pool.Consumer("test")

	px, _ := c.Next()
	c.Flag(px.ID)
	c.Release(px.ID)

	time.Sleep(5 * time.Millisecond)

	_, err := c.Next()
	if err != nil {
		t.Errorf("expected flag to expire, got: %v", err)
	}
}

func TestConsumer_Next_PrefersUnused(t *testing.T) {
	pool, _ := New(Config{
		Proxies: []string{"1.2.3.4:8080", "5.6.7.8:3128"},
	})
	c := pool.Consumer("test")

	first, _ := c.Next()
	second, _ := c.Next()
	if second.ID == first.ID {
		t.Error("expected different proxy when unused one available")
	}
}

func TestConsumer_Next_FallbackToInUse(t *testing.T) {
	pool, _ := New(Config{
		Proxies: []string{"1.2.3.4:8080"},
	})
	c := pool.Consumer("test")

	first, _ := c.Next()
	second, _ := c.Next()

	if second.ID != first.ID {
		t.Error("expected same proxy when all in use")
	}
}

func TestConsumer_Isolation(t *testing.T) {
	pool, _ := New(Config{
		Proxies:      []string{"1.2.3.4:8080"},
		FlagDuration: 1 * time.Hour,
	})

	c1 := pool.Consumer("module-a")
	c2 := pool.Consumer("module-b")

	px, _ := c1.Next()
	c1.Flag(px.ID)
	c1.Release(px.ID)

	_, err := c1.Next()
	if err == nil {
		t.Error("c1 should not get flagged proxy")
	}

	px2, err := c2.Next()
	if err != nil {
		t.Errorf("c2 should get the proxy: %v", err)
	}
	if px2.Host != "1.2.3.4" {
		t.Errorf("expected 1.2.3.4, got %s", px2.Host)
	}
}

func TestConsumer_ReleaseAll(t *testing.T) {
	pool, _ := New(Config{
		Proxies: []string{"1.2.3.4:8080", "5.6.7.8:3128"},
	})
	c := pool.Consumer("test")

	c.Next()
	c.Next()

	if len(c.InUse()) != 2 {
		t.Errorf("expected 2 in use, got %d", len(c.InUse()))
	}

	c.ReleaseAll()

	if len(c.InUse()) != 0 {
		t.Errorf("expected 0 in use, got %d", len(c.InUse()))
	}
}

func TestConsumer_Next_NoProxies(t *testing.T) {
	pool, _ := New(Config{})
	c := pool.Consumer("test")

	_, err := c.Next()
	if err == nil {
		t.Error("expected error on empty pool")
	}
}

func TestConsumer_Next_SkipsDead(t *testing.T) {
	pool, _ := New(Config{
		Proxies: []string{"1.2.3.4:8080", "5.6.7.8:3128"},
	})
	c := pool.Consumer("test")
	all := pool.All()
	pool.MarkDead(all[0].ID)

	for range 10 {
		c.Release(all[1].ID)
		px, err := c.Next()
		if err != nil {
			t.Fatal(err)
		}
		if px.ID == all[0].ID {
			t.Error("consumer returned dead proxy")
		}
	}
}
