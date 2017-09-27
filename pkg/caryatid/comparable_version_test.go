package caryatid

import (
	"fmt"
	"testing"
)

func TestNewComparableVersion(t *testing.T) {
	intArraysAreEqual := func(ia1 []int, ia2 []int) bool {
		if len(ia1) != len(ia2) {
			return false
		}
		for idx, _ := range ia1 {
			if ia1[idx] != ia2[idx] {
				return false
			}
		}
		return true
	}

	type TestCase struct {
		SemVer      string
		ExpectedVer []int
		ExpectedTag string
		ExpectedErr bool
	}

	testCases := []TestCase{
		TestCase{"1.2.3", []int{1, 2, 3}, "", false},
		TestCase{"1.2.3-LONGASSPRERELEASE", []int{1, 2, 3}, "LONGASSPRERELEASE", false},
		TestCase{"10.20.30", []int{10, 20, 30}, "", false},
		TestCase{"0-X", []int{0}, "X", false},
		TestCase{"-JUSTPRERELEASE", []int{}, "", true},
		TestCase{"-X", []int{}, "", true},
		TestCase{"-4", []int{}, "", true},
		TestCase{"JUSTTEXT", []int{}, "", true},
		TestCase{"-1.2.3-PREREL", []int{}, "", true},
		TestCase{"1.-2.3-PRE", []int{}, "", true},
	}

	for _, tc := range testCases {
		result, err := NewComparableVersion(tc.SemVer)
		callMsg := fmt.Sprintf("NewComparableVersion(%v)", tc.SemVer)
		if tc.ExpectedErr {
			if err == nil {
				t.Fatalf("%v was expected to return an error, but returned a result '%v'\n", callMsg, result)
			}
		} else {
			if err != nil {
				t.Fatalf("%v returned an unexpected error '%v'\n", callMsg, err)
			} else if !intArraysAreEqual(result.Version, tc.ExpectedVer) {
				t.Fatalf("%v returned a .Version of '%v' but we expected '%v'\n", callMsg, result.Version, tc.ExpectedVer)
			} else if result.Prerelease != tc.ExpectedTag {
				t.Fatalf("%v returned a .Prerelease of '%v' but we expected '%v'\n", callMsg, result.Prerelease, tc.ExpectedTag)
			}
		}

	}
}

func TestCompareIntArray(t *testing.T) {
	type TestCase struct {
		A1        []int
		A2        []int
		ExpResult VersionComparator
	}
	aVersArr := []int{1, 0, 0}
	testCases := []TestCase{
		TestCase{aVersArr, aVersArr, VersionEquals},
		TestCase{[]int{1, 0, 0}, []int{1, 0, 0}, VersionEquals},
		TestCase{[]int{1, 0, 0}, []int{1, 0, 1}, VersionLessThan},
		TestCase{[]int{1, 0, 1}, []int{1, 0, 0}, VersionGreaterThan},
		TestCase{[]int{1, 0, 0}, []int{1, 1, 0}, VersionLessThan},
		TestCase{[]int{1, 1, 0}, []int{1, 0, 0}, VersionGreaterThan},
		TestCase{[]int{1, 1, 0}, []int{2, 1, 0}, VersionLessThan},
		TestCase{[]int{2, 1, 0}, []int{1, 0, 0}, VersionGreaterThan},
		TestCase{[]int{1, 0, 0}, []int{1, 0, 0, 0}, VersionEquals},
		TestCase{[]int{1, 0, 0, 0}, []int{1, 0, 0}, VersionEquals},
		TestCase{[]int{2, 0, 0}, []int{1, 0, 0, 0}, VersionGreaterThan},
		TestCase{[]int{1, 0, 0, 0}, []int{2, 0, 0}, VersionLessThan},
		TestCase{[]int{2}, []int{2, 1, 1, 1}, VersionLessThan},
		TestCase{[]int{2, 1, 1, 1}, []int{2}, VersionGreaterThan},
	}
	for _, tc := range testCases {
		if result := CompareIntArray(tc.A1, tc.A2); result != tc.ExpResult {
			t.Fatalf("CompareIntArray(%v, %v) returned '%v' but we expected '%v'\n", tc.A1, tc.A2, result, tc.ExpResult)
		}
	}
}

func TestComparableVersionCompare(t *testing.T) {
	type TestCase struct {
		v1    ComparableVersion
		v2    ComparableVersion
		exres VersionComparator
	}
	var (
		v100, _      = NewComparableVersion("1.0.0")
		v100ALPHA, _ = NewComparableVersion("1.0.0-ALPHA")
		v100BETA, _  = NewComparableVersion("1.0.0-BETA")
		v100BETAb, _ = NewComparableVersion("1.0.0-BETA")
	)
	testCases := []TestCase{
		TestCase{v100BETA, v100BETA, VersionEquals},
		TestCase{v100BETA, v100BETAb, VersionEquals},
		TestCase{v100, v100BETA, VersionEqualsPrereleaseMismatch},
		TestCase{v100ALPHA, v100BETA, VersionEqualsPrereleaseMismatch},
	}

	for _, tcase := range testCases {
		if result := tcase.v1.Compare(&tcase.v2); result != tcase.exres {
			t.Fatalf(
				"Expected '%v' .Compare '%v' == '%v', but it returned '%v' instead\n",
				tcase.v1, tcase.v2, tcase.exres.String(), result.String())
		}
	}
}
