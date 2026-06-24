package proxypool

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Store interface {
	Load() ([]Proxy, error)
	Save(proxies []Proxy) error
}

type JSONStore struct {
	path string
}

func NewJSONStore(path string) *JSONStore {
	return &JSONStore{path: path}
}

func (s *JSONStore) Load() ([]Proxy, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var proxies []Proxy
	if err := json.Unmarshal(data, &proxies); err != nil {
		return nil, err
	}
	return proxies, nil
}

func (s *JSONStore) Save(proxies []Proxy) error {
	data, err := json.MarshalIndent(proxies, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(s.path)
	tmp, err := os.CreateTemp(dir, ".proxypool-*.json")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}

	return os.Rename(tmpPath, s.path)
}
