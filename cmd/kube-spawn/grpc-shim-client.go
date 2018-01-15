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
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/kinvolk/kube-spawn/pkg/grpcshim"
)

var (
	grpcClientCmd = &cobra.Command{
		Use:    "grpc-shim-client",
		Short:  "Run the kube-spawn grpc shim client",
		Hidden: true,
		Run:    runGRPCClient,
	}
)

func init() {
	kubespawnCmd.AddCommand(grpcClientCmd)
}

func runGRPCClient(cmd *cobra.Command, args []string) {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	kubeSpawnClient := grpcshim.NewKubeSpawnClient(conn)

	clusterName := viper.GetString("cluster-name")

	var result *grpcshim.Result

	action := args[0]
	switch action {
	case "list":
		result, err = kubeSpawnClient.List(context.Background(), &grpcshim.Empty{})
		if err != nil {
			log.Fatalf("grpc request failed: %v\n", err)
		} else {
			log.Printf("OK")
			log.Println(result.Stdout)
			log.Println(result.Stderr)
		}
	case "get-kubeconfig":
		result, err = kubeSpawnClient.GetKubeconfig(context.Background(), &grpcshim.Cluster{Name: clusterName})
		if err != nil {
			log.Fatalf("grpc request failed: %v\n", err)
		} else {
			fmt.Println(result.Kubeconfig)
		}
	case "create":
		result, err = kubeSpawnClient.Create(context.Background(), &grpcshim.ClusterProperties{Name: clusterName, NumberNodes: 2})
		if err != nil {
			log.Fatalf("grpc request failed: %v\n", err)
		} else {
			log.Printf("OK")
			log.Println(result.Stdout)
			log.Println(result.Stderr)
		}
	case "start":
		result, err = kubeSpawnClient.Start(context.Background(), &grpcshim.Cluster{Name: clusterName})
		if err != nil {
			log.Fatalf("grpc request failed: %v\n", err)
		} else {
			log.Printf("OK")
			log.Println(result.Stdout)
			log.Println(result.Stderr)
		}
	case "destroy":
		result, err = kubeSpawnClient.Destroy(context.Background(), &grpcshim.Cluster{Name: clusterName})
		if err != nil {
			log.Fatalf("grpc request failed: %v\n", err)
		} else {
			log.Printf("OK")
			log.Println(result.Stdout)
			log.Println(result.Stderr)
		}
	default:
		log.Fatalf("unknown action %q - aborting", action)
	}
}
