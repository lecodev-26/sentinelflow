package saml

import (
	"context"
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	crewsaml "github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"
)

// Config contains the minimum explicit configuration required for a SAML 2.0 SP.
// Production never generates a keypair implicitly: certificate/key paths and IdP metadata are explicit.
type Config struct {
	EntityID         string
	PublicURL        string
	CertificateFile  string
	PrivateKeyFile   string
	IDPMetadataFile  string
	IDPMetadataURL   string
	AllowedClockSkew time.Duration
}

type Service struct {
	mu       sync.RWMutex
	cfg      Config
	sp       *crewsaml.ServiceProvider
	requests map[string]time.Time
}

func New(cfg Config) (*Service, error) {
	if cfg.EntityID == "" || cfg.PublicURL == "" {
		return nil, errors.New("SAML entity_id and public_url are required")
	}
	if cfg.CertificateFile == "" || cfg.PrivateKeyFile == "" {
		return nil, errors.New("SAML certificate_file and private_key_file are required")
	}
	certPair, err := tls.LoadX509KeyPair(cfg.CertificateFile, cfg.PrivateKeyFile)
	if err != nil {
		return nil, fmt.Errorf("load SAML keypair: %w", err)
	}
	if len(certPair.Certificate) == 0 {
		return nil, errors.New("SAML certificate is empty")
	}
	cert, err := x509.ParseCertificate(certPair.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("parse SAML certificate: %w", err)
	}
	metadata, err := loadMetadata(cfg)
	if err != nil {
		return nil, err
	}
	base, err := url.Parse(strings.TrimRight(cfg.PublicURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("invalid SAML public_url: %w", err)
	}
	sp := &crewsaml.ServiceProvider{
		EntityID: cfg.EntityID, Key: certPair.PrivateKey.(crypto.Signer), Certificate: cert,
		IDPMetadata: metadata, MetadataURL: *mustURL(base, "/auth/saml/metadata"),
		AcsURL: *mustURL(base, "/auth/saml/acs"), SloURL: *mustURL(base, "/auth/saml/slo"),
		AuthnNameIDFormat: crewsaml.UnspecifiedNameIDFormat,
	}
	return &Service{cfg: cfg, sp: sp, requests: make(map[string]time.Time)}, nil
}

func (s *Service) Metadata() *crewsaml.EntityDescriptor { return s.sp.Metadata() }
func (s *Service) LoginURL() (string, string, error) {
	if s.sp.IDPMetadata == nil || s.sp.IDPMetadata.IDPSSODescriptors == nil || len(s.sp.IDPMetadata.IDPSSODescriptors) == 0 {
		return "", "", errors.New("SAML IdP metadata has no SSO descriptor")
	}
	loc := s.sp.IDPMetadata.IDPSSODescriptors[0].SingleSignOnServices[0].Location
	req, err := s.sp.MakeAuthenticationRequest(loc, "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect", "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST")
	if err != nil {
		return "", "", err
	}
	u, err := req.Redirect("", s.sp)
	if err != nil {
		return "", "", err
	}
	s.mu.Lock()
	s.requests[req.ID] = time.Now()
	s.mu.Unlock()
	return u.String(), req.ID, nil
}
func (s *Service) ParseResponse(r *http.Request) (*crewsaml.Assertion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	ids := make([]string, 0, len(s.requests))
	for id, created := range s.requests {
		if now.Sub(created) > 10*time.Minute {
			delete(s.requests, id)
			continue
		}
		ids = append(ids, id)
	}
	assertion, err := s.sp.ParseResponse(r, ids)
	if err != nil {
		return nil, err
	}
	if assertion == nil || assertion.Subject == nil || assertion.Subject.NameID == nil || strings.TrimSpace(assertion.Subject.NameID.Value) == "" {
		return nil, errors.New("SAML assertion missing NameID")
	}
	for id := range s.requests {
		delete(s.requests, id)
	}
	return assertion, nil
}

func loadMetadata(cfg Config) (*crewsaml.EntityDescriptor, error) {
	if cfg.IDPMetadataFile != "" {
		b, err := os.ReadFile(cfg.IDPMetadataFile)
		if err != nil {
			return nil, fmt.Errorf("read SAML IdP metadata: %w", err)
		}
		m, err := samlsp.ParseMetadata(b)
		if err != nil {
			return nil, fmt.Errorf("parse SAML IdP metadata: %w", err)
		}
		return m, nil
	}
	if cfg.IDPMetadataURL != "" {
		u, err := url.Parse(cfg.IDPMetadataURL)
		if err != nil {
			return nil, err
		}
		m, err := samlsp.FetchMetadata(context.Background(), http.DefaultClient, *u)
		if err != nil {
			return nil, fmt.Errorf("fetch SAML IdP metadata: %w", err)
		}
		return m, nil
	}
	return nil, errors.New("SAML IdP metadata_file or metadata_url is required")
}
func mustURL(base *url.URL, path string) *url.URL { u := *base; u.Path = path; return &u }
