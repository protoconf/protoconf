package utils

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// TLSFiles holds the paths and/or inline PEM text for TLS certificate, key, and CA.
// File paths and inline PEM text are mutually exclusive per field pair:
// if both CertFile and CertText are set, CertFile takes precedence.
type TLSFiles struct {
	// CertFile is the path to the PEM-encoded certificate file.
	CertFile string
	// CertText is the PEM-encoded certificate inline text.
	CertText string
	// KeyFile is the path to the PEM-encoded private key file.
	KeyFile string
	// KeyText is the PEM-encoded private key inline text.
	KeyText string
	// CAFile is the path to the PEM-encoded CA certificate file.
	CAFile string
	// CAText is the PEM-encoded CA certificate inline text.
	CAText string
}

// BuildTLSConfig constructs a *tls.Config from the provided TLSFiles.
//
// Returns (nil, nil) when neither cert nor key is provided (no TLS configured).
// Returns an error if only one of cert/key is set, if files cannot be read,
// or if the cert/key pair or CA certificate cannot be parsed.
func BuildTLSConfig(f TLSFiles) (*tls.Config, error) {
	hasCert := f.CertFile != "" || f.CertText != ""
	hasKey := f.KeyFile != "" || f.KeyText != ""

	// No TLS configured — return nil to signal plaintext mode.
	if !hasCert && !hasKey {
		return nil, nil
	}

	// Partial configuration is an error.
	if hasCert != hasKey {
		return nil, fmt.Errorf("tls: cert and key must both be set")
	}

	// Read cert PEM.
	var certPEM []byte
	if f.CertFile != "" {
		data, err := os.ReadFile(f.CertFile)
		if err != nil {
			return nil, fmt.Errorf("tls: reading cert file %q: %w", f.CertFile, err)
		}
		certPEM = data
	} else {
		certPEM = []byte(f.CertText)
	}

	// Read key PEM.
	var keyPEM []byte
	if f.KeyFile != "" {
		data, err := os.ReadFile(f.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("tls: reading key file %q: %w", f.KeyFile, err)
		}
		keyPEM = data
	} else {
		keyPEM = []byte(f.KeyText)
	}

	// Parse the cert/key pair.
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("tls: parsing cert/key pair: %w", err)
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	// Optional CA configuration for mutual TLS.
	hasCA := f.CAFile != "" || f.CAText != ""
	if hasCA {
		var caPEM []byte
		if f.CAFile != "" {
			data, err := os.ReadFile(f.CAFile)
			if err != nil {
				return nil, fmt.Errorf("tls: reading CA file %q: %w", f.CAFile, err)
			}
			caPEM = data
		} else {
			caPEM = []byte(f.CAText)
		}

		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caPEM) {
			return nil, fmt.Errorf("tls: failed to parse CA certificate")
		}
		tlsCfg.ClientCAs = pool
		tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert
	}

	return tlsCfg, nil
}
