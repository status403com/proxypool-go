package sqlitestore

import (
	"path/filepath"
	"testing"

	proxypool "github.com/status403com/proxypool-go"
)

func TestSQLiteStore_SaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	store, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	proxies := []proxypool.Proxy{
		{ID: "1", Host: "1.2.3.4", Port: "8080", Protocol: proxypool.HTTP},
		{ID: "2", Host: "5.6.7.8", Port: "3128", Username: "u", Password: "p", Protocol: proxypool.SOCKS5},
	}

	if err := store.Save(proxies); err != nil {
		t.Fatal(err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}

	if len(loaded) != 2 {
		t.Fatalf("expected 2, got %d", len(loaded))
	}
	if loaded[0].Host != "1.2.3.4" {
		t.Errorf("expected 1.2.3.4, got %s", loaded[0].Host)
	}
	if loaded[1].Username != "u" {
		t.Errorf("expected user u, got %s", loaded[1].Username)
	}
	if loaded[1].Protocol != proxypool.SOCKS5 {
		t.Errorf("expected socks5, got %s", loaded[1].Protocol)
	}
}

func TestSQLiteStore_LoadEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.db")

	store, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 0 {
		t.Errorf("expected empty, got %d", len(loaded))
	}
}

func TestSQLiteStore_SaveOverwrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "overwrite.db")

	store, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	store.Save([]proxypool.Proxy{
		{ID: "1", Host: "1.2.3.4", Port: "8080", Protocol: proxypool.HTTP},
		{ID: "2", Host: "5.6.7.8", Port: "3128", Protocol: proxypool.HTTP},
	})

	store.Save([]proxypool.Proxy{
		{ID: "3", Host: "9.10.11.12", Port: "1080", Protocol: proxypool.SOCKS5},
	})

	loaded, _ := store.Load()
	if len(loaded) != 1 {
		t.Fatalf("expected 1 after overwrite, got %d", len(loaded))
	}
	if loaded[0].Host != "9.10.11.12" {
		t.Errorf("expected 9.10.11.12, got %s", loaded[0].Host)
	}
}

func TestSQLiteStore_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "roundtrip.db")

	store1, _ := New(path)
	store1.Save([]proxypool.Proxy{
		{ID: "a", Host: "1.2.3.4", Port: "8080", Username: "u", Password: "p", Protocol: proxypool.HTTP},
	})
	store1.Close()

	store2, _ := New(path)
	defer store2.Close()

	loaded, _ := store2.Load()
	if len(loaded) != 1 {
		t.Fatalf("expected 1, got %d", len(loaded))
	}
	if loaded[0].Username != "u" || loaded[0].Password != "p" {
		t.Error("credentials not persisted across close/reopen")
	}
}

func TestSQLiteStore_PoolIntegration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pool.db")

	store, _ := New(path)
	defer store.Close()

	pool, err := proxypool.New(proxypool.Config{
		Proxies: []string{"1.2.3.4:8080", "5.6.7.8:3128"},
		Store:   store,
	})
	if err != nil {
		t.Fatal(err)
	}
	pool.Close()

	loaded, _ := store.Load()
	if len(loaded) != 2 {
		t.Errorf("expected 2 persisted, got %d", len(loaded))
	}
}
