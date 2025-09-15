package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type VersionConstraint struct {
	Operator   string
	Version    Version
	Constraint string
}

type Version struct {
	Major int
	Minor int
	Patch int
}

func ParseVersion(versionStr string) (Version, error) {
	versionStr = strings.TrimPrefix(versionStr, "v")

	parts := strings.Split(versionStr, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("invalid version format: %s", versionStr)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Version{}, fmt.Errorf("invalid major version: %s", parts[0])
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return Version{}, fmt.Errorf("invalid minor version: %s", parts[1])
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return Version{}, fmt.Errorf("invalid patch version: %s", parts[2])
	}

	return Version{Major: major, Minor: minor, Patch: patch}, nil
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func (v Version) Compare(version Version) int {
	if v.Major != version.Major {
		if v.Major < version.Major {
			return -1
		}
		return 1
	}

	if v.Minor != version.Minor {
		if v.Minor < version.Minor {
			return -1
		}
		return 1
	}

	if v.Patch != version.Patch {
		if v.Patch < version.Patch {
			return -1
		}
		return 1
	}

	return 0
}

func ParseVersionConstraint(constraint string) (VersionConstraint, error) {
	constraint = strings.TrimSpace(constraint)

	re := regexp.MustCompile(`^(>=|>|<=|<|=)?(\d+\.\d+\.\d+)$`)
	matches := re.FindStringSubmatch(constraint)

	if matches == nil {
		return VersionConstraint{}, fmt.Errorf("invalid project cli version constraint format: %s", constraint)
	}

	operator := matches[1]
	versionStr := matches[2]

	if operator == "" {
		operator = "="
	}

	version, err := ParseVersion(versionStr)
	if err != nil {
		return VersionConstraint{}, err
	}

	return VersionConstraint{
		Operator:   operator,
		Version:    version,
		Constraint: constraint,
	}, nil
}

func (c VersionConstraint) Matches(version Version) bool {
	switch c.Operator {
	case "=":
		return version.Compare(c.Version) == 0
	case ">":
		return version.Compare(c.Version) > 0
	case ">=":
		return version.Compare(c.Version) >= 0
	case "<":
		return version.Compare(c.Version) < 0
	case "<=":
		return version.Compare(c.Version) <= 0
	default:
		return false
	}
}

func FindMatchingVersion(versionConstraint string, availableVersions []string) (string, error) {
	parsedConstraint, err := ParseVersionConstraint(versionConstraint)
	if err != nil {
		return "", err
	}

	var matchingVersions []Version
	versionMap := make(map[string]Version)

	for _, vStr := range availableVersions {
		v, err := ParseVersion(vStr)
		if err != nil {
			continue
		}
		versionMap[vStr] = v

		if parsedConstraint.Matches(v) {
			matchingVersions = append(matchingVersions, v)
		}
	}

	if len(matchingVersions) == 0 {
		return "", fmt.Errorf("please install the version %s compatible with the project", versionConstraint)
	}

	highest := matchingVersions[0]
	for _, v := range matchingVersions[1:] {
		if v.Compare(highest) > 0 {
			highest = v
		}
	}

	for vStr, v := range versionMap {
		if v.Compare(highest) == 0 {
			return vStr, nil
		}
	}

	return highest.String(), nil
}

func IsVersionConstraint(s string) bool {
	constraintPrefixes := []string{">=", ">", "<=", "<", "="}
	for _, prefix := range constraintPrefixes {
		if strings.HasPrefix(strings.TrimSpace(s), prefix) {
			return true
		}
	}

	return false
}

func ValidateVersionAgainstConstraint(version, projectRequiredVersion string) error {
	versionConstraint, err := ParseVersionConstraint(projectRequiredVersion)
	if err != nil {
		return fmt.Errorf("invalid cli version '%s' provided in .jfrog-version file", projectRequiredVersion)
	}

	targetVersion, err := ParseVersion(version)
	if err != nil {
		return fmt.Errorf("unable to parse version '%s' ", version)
	}

	if !versionConstraint.Matches(targetVersion) {
		return fmt.Errorf("please use a version %s", projectRequiredVersion)
	}

	return nil
}

func GetInstalledVersions() ([]string, error) {
	jfVersions, err := os.ReadDir(JfvmVersions)
	if err != nil {
		return nil, err
	}

	var versions []string
	for _, entry := range jfVersions {
		if entry.IsDir() {
			binPath := filepath.Join(JfvmVersions, entry.Name(), BinaryName)
			if _, err := os.Stat(binPath); err == nil {
				versions = append(versions, entry.Name())
			}
		}
	}

	sort.Slice(versions, func(i, j int) bool {
		v1, err1 := ParseVersion(versions[i])
		v2, err2 := ParseVersion(versions[j])

		if err1 != nil || err2 != nil {
			return versions[i] < versions[j]
		}

		return v1.Compare(v2) < 0
	})

	return versions, nil
}
