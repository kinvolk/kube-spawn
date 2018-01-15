/*
Copyright 2018 Kinvolk GmbH

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
	"log"
	"math/big"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/kinvolk/kube-spawn/pkg/grpcdaemon"
)

var (
	grpcDaemonCmd = &cobra.Command{
		Use:    "grpc-daemon",
		Short:  "Run the kube-spawn grpc daemon",
		Hidden: true,
		Run:    runGRPCDaemon,
	}
)

func init() {
	kubespawnCmd.AddCommand(grpcDaemonCmd)

	grpcDaemonCmd.Flags().String("addr", "127.0.0.1:50051", "Address to listen on")
	grpcDaemonCmd.Flags().String("cni-plugin-dir", "/opt/cni/bin", "Path to directory with CNI plugins")
	grpcDaemonCmd.Flags().Bool("tls-self-signed", false, "Generate and use a self-signed certificate")

	viper.BindPFlags(grpcDaemonCmd.Flags())
}

func runGRPCDaemon(cmd *cobra.Command, args []string) {
	addr := viper.GetString("addr")
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	var daemonCert *tls.Certificate
	if viper.GetBool("tls-self-signed") {
		// Derived from https://golang.org/src/crypto/tls/generate_cert.go

		privateKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
		if err != nil {
			log.Fatalf("Failed to generate private key: %v", err)
		}

		serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
		serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
		if err != nil {
			log.Fatalf("Failed to generate serial number: %v", err)
		}

		notBefore := time.Now()
		notAfter := notBefore.Add(365 * 24 * time.Hour)

		template := x509.Certificate{
			SerialNumber: serialNumber,
			Subject: pkix.Name{
				Organization: []string{"kube-spawn insecure self-signed certificate"},
			},
			NotBefore: notBefore,
			NotAfter:  notAfter,

			KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
			ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			BasicConstraintsValid: true,
		}

		template.IPAddresses = append(template.IPAddresses, net.ParseIP("127.0.0.1"))

		derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
		if err != nil {
			log.Fatalf("Failed to create certificate: %s", err)
		}

		var certPEM bytes.Buffer
		if err := pem.Encode(&certPEM, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
			log.Fatalf("Failed to PEM encode certificate: %v", err)
		}

		var keyPEM bytes.Buffer
		privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
		if err != nil {
			log.Fatalf("Failed to marshal ECDSA private key: %v", err)
		}
		if err := pem.Encode(&keyPEM, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privateKeyBytes}); err != nil {
			log.Fatalf("Failed to PEM encode certificate: %v", err)
		}

		cert, err := tls.X509KeyPair(certPEM.Bytes(), keyPEM.Bytes())
		if err != nil {
			log.Fatalf("Failed to parse PEM data: %v", err)
		}
		daemonCert = &cert
	}

	config := grpcdaemon.GRPCDaemonConfig{
		CNIPluginDir:     viper.GetString("cni-plugin-dir"),
		ContainerRuntime: "docker",
		KubespawnDir:     viper.GetString("dir"),
		Certificate:      daemonCert,
	}

	daemon, err := grpcdaemon.NewGRPCDaemon(&config, listener)
	if err != nil {
		log.Fatal(err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, os.Kill)

	go func() {
		log.Printf("Listening on %s\n", addr)
		if err := daemon.Serve(); err != nil {
			log.Fatalf("grpc server error: %v\n", err)
		}
	}()

	<-sigChan

	log.Printf("Shutting down ...\n")

	ctx, cancelCtx := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelCtx()
	if err := daemon.Shutdown(ctx); err != nil {
		log.Fatalf("Clean shutdown failed: %v\n", err)
	}
}
