package version

import (
	"fmt"
	"regexp"
	"strconv"
)

// Semver represents a parsed semantic version.
type Semver struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string
}

var semverRe = regexp.MustCompile(`^v([0-9]+)\.([0-9]+)\.([0-9]+)([-].+)?$`)

// Parse parses a version string like "v1.2.3" or "v1.2.3-rc.1".
func Parse(s string) (Semver, error) {
	m := semverRe.FindStringSubmatch(s)
	if m == nil {
		return Semver{}, fmt.Errorf("invalid semver: %q", s)
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	patch, _ := strconv.Atoi(m[3])
	return Semver{Major: major, Minor: minor, Patch: patch, Prerelease: m[4]}, nil
}

// BumpPatch returns a new version with the patch number incremented and prerelease cleared.
func (v Semver) BumpPatch() Semver {
	return Semver{Major: v.Major, Minor: v.Minor, Patch: v.Patch + 1}
}

// String returns the version as "vMAJOR.MINOR.PATCH[prerelease]".
func (v Semver) String() string {
	return fmt.Sprintf("v%d.%d.%d%s", v.Major, v.Minor, v.Patch, v.Prerelease)
}

// GreaterThan returns true if v is strictly greater than other.
func (v Semver) GreaterThan(other Semver) bool {
	if v.Major != other.Major {
		return v.Major > other.Major
	}
	if v.Minor != other.Minor {
		return v.Minor > other.Minor
	}
	return v.Patch > other.Patch
}
