/*
Copyright 2017 Kinvolk GmbH

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

package distribution

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/docker/docker/api/types"
	containertypes "github.com/docker/docker/api/types/container"
	networktypes "github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

func dockerProgress(rc io.ReadCloser) error {
	var status string

	bufReader := bufio.NewReader(rc)
	for {
		line, _, err := bufReader.ReadLine()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}

		var jsonLine map[string]interface{}
		if err := json.Unmarshal(line, &jsonLine); err != nil {
			return err
		}

		if errMsg, ok := jsonLine["error"]; ok {
			if errMsgStr, ok := errMsg.(string); ok {
				return fmt.Errorf(errMsgStr)
			}
		}

		if s, ok := jsonLine["status"]; ok {
			if s.(string) != status {
				status = s.(string)
				log.Println(status)
			}
		}
	}
}

func StartRegistry() error {
	ctx := context.Background()

	cli, err := client.NewEnvClient()
	if err != nil {
		return err
	}

	log.Println("Pulling registry image")
	readerCloser, err := cli.ImagePull(ctx, "docker.io/library/registry:2", types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer readerCloser.Close()

	if err := dockerProgress(readerCloser); err != nil {
		return err
	}

	if _, err := cli.ContainerCreate(context.Background(), &containertypes.Config{
		Image: "registry:2",
	}, &containertypes.HostConfig{
		PortBindings: nat.PortMap{"5000/tcp": []nat.PortBinding{nat.PortBinding{
			HostIP:   "0.0.0.0",
			HostPort: "5000",
		}}},
	}, &networktypes.NetworkingConfig{}, "registry"); err != nil && !strings.Contains(err.Error(), "Conflict") {
		return err
	}
	return cli.ContainerStart(ctx, "registry", types.ContainerStartOptions{})
}

func PushImage() error {
	cli, err := client.NewEnvClient()
	if err != nil {
		return err
	}

	if err := cli.ImageTag(
		context.Background(),
		"gcr.io/google_containers/hyperkube-amd64",
		"10.22.0.1:5000/hyperkube-amd64",
	); err != nil {
		return err
	}

	log.Println("Pushing hyperkube image to local registry")
	readerCloser, err := cli.ImagePush(context.Background(), "10.22.0.1:5000/hyperkube-amd64", types.ImagePushOptions{
		All: true,
		// RegistryAuth header cannot be empty, even if no authentication is used at all...
		RegistryAuth: "123",
	})
	if err != nil {
		return err
	}
	defer readerCloser.Close()

	if err := dockerProgress(readerCloser); err != nil {
		return err
	}
	return nil
}
