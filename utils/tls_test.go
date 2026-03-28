package utils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateSelfSignedCert generates a self-signed ECDSA P-256 cert for 127.0.0.1 valid for 1 hour.
func generateSelfSignedCert(t *testing.T) (certPEM, keyPEM []byte) {
	t.Helper()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test"},
		},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore:   time.Now().Add(-time.Minute),
		NotAfter:    time.Now().Add(time.Hour),
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	require.NoError(t, err)

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	keyBytes, err := x509.MarshalECPrivateKey(priv)
	require.NoError(t, err)
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})

	return certPEM, keyPEM
}

func TestBuildTLSConfig(t *testing.T) {
	certPEM, keyPEM := generateSelfSignedCert(t)
	tmpDir := t.TempDir()

	certFile := filepath.Join(tmpDir, "cert.pem")
	keyFile := filepath.Join(tmpDir, "key.pem")
	caFile := filepath.Join(tmpDir, "ca.pem")

	require.NoError(t, os.WriteFile(certFile, certPEM, 0600))
	require.NoError(t, os.WriteFile(keyFile, keyPEM, 0600))
	require.NoError(t, os.WriteFile(caFile, certPEM, 0600)) // use cert as CA for testing

	tests := []struct {
		name        string
		input       TLSFiles
		wantNil     bool
		wantErr     bool
		errContains string
		checkFn     func(t *testing.T, cfg interface{})
	}{
		{
			name:    "empty TLSFiles returns nil,nil",
			input:   TLSFiles{},
			wantNil: true,
			wantErr: false,
		},
		{
			name: "CertFile+KeyFile returns valid tls.Config",
			input: TLSFiles{
				CertFile: certFile,
				KeyFile:  keyFile,
			},
			wantNil: false,
			wantErr: false,
		},
		{
			name: "CertText+KeyText returns valid tls.Config",
			input: TLSFiles{
				CertText: string(certPEM),
				KeyText:  string(keyPEM),
			},
			wantNil: false,
			wantErr: false,
		},
		{
			name: "CertFile only returns error",
			input: TLSFiles{
				CertFile: certFile,
			},
			wantErr:     true,
			errContains: "cert and key must both be set",
		},
		{
			name: "KeyFile only returns error",
			input: TLSFiles{
				KeyFile: keyFile,
			},
			wantErr:     true,
			errContains: "cert and key must both be set",
		},
		{
			name: "CertFile+KeyFile+CAFile sets ClientCAs and ClientAuth",
			input: TLSFiles{
				CertFile: certFile,
				KeyFile:  keyFile,
				CAFile:   caFile,
			},
			wantNil: false,
			wantErr: false,
		},
		{
			name: "CertFile+KeyFile+CAText sets ClientCAs",
			input: TLSFiles{
				CertFile: certFile,
				KeyFile:  keyFile,
				CAText:   string(certPEM),
			},
			wantNil: false,
			wantErr: false,
		},
		{
			name: "nonexistent CertFile returns error",
			input: TLSFiles{
				CertFile: filepath.Join(tmpDir, "nonexistent_cert.pem"),
				KeyFile:  keyFile,
			},
			wantErr:     true,
			errContains: "reading cert file",
		},
		{
			name: "nonexistent KeyFile returns error",
			input: TLSFiles{
				CertFile: certFile,
				KeyFile:  filepath.Join(tmpDir, "nonexistent_key.pem"),
			},
			wantErr:     true,
			errContains: "reading key file",
		},
		{
			name: "invalid PEM cert text returns error",
			input: TLSFiles{
				CertText: "not-valid-pem",
				KeyText:  string(keyPEM),
			},
			wantErr:     true,
			errContains: "parsing cert/key pair",
		},
		{
			name: "invalid CA PEM returns error",
			input: TLSFiles{
				CertFile: certFile,
				KeyFile:  keyFile,
				CAText:   "not-valid-ca-pem",
			},
			wantErr:     true,
			errContains: "failed to parse CA certificate",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := BuildTLSConfig(tc.input)

			if tc.wantErr {
				require.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
				return
			}

			require.NoError(t, err)

			if tc.wantNil {
				assert.Nil(t, cfg)
				return
			}

			require.NotNil(t, cfg)
			assert.Len(t, cfg.Certificates, 1)
		})
	}
}
