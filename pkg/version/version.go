package version

import "github.com/Masterminds/semver/v3"

var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
	Arch    = "unknown"
)

type ReleaseVersion struct{ current *semver.Version }

func (v ReleaseVersion) String() string {
	return v.current.String()
}

func (v ReleaseVersion) IsTargetVersionAhead(target *semver.Version) bool {
	return v.current.LessThan(target)
}

func (v ReleaseVersion) IsTargetVersionStringAhead(target string) (bool, error) {
	parsedTarget, parseTargetErr := semver.NewVersion(target)
	if parseTargetErr != nil {
		return false, parseTargetErr
	}

	return v.IsTargetVersionAhead(parsedTarget), nil
}

func (v ReleaseVersion) IsTargetVersionBehind(target *semver.Version) bool {
	return v.current.GreaterThan(target)
}

func (v ReleaseVersion) IsTargetVersionStringBehind(target string) (bool, error) {
	parsedTarget, parseTargetErr := semver.NewVersion(target)
	if parseTargetErr != nil {
		return false, parseTargetErr
	}

	return v.IsTargetVersionBehind(parsedTarget), nil
}

func (v ReleaseVersion) IsTargetVersionCurrent(target *semver.Version) bool {
	return v.current.Equal(target)
}

func (v ReleaseVersion) IsTargetVersionStringCurrent(target string) (bool, error) {
	parsedTarget, parseTargetErr := semver.NewVersion(target)
	if parseTargetErr != nil {
		return false, parseTargetErr
	}

	return v.IsTargetVersionCurrent(parsedTarget), nil
}

func BuildReleaseVersion(version string) (ReleaseVersion, error) {
	parsedVersion, parseErr := semver.NewVersion(version)
	if parseErr != nil {
		return ReleaseVersion{}, parseErr
	}

	return ReleaseVersion{parsedVersion}, nil
}
