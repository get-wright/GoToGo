// server/main.go
package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"GoToGo/server/api"
	"GoToGo/server/cert"
	"GoToGo/server/session"
)

var (
	certDir = flag.String("cert-dir", "certs", "Directory for certificates")
	port    = flag.String("port", "8443", "Server port")
	logFile = flag.String("log", "logs/server.log", "Log file path")
)

func main() {
	flag.Parse()

	// Setup logging
	os.MkdirAll("logs", 0755)
	f, err := os.OpenFile(*logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Error opening log file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	// Initialize certificate manager
	certManager, err := cert.NewCertManager(*certDir)
	if err != nil {
		log.Fatalf("Failed to initialize certificate manager: %v", err)
	}

	// Initialize session manager
	sessionManager := session.NewSessionManager(24 * time.Hour)

	// Initialize API handlers
	apiHandler := api.NewHandler(certManager, sessionManager)

	// Configure TLS
	tlsConfig := &tls.Config{
		ClientAuth:       tls.RequireAndVerifyClientCert,
		ClientCAs:        certManager.GetCertPool(),
		MinVersion:       tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{tls.CurveP256, tls.CurveP384, tls.CurveP521},
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		},
	}

	// Create HTTPS server
	server := &http.Server{
		Addr:         ":" + *port,
		Handler:      apiHandler.Router(),
		TLSConfig:    tlsConfig,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server
	certFile := filepath.Join(*certDir, "server-cert.pem")
	keyFile := filepath.Join(*certDir, "server-key.pem")
	log.Printf("Starting server on port %s...", *port)
	log.Fatal(server.ListenAndServeTLS(certFile, keyFile))
}
