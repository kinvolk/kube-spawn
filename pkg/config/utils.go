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

package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	toml "github.com/pelletier/go-toml"
	"github.com/pkg/errors"
	"github.com/spf13/viper"

	"github.com/kinvolk/kube-spawn/pkg/utils/fs"
)

const (
	Filename = "kspawn.toml"

	machineNameTemplate = "kubespawn%d"
	envFilename         = "current-env"
)

func MachineName(no int) string {
	return fmt.Sprintf(machineNameTemplate, no)
}

// TODO: this is not enough.
// need to always check machined or we might lose track in case of errors
func RunningMachines(cfg *ClusterConfiguration) int {
	var n int
	for _, m := range cfg.Machines {
		if m.Running {
			n++
		}
	}
	return n
}

func LoadConfig() (*ClusterConfiguration, error) {
	env, err := GetCurrentEnv()
	if err != nil {
		return nil, errors.Wrap(err, "unable to read current env")
	}

	cfgFile := path.Join(viper.GetString("dir"), env, Filename)
	viper.SetConfigFile(cfgFile)

	var cfg = &ClusterConfiguration{}
	err = viper.ReadInConfig()
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, errors.Wrap(err, "unable to decode viper config")
	}
	return cfg, err
}

func IsNotFound(err error) bool {
	switch err.(type) {
	case viper.ConfigFileNotFoundError:
		return true
	default:
		return os.IsNotExist(err)
	}
}

func WriteConfigToFile(cfg *ClusterConfiguration) error {
	cfgFilepath := path.Join(cfg.KubeSpawnDir, cfg.Name, Filename)
	raw, err := toml.Marshal(*cfg)
	if err != nil {
		return errors.Wrap(err, "unable to encode cluster config")
	}
	return fs.CreateBytes(cfgFilepath, raw)
}

func SetCurrentEnv(name string) error {
	envFile := path.Join(viper.GetString("dir"), envFilename)
	return fs.CreateBytes(envFile, []byte(name))
}

func GetCurrentEnv() (string, error) {
	envFile := path.Join(viper.GetString("dir"), envFilename)
	b, err := ioutil.ReadFile(envFile)
	if IsNotFound(err) {
		cName := viper.GetString("cluster-name")
		err = SetCurrentEnv(cName)
		if err == nil {
			return cName, nil
		}
	}
	return string(b), err
}
