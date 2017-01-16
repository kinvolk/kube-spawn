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

package cnispawn

import (
	"os"
	"os/exec"
	"runtime"
	"syscall"
)

func RunContainer(path string, background bool) error {
	runtime.LockOSThread()

	cniNetns, err := NewCniNetns()
	if err != nil {
		return err
	}

	if err := cniNetns.Set(); err != nil {
		return err
	}
	defer cniNetns.Close()

	systemdNspawnPath := os.Getenv("SYSTEMD_NSPAWN_PATH")

	if systemdNspawnPath == "" {
		systemdNspawnPath, err = exec.LookPath("systemd-nspawn")
		if err != nil {
			return err
		}
	}

	// TODO(nhlfr): Don't hardcode kubeadm-systemd specific options,
	// expose all the options of systemd-nspawn instead.
	args := []string{
		systemdNspawnPath,
		"--capability=cap_audit_control,cap_audit_read,cap_audit_write,cap_audit_control,cap_block_suspend,cap_chown,cap_dac_override,cap_dac_read_search,cap_fowner,cap_fsetid,cap_ipc_lock,cap_ipc_owner,cap_kill,cap_lease,cap_linux_immutable,cap_mac_admin,cap_mac_override,cap_mknod,cap_net_admin,cap_net_bind_service,cap_net_broadcast,cap_net_raw,cap_setgid,cap_setfcap,cap_setpcap,cap_setuid,cap_sys_admin,cap_sys_boot,cap_sys_chroot,cap_sys_module,cap_sys_nice,cap_sys_pacct,cap_sys_ptrace,cap_sys_rawio,cap_sys_resource,cap_sys_time,cap_sys_tty_config,cap_syslog,cap_wake_alarm",
		// "--bind=/proc/sys/net/bridge",
		// "--capability=cap_audit_write,cap_audit_control,cap_sys_admin,cap_sys_chroot,cap_net_admin,cap_setfcap,cap_syslog",
		"--bind=/sys/fs/cgroup",
		"--bind-ro=/boot",
		"--bind-ro=/lib/modules",
		"-bD",
		path,
	}

	env := os.Environ()
	// env = append(env, "SYSTEMD_NSPAWN_MOUNT_RW=true")
	env = append(env, "SYSTEMD_NSPAWN_API_VFS_WRITABLE=true")

	if background {
		_, err := syscall.ForkExec(systemdNspawnPath, args, &syscall.ProcAttr{
			Dir:   "",
			Env:   env,
			Files: []uintptr{},
			Sys:   &syscall.SysProcAttr{},
		})
		return err
	}
	return syscall.Exec(systemdNspawnPath, args, env)
}
