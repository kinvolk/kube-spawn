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
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/cobra"

	"github.com/kinvolk/kube-spawn/pkg/grpcshim"
)

var (
	grpcDaemonCmd = &cobra.Command{
		Use:    "grpc-shim-daemon",
		Short:  "Run the kube-spawn grpc shim daemon",
		Hidden: true,
		Run:    runGRPCDaemon,
	}
	addr = "0.0.0.0:50051"
)

func init() {
	kubespawnCmd.AddCommand(grpcDaemonCmd)
}

func runGRPCDaemon(cmd *cobra.Command, args []string) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	config := grpcshim.GRPCShimDaemonConfig{
		KubeSpawnDir: "/var/lib/kube-spawn",
	}

	daemon, err := grpcshim.NewGRPCShimDaemon(&config, listener)
	if err != nil {
		log.Fatal(err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, os.Kill)

	go func() {
		log.Printf("listening on %s\n", addr)
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
