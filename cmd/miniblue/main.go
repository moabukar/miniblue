package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/moabukar/miniblue/internal/server"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	port := os.Getenv("PORT")
	if port == "" {
		port = "4566"
	}
	tlsPort := os.Getenv("TLS_PORT")
	if tlsPort == "" {
		tlsPort = "4567"
	}

	srv := server.New()
	handler := srv.Handler()

	// Generate cert and save to ~/.miniblue/
	certDir := certDirectory()
	cert, certPEM, err := generateAndSaveCert(certDir)
	if err != nil {
		log.Fatalf("Failed to generate cert: %v", err)
	}

	certPath := filepath.Join(certDir, "cert.pem")
	log.Printf("miniblue certificate saved to %s", certPath)
	log.Printf("  Trust it:  export SSL_CERT_FILE=%s", certPath)
	log.Printf("  Or on Mac: sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain %s", certPath)
	log.Printf("  Or on Linux: sudo cp %s /usr/local/share/ca-certificates/miniblue.crt && sudo update-ca-certificates", certPath)

	// Graceful shutdown on SIGINT/SIGTERM
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start HTTP
	httpServer := &http.Server{Addr: ":" + port, Handler: handler}
	go func() {
		log.Printf("miniblue HTTP  on http://localhost:%s", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Start HTTPS
	log.Printf("miniblue HTTPS on https://localhost:%s", tlsPort)
	_ = certPEM // already saved to disk
	tlsServer := &http.Server{
		Addr:    ":" + tlsPort,
		Handler: handler,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
		},
	}
	go func() {
		if err := tlsServer.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTPS server error: %v", err)
		}
	}()

	// Block until shutdown signal
	<-stop
	log.Println("miniblue shutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	httpServer.Shutdown(ctx)
	tlsServer.Shutdown(ctx)
	log.Println("miniblue stopped")
}

func certDirectory() string {
	dir := os.Getenv("LOCAL_AZURE_CERT_DIR")
	if dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".miniblue")
}

func generateAndSaveCert(dir string) (tls.Certificate, []byte, error) {
	os.MkdirAll(dir, 0700)

	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")

	// Reuse existing cert if it's still valid
	if certPEM, err := os.ReadFile(certPath); err == nil {
		if keyPEM, err := os.ReadFile(keyPath); err == nil {
			if tlsCert, err := tls.X509KeyPair(certPEM, keyPEM); err == nil {
				// Check expiry
				if leaf, err := x509.ParseCertificate(tlsCert.Certificate[0]); err == nil {
					if time.Now().Before(leaf.NotAfter.Add(-24 * time.Hour)) {
						log.Printf("Reusing existing certificate (expires %s)", leaf.NotAfter.Format("2006-01-02"))
						return tlsCert, certPEM, nil
					}
				}
			}
		}
	}

	// Generate new
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, nil, err
	}

	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"miniblue"},
			CommonName:   "localhost",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, _ := x509.MarshalECPrivateKey(key)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	// Save to disk
	os.WriteFile(certPath, certPEM, 0644)
	os.WriteFile(keyPath, keyPEM, 0600)

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	return tlsCert, certPEM, err
}
