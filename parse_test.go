package proxypool

import "testing"

func TestParseProxies_HostPort(t *testing.T) {
	result := ParseProxies([]string{"1.2.3.4:8080"}, HTTP)
	if len(result) != 1 {
		t.Fatalf("expected 1, got %d", len(result))
	}
	p := result[0]
	if p.Host != "1.2.3.4" || p.Port != "8080" {
		t.Errorf("got host=%s port=%s", p.Host, p.Port)
	}
	if p.Username != "" || p.Password != "" {
		t.Errorf("expected no auth, got user=%s pass=%s", p.Username, p.Password)
	}
	if p.Protocol != HTTP {
		t.Errorf("expected HTTP, got %s", p.Protocol)
	}
	if p.ID == "" {
		t.Error("expected ID to be set")
	}
}

func TestParseProxies_HostPortUserPass(t *testing.T) {
	result := ParseProxies([]string{"1.2.3.4:8080:user:pass"}, HTTP)
	if len(result) != 1 {
		t.Fatalf("expected 1, got %d", len(result))
	}
	p := result[0]
	if p.Host != "1.2.3.4" || p.Port != "8080" {
		t.Errorf("got host=%s port=%s", p.Host, p.Port)
	}
	if p.Username != "user" || p.Password != "pass" {
		t.Errorf("got user=%s pass=%s", p.Username, p.Password)
	}
}

func TestParseProxies_UserPassAtHostPort(t *testing.T) {
	result := ParseProxies([]string{"user:pass@1.2.3.4:8080"}, SOCKS5)
	if len(result) != 1 {
		t.Fatalf("expected 1, got %d", len(result))
	}
	p := result[0]
	if p.Host != "1.2.3.4" || p.Port != "8080" {
		t.Errorf("got host=%s port=%s", p.Host, p.Port)
	}
	if p.Username != "user" || p.Password != "pass" {
		t.Errorf("got user=%s pass=%s", p.Username, p.Password)
	}
	if p.Protocol != SOCKS5 {
		t.Errorf("expected SOCKS5, got %s", p.Protocol)
	}
}

func TestParseProxies_UserPassHostPort(t *testing.T) {
	result := ParseProxies([]string{"user:pass:1.2.3.4:8080"}, HTTP)
	if len(result) != 1 {
		t.Fatalf("expected 1, got %d", len(result))
	}
	p := result[0]
	if p.Host != "1.2.3.4" || p.Port != "8080" {
		t.Errorf("got host=%s port=%s", p.Host, p.Port)
	}
	if p.Username != "user" || p.Password != "pass" {
		t.Errorf("got user=%s pass=%s", p.Username, p.Password)
	}
}

func TestParseProxies_URLScheme(t *testing.T) {
	raw := []string{
		"socks5://user:pass@1.2.3.4:8080",
		"http://1.2.3.4:3128",
		"socks4://user:pass@5.6.7.8:1080",
	}
	result := ParseProxies(raw, HTTP)
	if len(result) != 3 {
		t.Fatalf("expected 3, got %d", len(result))
	}

	if result[0].Protocol != SOCKS5 {
		t.Errorf("[0] expected SOCKS5, got %s", result[0].Protocol)
	}
	if result[0].Username != "user" {
		t.Errorf("[0] expected user, got %s", result[0].Username)
	}
	if result[1].Protocol != HTTP {
		t.Errorf("[1] expected HTTP, got %s", result[1].Protocol)
	}
	if result[1].Username != "" {
		t.Errorf("[1] expected no user, got %s", result[1].Username)
	}
	if result[2].Protocol != SOCKS4 {
		t.Errorf("[2] expected SOCKS4, got %s", result[2].Protocol)
	}
}

func TestParseProxies_HTTPSNormalized(t *testing.T) {
	result := ParseProxies([]string{"https://1.2.3.4:8080"}, HTTP)
	if len(result) != 1 {
		t.Fatalf("expected 1, got %d", len(result))
	}
	if result[0].Protocol != HTTP {
		t.Errorf("expected HTTP, got %s", result[0].Protocol)
	}
}

func TestParseProxies_EmptyAndWhitespace(t *testing.T) {
	raw := []string{"", "  ", "  1.2.3.4:8080  ", ""}
	result := ParseProxies(raw, HTTP)
	if len(result) != 1 {
		t.Fatalf("expected 1, got %d", len(result))
	}
	if result[0].Host != "1.2.3.4" {
		t.Errorf("expected 1.2.3.4, got %s", result[0].Host)
	}
}

func TestParseProxies_MixedFormats(t *testing.T) {
	raw := []string{
		"1.2.3.4:8080",
		"user:pass@5.6.7.8:3128",
		"socks5://admin:secret@9.10.11.12:1080",
		"10.0.0.1:8080:u:p",
	}
	result := ParseProxies(raw, HTTP)
	if len(result) != 4 {
		t.Fatalf("expected 4, got %d", len(result))
	}
}

func TestParseProxies_InvalidSkipped(t *testing.T) {
	raw := []string{"not-a-proxy", "1.2.3.4:8080"}
	result := ParseProxies(raw, HTTP)
	if len(result) != 1 {
		t.Fatalf("expected 1 (invalid skipped), got %d", len(result))
	}
}
