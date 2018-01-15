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

// TODO(schu): use context and timeout throughout the package

package grpcdaemon

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"path"

	"google.golang.org/grpc"
	grpccodes "google.golang.org/grpc/codes"
	grpccredentials "google.golang.org/grpc/credentials"

	"github.com/kinvolk/kube-spawn/pkg/cache"
	"github.com/kinvolk/kube-spawn/pkg/cluster"
)

type GRPCDaemonConfig struct {
	CNIPluginDir     string
	ContainerRuntime string
	KubespawnDir     string
	Certificate      *tls.Certificate
}

type GRPCDaemon struct {
	config *GRPCDaemonConfig

	listener   net.Listener
	grpcServer *grpc.Server
}

func NewGRPCDaemon(config *GRPCDaemonConfig, listener net.Listener) (*GRPCDaemon, error) {
	var serverOptions []grpc.ServerOption
	if config.Certificate != nil {
		serverOptions = append(serverOptions, grpc.Creds(grpccredentials.NewServerTLSFromCert(config.Certificate)))
	}
	d := &GRPCDaemon{
		config:     config,
		listener:   listener,
		grpcServer: grpc.NewServer(serverOptions...),
	}

	RegisterKubeSpawnServer(d.grpcServer, d)

	return d, nil
}

func (d *GRPCDaemon) Serve() error {
	return d.grpcServer.Serve(d.listener)
}

func (d *GRPCDaemon) Shutdown(ctx context.Context) error {
	d.grpcServer.GracefulStop()
	return nil
}

func (d *GRPCDaemon) Create(ctx context.Context, clusterProps *ClusterProps) (*Result, error) {
	clusterName := clusterProps.Name
	clusterDir := path.Join(d.config.KubespawnDir, "clusters", clusterName)

	kluster, err := cluster.New(clusterDir, clusterName)
	if err != nil {
		log.Printf("Failed to create cluster object: %v", err)
		return nil, grpc.Errorf(grpccodes.Internal, "Failed to create cluster object: %s", err)
	}

	clusterSettings := &cluster.ClusterSettings{
		CNIPluginDir:      d.config.CNIPluginDir,
		ContainerRuntime:  d.config.ContainerRuntime,
		KubernetesVersion: clusterProps.KubernetesVersion,
		HyperkubeImage:    clusterProps.HyperkubeImage,
	}

	clusterCache, err := cache.New(path.Join(d.config.KubespawnDir, "cache"))
	if err != nil {
		log.Printf("Failed to create cache object: %v", err)
		return nil, grpc.Errorf(grpccodes.Internal, "Failed to create cache object: %s", err)
	}

	if err := kluster.Create(clusterSettings, clusterCache); err != nil {
		log.Printf("Failed to create cluster: %v", err)
		return nil, grpc.Errorf(grpccodes.Internal, "Failed to create cluster: %s", err)
	}

	return &Result{
		Success: true,
	}, nil
}

func (d *GRPCDaemon) Start(ctx context.Context, clusterProps *ClusterProps) (*Result, error) {
	clusterName := clusterProps.Name
	clusterDir := path.Join(d.config.KubespawnDir, "clusters", clusterName)

	kluster, err := cluster.New(clusterDir, clusterName)
	if err != nil {
		log.Printf("Failed to create cluster object: %v", err)
		return nil, grpc.Errorf(grpccodes.Internal, "Failed to create cluster object: %s", err)
	}

	if err := kluster.Start(int(clusterProps.NumberNodes), d.config.CNIPluginDir); err != nil {
		log.Printf("Failed to start cluster: %v", err)
		return nil, grpc.Errorf(grpccodes.Internal, "Failed to start cluster: %s", err)
	}

	return &Result{
		Success: true,
	}, nil
}

func (d *GRPCDaemon) List(ctx context.Context, _ *Empty) (*Result, error) {
	// TODO
	return &Result{}, nil
}

func (d *GRPCDaemon) Destroy(ctx context.Context, clusterProps *ClusterProps) (*Result, error) {
	clusterName := clusterProps.Name
	clusterDir := path.Join(d.config.KubespawnDir, "clusters", clusterName)

	kluster, err := cluster.New(clusterDir, clusterName)
	if err != nil {
		log.Printf("Failed to create cluster object: %v", err)
		return nil, grpc.Errorf(grpccodes.Internal, "Failed to create cluster object: %s", err)
	}

	if err := kluster.Destroy(); err != nil {
		log.Printf("Failed to destroy cluster: %v", err)
		return nil, grpc.Errorf(grpccodes.Internal, "Failed to destroy cluster: %s", err)
	}

	return &Result{
		Success: true,
	}, nil
}

func (d *GRPCDaemon) GetKubeconfig(ctx context.Context, clusterProps *ClusterProps) (*Result, error) {
	clusterName := clusterProps.Name
	clusterDir := path.Join(d.config.KubespawnDir, "clusters", clusterName)

	kluster, err := cluster.New(clusterDir, clusterName)
	if err != nil {
		log.Printf("Failed to create cluster object: %v", err)
		return nil, grpc.Errorf(grpccodes.Internal, "Failed to create cluster object: %s", err)
	}

	kubeconfig, err := kluster.AdminKubeconfig()
	if err != nil {
		return nil, grpc.Errorf(grpccodes.Internal, "Failed to read kubeconfig file: %s", err)
	}

	return &Result{
		Success:    true,
		Kubeconfig: kubeconfig,
	}, nil
}
