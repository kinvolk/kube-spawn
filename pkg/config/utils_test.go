// Copyright 2017 Kinvolk GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"testing"
)

func TestIsVerConstraintOrDev(t *testing.T) {
	var cfg = &ClusterConfiguration{}

	for i, tt := range []struct {
		devCluster        bool
		kubernetesVersion string
		constraint        string
		expectedResult    bool
	}{
		{
			false,
			"v1.7.5",
			">=v1.7.5",
			true,
		},
		{
			false,
			"v1.7.5",
			"<v1.7.5",
			false,
		},
		{
			true,
			"v1.8.0",
			">=v1.7.5",
			true,
		},
		{
			false,
			"latest",
			">=v1.7.5",
			true,
		},
	} {
		cfg.DevCluster = tt.devCluster
		cfg.KubernetesVersion = tt.kubernetesVersion
		outResult := cfg.IsVerConstraintOrDev(tt.constraint)

		if tt.expectedResult != outResult {
			t.Errorf("IsVerConstraintOrDev %d expected %v got %v", i, tt.expectedResult, outResult)
		}
	}
}
