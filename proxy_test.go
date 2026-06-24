package proxypool

import "testing"

func TestProxyURL_HTTP(t *testing.T) {
	p := Proxy{Host: "1.2.3.4", Port: "8080", Protocol: HTTP}
	want := "http://1.2.3.4:8080"
	if got := p.URL(); got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestProxyURL_WithAuth(t *testing.T) {
	p := Proxy{Host: "1.2.3.4", Port: "8080", Username: "user", Password: "pass", Protocol: HTTP}
	want := "http://user:pass@1.2.3.4:8080"
	if got := p.URL(); got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestProxyURL_SOCKS5(t *testing.T) {
	p := Proxy{Host: "1.2.3.4", Port: "1080", Username: "u", Password: "p", Protocol: SOCKS5}
	want := "socks5://u:p@1.2.3.4:1080"
	if got := p.URL(); got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestProxyURL_SOCKS4(t *testing.T) {
	p := Proxy{Host: "1.2.3.4", Port: "1080", Protocol: SOCKS4}
	want := "socks4://1.2.3.4:1080"
	if got := p.URL(); got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestProxyURL_HTTPSNormalized(t *testing.T) {
	p := Proxy{Host: "1.2.3.4", Port: "8080", Protocol: "https"}
	want := "http://1.2.3.4:8080"
	if got := p.URL(); got != want {
		t.Errorf("https should normalize to http, got %s", got)
	}
}

func TestProxyString(t *testing.T) {
	p := Proxy{Host: "1.2.3.4", Port: "8080", Protocol: HTTP}
	if p.String() != p.URL() {
		t.Error("String() should equal URL()")
	}
}

func TestNormalizeProtocol(t *testing.T) {
	tests := []struct {
		in   Protocol
		want Protocol
	}{
		{"http", HTTP},
		{"https", HTTP},
		{"socks4", SOCKS4},
		{"socks5", SOCKS5},
		{"unknown", HTTP},
		{"", HTTP},
	}
	for _, tt := range tests {
		if got := normalizeProtocol(tt.in); got != tt.want {
			t.Errorf("normalizeProtocol(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
