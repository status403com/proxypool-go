package proxypool

import (
	"net/url"
	"strings"

	"github.com/google/uuid"
)

func ParseProxies(raw []string, defaultProtocol Protocol) []Proxy {
	proto := normalizeProtocol(defaultProtocol)
	proxies := make([]Proxy, 0, len(raw))

	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		p := Proxy{
			ID:       uuid.NewString(),
			Protocol: proto,
		}

		if strings.Contains(line, "://") {
			u, err := url.Parse(line)
			if err == nil && u.Hostname() != "" {
				p.Host = u.Hostname()
				p.Port = u.Port()
				if u.User != nil {
					p.Username = u.User.Username()
					p.Password, _ = u.User.Password()
				}
				p.Protocol = normalizeProtocol(Protocol(u.Scheme))
				if p.Host != "" && p.Port != "" {
					proxies = append(proxies, p)
				}
				continue
			}
		}

		if atIdx := strings.Index(line, "@"); atIdx > 0 {
			auth := line[:atIdx]
			hostPort := line[atIdx+1:]
			parts := strings.SplitN(auth, ":", 2)
			if len(parts) == 2 {
				p.Username = parts[0]
				p.Password = parts[1]
			}
			hp := strings.SplitN(hostPort, ":", 2)
			if len(hp) == 2 {
				p.Host = hp[0]
				p.Port = hp[1]
			}
		} else {
			parts := strings.Split(line, ":")
			switch len(parts) {
			case 2:
				p.Host = parts[0]
				p.Port = parts[1]
			case 4:
				if isPort(parts[1]) {
					p.Host = parts[0]
					p.Port = parts[1]
					p.Username = parts[2]
					p.Password = parts[3]
				} else {
					p.Username = parts[0]
					p.Password = parts[1]
					p.Host = parts[2]
					p.Port = parts[3]
				}
			}
		}

		if p.Host != "" && p.Port != "" {
			proxies = append(proxies, p)
		}
	}

	return proxies
}

func isPort(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0 && len(s) <= 5
}
