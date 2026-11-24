package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)
func TestGetListener(t *testing.T) {
	tempdir := t.TempDir()
	unixAddr := filepath.Join(tempdir, "test.sock")

	t.Run("unix", func(t *testing.T) {
		cfg := MeasurementConfig{
			ListenerType: ListenerUnix,
			ListenerAddr: unixAddr,
		}
		ln, err := GetListener(cfg)
		if err != nil {
			t.Fatalf("Error setting up listener: %v", err)
		}
		defer ln.Close()
		if _, ok := ln.(*net.UnixListener); !ok {
			t.Errorf("Expected unix listener, got %T", ln)
		}
		if ln.Addr().String() != unixAddr {
			t.Errorf("Unexpected listener address.  Expected: %v, got %v", unixAddr, ln.Addr().String())
		}

	})
	t.Run("tcp", func(t *testing.T) {
		cfg := MeasurementConfig{
			ListenerType: ListenerTcp,
			ListenerAddr: "127.0.0.1:0",
		}
		ln, err := GetListener(cfg)
		if err != nil {
			t.Fatalf("Error setting up listener: %v", err)
		}
		defer ln.Close()
		if _, ok := ln.(*net.TCPListener); !ok {
			t.Errorf("Expected tcp listener, got %T", ln)
		}
		ip, port, _ := strings.Cut(ln.Addr().String(), ":")
		if ip != "127.0.0.1" {
			t.Errorf("Unexpected listener address.  Expected next open port on IP 127.0.0.1, got %v:%v", ip, port)
		}
	})
}

func TestRunMetricsServer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := &MeasurementConfig{
		ListenerType: ListenerTcp,
		ListenerAddr: ":8091",
	}
	if err := ConfigureListener(cfg); err != nil {
		t.Fatalf("Error configuring listener: %v", err)
	}

	errCh := make(chan error, 1)
	go func() { errCh <- RunMetricsServer(ctx, cfg) }()
	tempCh := make(chan float64, 30)
	volCh := make(chan float64, 30)

	go func() {
		for range 100 {

			resp, err := http.Get(fmt.Sprintf("http://%s/", cfg.ln.Addr().String()))
			if err != nil {
				t.Fatalf("Unexpected error calling metrics server: %v", err)
			}
			defer resp.Body.Close()

			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Errorf("Failed to read body: %v", err)
			}

			temp, err := parseMetricValues(t, io.NopCloser(bytes.NewReader(bodyBytes)), "room_temperature_celsius", -10, 40)
			if err != nil {
				t.Errorf("Error scraping temp metric: %v\nResponse: \n%s", err, string(bodyBytes))
			}
			for _, v := range temp {
				tempCh <- v
			}
			vol, err := parseMetricValues(t, io.NopCloser(bytes.NewReader(bodyBytes)), "room_volume_decibels", -10, 60)
			if err != nil {
				t.Errorf("Error scraping volume metric: %v\nResponse: \n%s", err, string(bodyBytes))
			}
			for _, v := range vol {
				volCh <- v
			}
		}

		close(tempCh)
		close(volCh)
		cancel()
	}()

	var wg sync.WaitGroup
	wg.Add(2)

	var lastTemp float64
	var unchangedTemp int
	go func() {
		for i := range tempCh {
			if lastTemp == 0 {
				lastTemp = i
				continue
			}
			if i == lastTemp {
				unchangedTemp++
			}
			lastTemp = i
		}
		wg.Done()
	}()

	var lastVol float64
	var unchangedVol int
	go func() {
		for i := range volCh {
			if lastVol == 0 {
				lastVol = i
				continue
			}
			if i == lastVol {
				unchangedVol++
			}
			lastVol = i
		}
		wg.Done()
	}()

	wg.Wait()

	if unchangedTemp > 10 {
		t.Errorf("%d temperature readings unchanged", unchangedTemp)
	}
	if unchangedVol > 10 {
		t.Errorf("%d volume readings unchanged. Last reading: %f", unchangedVol, lastVol)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("unexpected server error: %v", err)
	}
}

func TestHttpsRunMetricsServer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serverTLSConfig, clientTLSConfig := newTestmTLSConfig(t)

	cfg := &MeasurementConfig{
		ListenerType: ListenerTcp,
		ListenerAddr: ":8091",
		tlsConfig: serverTLSConfig,
	}
	if err := ConfigureListener(cfg); err != nil {
		t.Fatalf("Error configuring listener: %v", err)
	}

	go RunMetricsServer(ctx, cfg)
	t.Run("Correct mtls works", func(t *testing.T) {

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: clientTLSConfig,
		},
	}
	resp, err := client.Get(fmt.Sprintf("https://%s/", cfg.ln.Addr().String()))
			if err != nil {
				t.Fatalf("Unexpected error calling metrics server: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("Expected 200 OK, got %d", resp.StatusCode)
			}
			if resp.TLS == nil {
				t.Fatal("Expected TLS connection, but server resp.TLS is nil")
			}
		})
	t.Run("Missing client auth fails", func(t *testing.T) {
		unauthConfig := &tls.Config{
			Certificates: nil,
			RootCAs: clientTLSConfig.RootCAs,
		}
		unAuthClient := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: unauthConfig,
			},
		}
	_, err := unAuthClient.Get(fmt.Sprintf("https://%s/", cfg.ln.Addr().String()))
			if err == nil {
				t.Fatalf("Expected TLS handshake error when client certificate missing")
			}
	})

}

func newTestmTLSConfig(t *testing.T) (serverTLSConfig, clientTLSConfig *tls.Config) {
	t.Helper()

	now := time.Now()
	validFrom := now.Add(-1 * time.Minute)
	validTo := now.Add(5 * time.Minute)

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "Ephemeral Test CA"},
		NotBefore:    validFrom,
		NotAfter:     validTo,
		KeyUsage:     x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA: true,
	}

	rootCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}

	rootCert, err := x509.ParseCertificate(rootCertDER)
	if err != nil {
		t.Fatal(err)
	}

	issueLeaf := func(serial int64, cn string, usage x509.ExtKeyUsage, ips []net.IP, dns []string) tls.Certificate {
		leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatal(err)
		}

		leafTemplate := &x509.Certificate{
			SerialNumber: big.NewInt(serial),
			Subject:      pkix.Name{CommonName: cn},
			NotBefore:    validFrom,
			NotAfter:     validTo,
			ExtKeyUsage:  []x509.ExtKeyUsage{usage},
			IPAddresses:  ips,
			DNSNames:     dns,
		}

		leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, rootCert, &leafKey.PublicKey, caKey)
		if err != nil {
			t.Fatal(err)
		}

		keyBytes, err := x509.MarshalECPrivateKey(leafKey)
		if err != nil {
			t.Fatal(err)
		}

		certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leafDER})
		keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})

		tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
		if err != nil {
			t.Fatal(err)
		}
		return tlsCert
	}


	caPool := x509.NewCertPool()
	caPool.AddCert(rootCert)
	serverCert := issueLeaf(2, "metrics server", x509.ExtKeyUsageServerAuth, []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::")}, []string{"localhost"})
	clientCert := issueLeaf(3, "client-cert", x509.ExtKeyUsageClientAuth, nil, nil)

	serverTLSConfig = &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientCAs:    caPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS13,
	}
	clientTLSConfig = &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs: caPool,
	}
	return
}