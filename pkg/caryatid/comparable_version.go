//go:generate stringer -type=VersionComparator -output=comparable_version_string.go

package caryatid

import (
	"fmt"
	"strconv"
	"strings"
)

/*
VersionComparator represents the numerical relationship between to Version structs
VersionEquals indicates that the two structs are equal
VersionEqualsPrereleaseMismatch indicates that the two structs have equal numerical versions, but mismatched Prerelease tags
VersionLessThan indicates that the Version in question has a lower numerical version than the one its compared to
VersionGreaterThan indicates that the Version in question has a higher numerical version than the one its compared to
*/

type VersionComparator int

const (
	VersionEquals                   VersionComparator = iota
	VersionEqualsPrereleaseMismatch VersionComparator = iota
	VersionLessThan                 VersionComparator = iota
	VersionGreaterThan              VersionComparator = iota
)

type VersionComparatorList []VersionComparator

func (list1 *VersionComparatorList) Contains(list2 VersionComparatorList) bool {
	for _, cv2 := range list2 {

		contained := false
		for _, cv1 := range *list1 {
			if cv1 == cv2 {
				contained = true
			}
		}

		if !contained {
			return false
		}
	}
	return true
}

// ComparableVersion represents a semantic version
// It holds an array of ints representing the version, and a string representing the prerelease tag
// Example semantic version 1.5.3-BETA: ComparableVersion{[]int{1, 5, 3} "BETA"}
type ComparableVersion struct {
	Version    []int
	Prerelease string
}

// ComparableVersion returns a ComparableVersion struct for a semver string
func NewComparableVersion(semver string) (cvers ComparableVersion, err error) {
	var verStr string
	if strings.Contains(semver, "-") {
		splitSemver := strings.Split(semver, "-")
		if len(splitSemver) > 2 {
			err = fmt.Errorf("Too many dash (-) characters in semver '%v'\n", semver)
			return
		}
		verStr = splitSemver[0]
		cvers.Prerelease = splitSemver[1]
	} else {
		verStr = semver
	}

	splitVers := strings.Split(verStr, ".")
	for _, strComponent := range splitVers {
		component, parseIntErr := strconv.ParseInt(strComponent, 10, 0)
		if parseIntErr != nil {
			err = fmt.Errorf("Could not decode component '%v' from version string '%v': %v\n", strComponent, semver, parseIntErr)
			return
		}
		cvers.Version = append(cvers.Version, int(component))
	}
	return
}

// NewVersionComparator returns a new VersionComparator from a comparator string
// Input may be one of < > = <= >=
// NOTE that <= and >= comparator strings will also match equal versions with different prereleases
// so `=` will return only VersionEquals,
// but `>=` will return VersionEquals, VersionEqualsPrereleaseMismatch, and VersionGreaterThan
func NewVersionComparator(compstring string) (comparators VersionComparatorList, err error) {
	switch compstring {
	case "<":
		comparators = VersionComparatorList{VersionLessThan}
	case ">":
		comparators = VersionComparatorList{VersionGreaterThan}
	case "=":
		comparators = VersionComparatorList{VersionEquals}
	case "<=":
		comparators = VersionComparatorList{VersionEquals, VersionEqualsPrereleaseMismatch, VersionLessThan}
	case ">=":
		comparators = VersionComparatorList{VersionEquals, VersionEqualsPrereleaseMismatch, VersionGreaterThan}
	default:
		err = fmt.Errorf("Invalid comparator string '%v'\n", compstring)
	}
	return
}

// CompareIntArray returns a VersionComparator to represent the relationship between two arrays of integers
// If the arrays are not of equal length, the smaller array will have zeroes appended to it for the sake of the comparison
func CompareIntArray(va1 []int, va2 []int) VersionComparator {
	if len(va1) == 0 || len(va2) == 0 {
		if len(va1) == len(va2) {
			return VersionEquals
		} else if len(va1) > 0 {
			return VersionGreaterThan
		} else { // if len(va2) > 0 {
			return VersionLessThan
		}
	}
	if va1[0] == va2[0] {
		if len(va1) == 1 && len(va2) == 1 {
			return VersionEquals
		} else {
			if len(va1) <= 1 {
				va1 = append(va1, 0)
			} else if len(va2) <= 1 {
				va2 = append(va2, 0)
			}
			return CompareIntArray(va1[1:], va2[1:])
		}
	} else if va1[0] > va2[0] {
		return VersionGreaterThan
	} else { // if va1[0] < va2[0] {
		return VersionLessThan
	}
}

// Compare returns a VersionComparator to represent the relationship between two ComparableVersion structs
func (cv1 *ComparableVersion) Compare(cv2 *ComparableVersion) VersionComparator {
	vResult := CompareIntArray(cv1.Version, cv2.Version)
	if vResult == VersionEquals && cv1.Prerelease != cv2.Prerelease {
		return VersionEqualsPrereleaseMismatch
	} else {
		return vResult
	}
}
