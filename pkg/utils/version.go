package utils

import "github.com/Masterminds/semver"

func CheckVersionConstraint(version, constraint string) bool {
	v, err := semver.NewVersion(version)
	if err != nil {
		return false
	}

	c, _ := semver.NewConstraint(constraint)
	return c.Check(v)
}
