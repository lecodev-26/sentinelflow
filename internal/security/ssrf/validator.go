package ssrf

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// Validator valida URLs para prevenir SSRF
type Validator struct {
	allowedHosts []string
	blockPrivate bool
	blockLocal   bool
}

// NewValidator crea un validador por defecto
func NewValidator() *Validator {
	return &Validator{
		allowedHosts: []string{},
		blockPrivate: true,
		blockLocal:   true,
	}
}

// Validate verifica que una URL sea segura
func (v *Validator) Validate(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Solo http y https
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("scheme not allowed: %s", u.Scheme)
	}

	host := u.Hostname()

	// Bloquear localhost
	if v.blockLocal {
		if host == "localhost" || host == "127.0.0.1" || host == "::1" || host == "0.0.0.0" {
			return fmt.Errorf("localhost not allowed")
		}
	}

	// Resolver IP
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("cannot resolve host: %w", err)
	}

	for _, ip := range ips {
		if v.blockPrivate && isPrivateIP(ip) {
			return fmt.Errorf("private IP not allowed: %s", ip.String())
		}
		// Metadata endpoints de cloud providers
		if isCloudMetadata(ip) {
			return fmt.Errorf("cloud metadata endpoint blocked")
		}
	}

	// Allowlist si está definida
	if len(v.allowedHosts) > 0 {
		allowed := false
		for _, h := range v.allowedHosts {
			if strings.EqualFold(host, h) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("host not in allowlist: %s", host)
		}
	}

	return nil
}

// isPrivateIP verifica si una IP es privada
func isPrivateIP(ip net.IP) bool {
	if ip4 := ip.To4(); ip4 != nil {
		// 10.0.0.0/8
		if ip4[0] == 10 {
			return true
		}
		// 172.16.0.0/12
		if ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31 {
			return true
		}
		// 192.168.0.0/16
		if ip4[0] == 192 && ip4[1] == 168 {
			return true
		}
		// 169.254.0.0/16 (link-local)
		if ip4[0] == 169 && ip4[1] == 254 {
			return true
		}
		// 127.0.0.0/8
		if ip4[0] == 127 {
			return true
		}
	}
	// IPv6 loopback
	if ip.Equal(net.IPv6loopback) {
		return true
	}
	return false
}

// isCloudMetadata verifica IPs de metadata de cloud
func isCloudMetadata(ip net.IP) bool {
	metadataIPs := []string{
		"169.254.169.254", // AWS, GCP, Azure
		"169.254.170.2",   // ECS
		"fd00:ec2::254",   // AWS IPv6
	}
	for _, m := range metadataIPs {
		if ip.String() == m {
			return true
		}
	}
	return false
}

// AddAllowedHost añade un host a la allowlist
func (v *Validator) AddAllowedHost(host string) {
	v.allowedHosts = append(v.allowedHosts, host)
}
