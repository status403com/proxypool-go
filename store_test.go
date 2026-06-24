package proxypool

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJSONStore_SaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "proxies.json")
	store := NewJSONStore(path)

	proxies := []Proxy{
		{ID: "1", Host: "1.2.3.4", Port: "8080", Protocol: HTTP},
		{ID: "2", Host: "5.6.7.8", Port: "3128", Username: "u", Password: "p", Protocol: SOCKS5},
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
	if loaded[1].Protocol != SOCKS5 {
		t.Errorf("expected socks5, got %s", loaded[1].Protocol)
	}
}

func TestJSONStore_LoadMissing(t *testing.T) {
	store := NewJSONStore(filepath.Join(t.TempDir(), "nonexistent.json"))
	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 0 {
		t.Errorf("expected empty, got %d", len(loaded))
	}
}

func TestJSONStore_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "proxies.json")
	store := NewJSONStore(path)

	proxies := []Proxy{{ID: "1", Host: "1.2.3.4", Port: "8080", Protocol: HTTP}}
	if err := store.Save(proxies); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty file")
	}
}

func TestPool_WithJSONStore_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "proxies.json")

	pool1, err := New(Config{
		Proxies: []string{"1.2.3.4:8080", "5.6.7.8:3128"},
		Store:   NewJSONStore(path),
	})
	if err != nil {
		t.Fatal(err)
	}
	pool1.Close()

	pool2, err := New(Config{
		Store: NewJSONStore(path),
	})
	if err != nil {
		t.Fatal(err)
	}

	if pool2.Len() != 2 {
		t.Errorf("expected 2 after reload, got %d", pool2.Len())
	}

	all := pool2.All()
	hosts := map[string]bool{all[0].Host: true, all[1].Host: true}
	if !hosts["1.2.3.4"] || !hosts["5.6.7.8"] {
		t.Errorf("unexpected hosts after reload: %v", all)
	}
}

func TestPool_WithJSONStore_AddPersists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "proxies.json")

	pool1, _ := New(Config{
		Proxies: []string{"1.2.3.4:8080"},
		Store:   NewJSONStore(path),
	})
	pool1.Add("5.6.7.8:3128")
	pool1.Close()

	pool2, _ := New(Config{Store: NewJSONStore(path)})
	if pool2.Len() != 2 {
		t.Errorf("expected 2 after add+reload, got %d", pool2.Len())
	}
}

func TestPool_WithJSONStore_RemovePersists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "proxies.json")

	pool1, _ := New(Config{
		Proxies: []string{"1.2.3.4:8080", "5.6.7.8:3128"},
		Store:   NewJSONStore(path),
	})
	all := pool1.All()
	pool1.Remove(all[0].ID)
	pool1.Close()

	pool2, _ := New(Config{Store: NewJSONStore(path)})
	if pool2.Len() != 1 {
		t.Errorf("expected 1 after remove+reload, got %d", pool2.Len())
	}
}
