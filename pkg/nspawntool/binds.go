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

package nspawntool

import (
	"fmt"

	"github.com/kinvolk/kube-spawn/pkg/config"
)

const bindro string = "--bind-ro="
const bindrw string = "--bind="

func optionsFromBindmountConfig(bm config.BindmountConfiguration) []string {
	var opts []string
	robinds := generateBinds(bindro, bm.ReadOnly)
	opts = append(opts, robinds...)
	rwbinds := generateBinds(bindrw, bm.ReadWrite)
	opts = append(opts, rwbinds...)
	return opts
}

func generateBinds(prefix string, binds []config.Pathmap) []string {
	var opts []string
	for _, pm := range binds {
		opts = append(opts, fmt.Sprintf("%s%s:%s", prefix, pm.Src, pm.Dst))
	}
	return opts
}
