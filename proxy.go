package proxypool

import "fmt"

type Protocol string

const (
	HTTP   Protocol = "http"
	SOCKS4 Protocol = "socks4"
	SOCKS5 Protocol = "socks5"
)

func normalizeProtocol(p Protocol) Protocol {
	switch p {
	case SOCKS4, SOCKS5:
		return p
	default:
		return HTTP
	}
}

type Entry struct {
	Host     string
	Port     string
	User     string
	Pass     string
	Protocol Protocol
}

type Proxy struct {
	ID       string   `json:"id"`
	Host     string   `json:"host"`
	Port     string   `json:"port"`
	Username string   `json:"username,omitempty"`
	Password string   `json:"password,omitempty"`
	Protocol Protocol `json:"protocol"`
}

func (p Proxy) URL() string {
	proto := normalizeProtocol(p.Protocol)
	auth := ""
	if p.Username != "" {
		auth = p.Username + ":" + p.Password + "@"
	}
	return fmt.Sprintf("%s://%s%s:%s", proto, auth, p.Host, p.Port)
}

func (p Proxy) String() string { return p.URL() }
