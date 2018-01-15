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
	"crypto/tls"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/kinvolk/kube-spawn/pkg/grpcdaemon"
)

var (
	grpcClientCmd = &cobra.Command{
		Use:    "grpc-client",
		Short:  "Run the kube-spawn grpc client",
		Hidden: true,
		Run:    runGRPCClient,
	}
)

func init() {
	kubespawnCmd.AddCommand(grpcClientCmd)

	grpcClientCmd.Flags().String("server", "127.0.0.1:50051", "Address of kube-spawn grpc daemon")
	grpcClientCmd.Flags().Bool("disable-tls", false, "Disable grpc transport security")
	grpcClientCmd.Flags().Bool("tls-skip-verify", false, "Disable TLS certificate verification")

	viper.BindPFlags(grpcClientCmd.Flags())
}

func runGRPCClient(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		log.Fatal("No action given")
	}

	server := viper.GetString("server")

	var dialOptions []grpc.DialOption
	if viper.GetBool("disable-tls") {
		dialOptions = append(dialOptions, grpc.WithInsecure())
	} else {
		var tlsConfig *tls.Config
		if viper.GetBool("tls-skip-verify") {
			tlsConfig = &tls.Config{InsecureSkipVerify: true}
		}
		creds := credentials.NewTLS(tlsConfig)
		dialOptions = append(dialOptions, grpc.WithTransportCredentials(creds))
	}

	log.Printf("Connecting to %s\n", server)

	conn, err := grpc.Dial(server, dialOptions...)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	kubeSpawnClient := grpcdaemon.NewKubeSpawnClient(conn)

	clusterName := viper.GetString("cluster-name")

	var result *grpcdaemon.Result

	action := args[0]
	switch action {
	case "list":
		result, err = kubeSpawnClient.List(context.Background(), &grpcdaemon.Empty{})
	case "get-kubeconfig":
		result, err = kubeSpawnClient.GetKubeconfig(context.Background(), &grpcdaemon.ClusterProps{Name: clusterName})
	case "create":
		result, err = kubeSpawnClient.Create(context.Background(), &grpcdaemon.ClusterProps{Name: clusterName, KubernetesVersion: "v1.9.2"})
	case "start":
		result, err = kubeSpawnClient.Start(context.Background(), &grpcdaemon.ClusterProps{Name: clusterName, NumberNodes: 2})
	case "destroy":
		result, err = kubeSpawnClient.Destroy(context.Background(), &grpcdaemon.ClusterProps{Name: clusterName})
	default:
		log.Fatalf("unknown action %q - aborting", action)
	}
	if err != nil {
		log.Fatalf("grpc request failed: %v\n", err)
	} else {
		log.Print("OK")
		log.Printf("%+v", result)
	}
}
