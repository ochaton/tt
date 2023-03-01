package install_ee

import (
	"sort"

	"github.com/tarantool/tt/cli/util"
	"github.com/tarantool/tt/cli/version"
)

// EEVersion is a structure that contains specific information about SDK bundle.
type EEVersion struct {
	VersionInfo version.Version
	Prefix      string
}

// SortEEVersions sorts versions from oldest to newest.
func SortEEVersions(versions []EEVersion) {
	sort.SliceStable(versions, func(i, j int) bool {
		verLeft := versions[i]
		verRight := versions[j]

		left := []uint64{verLeft.VersionInfo.Major, verLeft.VersionInfo.Minor,
			verLeft.VersionInfo.Patch, uint64(verLeft.VersionInfo.Release.Type),
			verLeft.VersionInfo.Release.Num, verLeft.VersionInfo.Additional, verLeft.VersionInfo.Revision}
		right := []uint64{verRight.VersionInfo.Major, verRight.VersionInfo.Minor,
			verRight.VersionInfo.Patch, uint64(verRight.VersionInfo.Release.Type),
			verRight.VersionInfo.Release.Num, verRight.VersionInfo.Additional, verRight.VersionInfo.Revision}

		largestLen := util.Max(len(left), len(right))

		for i := 0; i < largestLen; i++ {
			var valLeft, valRight uint64 = 0, 0
			if i < len(left) {
				valLeft = left[i]
			}

			if i < len(right) {
				valRight = right[i]
			}

			if valLeft != valRight {
				return valLeft < valRight
			}
		}

		return false
	})
}
