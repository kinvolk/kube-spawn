package bootstrap

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
)

const NspawnNetPath string = "/etc/cni/net.d/10-kube-spawn-net.conf"
const NspawnNetConf string = `
{
    "cniVersion": "0.2.0",
    "name": "kube-spawn-net",
    "type": "bridge",
    "bridge": "cni0",
    "isGateway": true,
    "ipMasq": true,
    "ipam": {
        "type": "host-local",
        "subnet": "10.22.0.0/16",
        "routes": [
            { "dst": "0.0.0.0/0" }
        ]
    }
}`
const LoopbackNetPath string = "/etc/cni/net.d/10-loopback.conf"
const LoopbackNetConf string = `
{
    "cniVersion": "0.2.0",
    "type": "loopback"
}`

func writeNetConf(fpath, content string) error {
	if _, err := os.Stat(fpath); os.IsExist(err) {
		return nil
	}
	dir, _ := path.Split(fpath)
	os.MkdirAll(dir, os.ModePerm)
	if err := ioutil.WriteFile(fpath, []byte(content), os.ModePerm); err != nil {
		return fmt.Errorf("error writing %s: %s", fpath, err)
	}
	return nil
}

func WriteNetConf() error {
	if err := writeNetConf(NspawnNetPath, NspawnNetConf); err != nil {
		return err
	}
	if err := writeNetConf(LoopbackNetPath, LoopbackNetConf); err != nil {
		return err
	}
	return nil
}
