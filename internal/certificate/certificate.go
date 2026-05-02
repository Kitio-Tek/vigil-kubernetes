/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package certificate manages TLS certificates for PostgreSQL clusters. It
// generates self-signed CA and server certificates, validates existing
// certificates for expiry, and produces the Kubernetes Secret manifests that
// the operator mounts into cluster pods.
package certificate

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

const (
	// DefaultCertDuration is the validity period for generated certificates.
	DefaultCertDuration = 365 * 24 * time.Hour

	// DefaultRenewBefore is how far in advance the operator renews certificates.
	DefaultRenewBefore = 30 * 24 * time.Hour

	// CACommonNameSuffix is appended to the cluster name to form the CA CN.
	CACommonNameSuffix = "-ca"

	// ServerCommonNameSuffix is appended to the cluster name for the server cert.
	ServerCommonNameSuffix = "-server"
)

// KeyPair holds a PEM-encoded TLS certificate and private key.
type KeyPair struct {
	// CertPEM is the PEM-encoded certificate (or certificate chain).
	CertPEM []byte
	// KeyPEM is the PEM-encoded private key.
	KeyPEM []byte
}

// GenerateCA creates a self-signed CA certificate and key for the named cluster.
func GenerateCA(clusterName string, duration time.Duration) (*KeyPair, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating CA key: %w", err)
	}

	serial, err := randomSerial()
	if err != nil {
		return nil, err
	}

	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   clusterName + CACommonNameSuffix,
			Organization: []string{"vigil-kubernetes"},
		},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(duration),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, fmt.Errorf("creating CA certificate: %w", err)
	}

	return pemEncode(certDER, key)
}

// GenerateServerCert generates a server certificate signed by the provided CA
// for the given DNS SANs. The CA key pair must include both the certificate
// and private key in PEM format.
func GenerateServerCert(clusterName string, sans []string, ca *KeyPair, duration time.Duration) (*KeyPair, error) {
	caCert, caKey, err := decodePEM(ca)
	if err != nil {
		return nil, fmt.Errorf("decoding CA: %w", err)
	}

	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating server key: %w", err)
	}

	serial, err := randomSerial()
	if err != nil {
		return nil, err
	}

	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   clusterName + ServerCommonNameSuffix,
			Organization: []string{"vigil-kubernetes"},
		},
		DNSNames:  sans,
		NotBefore: time.Now().Add(-time.Minute),
		NotAfter:  time.Now().Add(duration),
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, caCert, &serverKey.PublicKey, caKey)
	if err != nil {
		return nil, fmt.Errorf("creating server certificate: %w", err)
	}

	return pemEncode(certDER, serverKey)
}

// NeedsRenewal returns true when the certificate will expire within the
// renewBefore window. It parses the first certificate in certPEM.
func NeedsRenewal(certPEM []byte, renewBefore time.Duration) (bool, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return false, fmt.Errorf("no PEM block found in certificate data")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return false, fmt.Errorf("parsing certificate: %w", err)
	}
	return time.Now().Add(renewBefore).After(cert.NotAfter), nil
}

// ExpiresAt parses the NotAfter field of the first certificate in certPEM.
func ExpiresAt(certPEM []byte) (time.Time, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return time.Time{}, fmt.Errorf("no PEM block found")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing certificate: %w", err)
	}
	return cert.NotAfter, nil
}

// ServerSANs returns the DNS SANs that the server certificate for a cluster
// should cover, given the cluster name, namespace, and headless service name.
func ServerSANs(clusterName, namespace, headlessSvcName string) []string {
	return []string{
		clusterName,
		clusterName + "." + namespace,
		clusterName + "." + namespace + ".svc",
		clusterName + "." + namespace + ".svc.cluster.local",
		"*." + headlessSvcName + "." + namespace + ".svc.cluster.local",
		"localhost",
	}
}

func randomSerial() (*big.Int, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("generating serial number: %w", err)
	}
	return serial, nil
}

func pemEncode(certDER []byte, key *ecdsa.PrivateKey) (*KeyPair, error) {
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("marshalling private key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return &KeyPair{CertPEM: certPEM, KeyPEM: keyPEM}, nil
}

func decodePEM(kp *KeyPair) (*x509.Certificate, *ecdsa.PrivateKey, error) {
	certBlock, _ := pem.Decode(kp.CertPEM)
	if certBlock == nil {
		return nil, nil, fmt.Errorf("no certificate PEM block")
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing certificate: %w", err)
	}

	keyBlock, _ := pem.Decode(kp.KeyPEM)
	if keyBlock == nil {
		return nil, nil, fmt.Errorf("no key PEM block")
	}
	key, err := x509.ParseECPrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing private key: %w", err)
	}
	return cert, key, nil
}
