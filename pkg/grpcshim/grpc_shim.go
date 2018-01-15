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

package grpcshim

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"path"

	"google.golang.org/grpc"
	grpccodes "google.golang.org/grpc/codes"

	"github.com/kinvolk/kube-spawn/pkg/utils"
)

type GRPCShimDaemonConfig struct {
	KubeSpawnDir string
}

type GRPCShimDaemon struct {
	config *GRPCShimDaemonConfig

	listener   net.Listener
	grpcServer *grpc.Server
}

func NewGRPCShimDaemon(config *GRPCShimDaemonConfig, listener net.Listener) (*GRPCShimDaemon, error) {
	d := &GRPCShimDaemon{
		config:     config,
		listener:   listener,
		grpcServer: grpc.NewServer(),
	}

	RegisterKubeSpawnServer(d.grpcServer, d)

	return d, nil
}

func (d *GRPCShimDaemon) Serve() error {
	return d.grpcServer.Serve(d.listener)
}

func (d *GRPCShimDaemon) Shutdown(ctx context.Context) error {
	// TODO(schu): use context and timeout
	d.grpcServer.GracefulStop()
	return nil
}

func (d *GRPCShimDaemon) Create(ctx context.Context, clusterProps *ClusterProperties) (*Result, error) {
	// TODO(schu): use context and timeout

	var (
		stdoutBuf bytes.Buffer
		stderrBuf bytes.Buffer
	)

	kubeSpawnCmd := utils.Command("kube-spawn", "create", "--cluster-name", clusterProps.Name, "--nodes", fmt.Sprint(clusterProps.NumberNodes))
	kubeSpawnCmd.Stdout = &stdoutBuf
	kubeSpawnCmd.Stderr = &stderrBuf

	result := &Result{}

	if err := kubeSpawnCmd.Run(); err == nil {
		result.Success = true
	} else {
		log.Printf("grpc error: %v\n", err)
	}

	result.Stdout = stdoutBuf.String()
	result.Stderr = stderrBuf.String()

	log.Printf("result: %+v\n", result)
	return result, nil
}

func (d *GRPCShimDaemon) Start(ctx context.Context, cluster *Cluster) (*Result, error) {
	var (
		stdoutBuf bytes.Buffer
		stderrBuf bytes.Buffer
	)

	kubeSpawnCmd := utils.Command("kube-spawn", "start", "--cluster-name", cluster.Name)
	kubeSpawnCmd.Stdout = &stdoutBuf
	kubeSpawnCmd.Stderr = &stderrBuf

	result := &Result{}

	if err := kubeSpawnCmd.Run(); err == nil {
		result.Success = true
	} else {
		log.Printf("grpc error: %v\n", err)
	}

	result.Stdout = stdoutBuf.String()
	result.Stderr = stderrBuf.String()

	log.Printf("result: %+v\n", result)
	return result, nil
}

func (d *GRPCShimDaemon) List(ctx context.Context, _ *Empty) (*Result, error) {
	var (
		stdoutBuf bytes.Buffer
		stderrBuf bytes.Buffer
	)

	kubeSpawnCmd := utils.Command("kube-spawn", "list")
	kubeSpawnCmd.Stdout = &stdoutBuf
	kubeSpawnCmd.Stderr = &stderrBuf

	result := &Result{}

	if err := kubeSpawnCmd.Run(); err == nil {
		result.Success = true
	} else {
		log.Printf("grpc error: %v\n", err)
	}

	result.Stdout = stdoutBuf.String()
	result.Stderr = stderrBuf.String()

	log.Printf("result: %+v\n", result)
	return result, nil
}

func (d *GRPCShimDaemon) Destroy(ctx context.Context, cluster *Cluster) (*Result, error) {
	var (
		stdoutBuf bytes.Buffer
		stderrBuf bytes.Buffer
	)

	kubeSpawnCmd := utils.Command("kube-spawn", "destroy", "--cluster-name", cluster.Name)
	kubeSpawnCmd.Stdout = &stdoutBuf
	kubeSpawnCmd.Stderr = &stderrBuf

	result := &Result{}

	if err := kubeSpawnCmd.Run(); err == nil {
		result.Success = true
	} else {
		log.Printf("grpc error: %v\n", err)
	}

	result.Stdout = stdoutBuf.String()
	result.Stderr = stderrBuf.String()

	log.Printf("result: %+v\n", result)
	return result, nil
}

func (d *GRPCShimDaemon) GetKubeconfig(ctx context.Context, cluster *Cluster) (*Result, error) {
	result := &Result{}

	kubeconfigPath := path.Join(d.config.KubeSpawnDir, cluster.Name, "kubeconfig")

	kubeconfigBytes, err := ioutil.ReadFile(kubeconfigPath)
	if err != nil {
		return nil, grpc.Errorf(grpccodes.Internal, "Failed to read kubeconfig file: %s", err)
	}

	result.Success = true
	result.Kubeconfig = string(kubeconfigBytes)

	log.Printf("result: %+v\n", result)
	return result, nil
}
