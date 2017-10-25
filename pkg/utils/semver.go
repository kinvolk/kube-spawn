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

package utils

import (
	"fmt"

	"github.com/Masterminds/semver"
)

// IsSemVerOrNewer returns true if a semantic version string keyVersion is newer
// than or same as conVersion. For example, if keyVersion is 1.8.1 and
// conVersion is 1.7.7, it returns true.
func IsSemVerOrNewer(keyVersion, conVersion string) bool {
	v, err := semver.NewVersion(keyVersion)
	if err != nil {
		return false
	}

	c, err := semver.NewConstraint(fmt.Sprintf(">=%s", conVersion))
	if err != nil {
		return false
	}

	return c.Check(v)
}

// IsSemVer returns true if a semantic version string keyVersion is the same as
// conVersion. For example, if both keyVersion and conVersion are 1.8.1,
// it returns true.
func IsSemVer(keyVersion, conVersion string) bool {
	v, err := semver.NewVersion(keyVersion)
	if err != nil {
		return false
	}

	c, err := semver.NewConstraint(fmt.Sprintf("=%s", conVersion))
	if err != nil {
		return false
	}

	return c.Check(v)
}
