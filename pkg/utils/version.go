package utils

import (
	"fmt"

	"github.com/Masterminds/semver"
)

func VersionConstraintSatisfied(version, constraint string) (bool, error) {
	v, err := semver.NewVersion(version)
	if err != nil {
		return false, fmt.Errorf("cannot parse %q as semver version: %v", version, err)
	}
	c, err := semver.NewConstraint(constraint)
	if err != nil {
		return false, fmt.Errorf("cannot parse %q as version constraint: %v", constraint, err)
	}
	return c.Check(v), nil
}
