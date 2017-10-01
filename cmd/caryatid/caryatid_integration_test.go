package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/mrled/caryatid/pkg/caryatid"
)

const integrationTestDirName = "integration_tests"

var (
	_, thisfile, _, runtimeCallerOk = runtime.Caller(0)
	thisdir, _                      = path.Split(thisfile)
	integrationTestDir              = path.Join(thisdir, integrationTestDirName)
)

func TestMain(m *testing.M) {
	var (
		err      error
		keepFlag = flag.Bool("k", false, fmt.Sprintf("Keep the %v directory after running integration tests", integrationTestDir))
	)

	// Have to check this here because we can't put logic outside of a function
	if !runtimeCallerOk {
		panic("Failed to detect thisdir using runtime.Caller()")
	}
	fmt.Printf("Detected running the test directory as '%v'\n", thisdir)

	err = os.MkdirAll(integrationTestDir, 0777)
	if err != nil {
		panic(fmt.Sprintf("Error trying to create test directory: %v\n", err))
	}

	testRv := m.Run()

	// os.Exit() doesn't respect defer, so we can't have defered the call to os.RemoveAll() at creation time
	if *keepFlag {
		fmt.Printf("Will not remove integraion test dir after tests complete\n%v\n", integrationTestDir)
	} else {
		os.RemoveAll(integrationTestDir)
	}

	os.Exit(testRv)
}

func TestShowAction(t *testing.T) {
	var (
		err    error
		result string

		boxName         = "TestShowActionBox"
		boxDesc         = "TestShowActionBox Description"
		catalogRootPath = integrationTestDir
		catalogPath     = path.Join(catalogRootPath, fmt.Sprintf("%v.json", boxName))
		catalogRootUri  = fmt.Sprintf("file://%v", catalogRootPath)
	)

	catalog := caryatid.Catalog{
		boxName,
		boxDesc,
		[]caryatid.Version{
			caryatid.Version{
				"1.5.3",
				[]caryatid.Provider{
					caryatid.Provider{
						"test-provider",
						"test:///asdf/asdfqwer/something.box",
						"FakeChecksum",
						"0xDECAFBAD",
					},
				},
			},
		},
	}
	expectedCatalogString := `{TestShowActionBox TestShowActionBox Description [{1.5.3 [{test-provider test:///asdf/asdfqwer/something.box FakeChecksum 0xDECAFBAD}]}]}
`

	jsonCatalog, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		t.Fatalf("Error trying to marshal catalog: %v\n", err)
	}

	err = ioutil.WriteFile(catalogPath, jsonCatalog, 0666)
	if err != nil {
		t.Fatalf("Error trying to write catalog: %v\n", err)
	}

	result, err = showAction(catalogRootUri, boxName)
	if err != nil {
		t.Fatalf("showAction() error: %v\n", err)
	}
	if result != expectedCatalogString {
		t.Fatalf("showAction() result was\n%v\nBut we expected it to be\n%v\nSad times :(", result, expectedCatalogString)
	}
}

func TestCreateTestBoxAction(t *testing.T) {
	var (
		err     error
		boxPath = path.Join(integrationTestDir, "TestCreateTestBoxAction.box")
	)

	err = createTestBoxAction(boxPath, "TestProvider")
	if err != nil {
		t.Fatalf("createTestBoxAction() failed with error: %v\n", err)
	}
}

func TestAddAction(t *testing.T) {

	type ExpectedMatch struct {
		Name string
		In   string
		Out  string
	}

	var (
		err             error
		catalogBytes    []byte
		catalog         caryatid.Catalog
		expectedMatches []ExpectedMatch

		boxPath        = path.Join(integrationTestDir, "incoming-TestAddAction.box")
		boxProvider    = "TestAddActionProvider"
		boxName        = "TestAddActionBox"
		boxDesc        = "TestAddActionBox is a test box"
		boxVersion     = "1.6.3"
		boxVersion2    = "2.0.1"
		catalogRootUri = fmt.Sprintf("file://%v", integrationTestDir)
		catalogPath    = path.Join(integrationTestDir, fmt.Sprintf("%v.json", boxName))
	)

	if err = caryatid.CreateTestBoxFile(boxPath, boxProvider, true); err != nil {
		t.Fatalf("TestAddAction(): Error trying to create test box file: %v\n", err)
	}

	// Test adding to an empty catalog
	err = addAction(boxPath, boxName, boxDesc, boxVersion, catalogRootUri)
	if err != nil {
		t.Fatalf("addAction() failed with error: %v\n", err)
	}

	catalogBytes, err = ioutil.ReadFile(catalogPath)
	if err != nil {
		t.Fatalf("Could not read catalog we just created at '%v'\n", catalogPath)
	}

	if err = json.Unmarshal(catalogBytes, &catalog); err != nil {
		t.Fatalf("Error trying to marshal the catalog: %v\n", err)
	}

	expectedMatches = []ExpectedMatch{
		ExpectedMatch{"catalog name", catalog.Name, boxName},
		ExpectedMatch{"catalog description", catalog.Description, boxDesc},
		ExpectedMatch{"box provider", catalog.Versions[0].Providers[0].Name, boxProvider},
		ExpectedMatch{"box version", catalog.Versions[0].Version, boxVersion},
	}
	for _, match := range expectedMatches {
		if match.In != match.Out {
			t.Fatalf("Expected %v to match, but the expected value was %v while the actual value was %v", match.Name, match.In, match.Out)
		}
	}

	// Test adding another box to the same, now non-empty, catalog
	err = addAction(boxPath, boxName, boxDesc, boxVersion2, catalogRootUri)
	if err != nil {
		t.Fatalf("addAction() failed with error: %v\n", err)
	}

	catalogBytes, err = ioutil.ReadFile(catalogPath)
	if err != nil {
		t.Fatalf("Could not read catalog we just created at '%v'\n", catalogPath)
	}

	if err = json.Unmarshal(catalogBytes, &catalog); err != nil {
		t.Fatalf("Error trying to marshal the catalog: %v\n", err)
	}

	expectedMatches = []ExpectedMatch{
		ExpectedMatch{"catalog name", catalog.Name, boxName},
		ExpectedMatch{"catalog description", catalog.Description, boxDesc},
		ExpectedMatch{"box provider", catalog.Versions[1].Providers[0].Name, boxProvider},
		ExpectedMatch{"box version", catalog.Versions[1].Version, boxVersion2},
	}
	for _, match := range expectedMatches {
		if match.In != match.Out {
			t.Fatalf("Expected %v to match, but the expected value was %v while the actual value was %v", match.Name, match.In, match.Out)
		}
	}
}

func TestQueryAction(t *testing.T) {
	var (
		err         error
		boxArtifact caryatid.BoxArtifact
		result      caryatid.Catalog

		boxProvider1 = "StrongSapling"
		boxProvider2 = "FeebleFungus"
		boxPath1     = path.Join(integrationTestDir, "incoming-TestQueryActionBox-1.box")
		boxPath2     = path.Join(integrationTestDir, "incoming-TestQueryActionBox-2.box")
		boxVersions1 = []string{"0.3.5", "0.3.5-BETA", "1.0.0", "1.0.0-PRE", "1.4.5", "1.2.3", "1.2.4"}
		boxVersions2 = []string{"0.3.4", "0.3.5-BETA", "1.0.1", "2.0.0", "2.10.0", "2.11.1", "1.2.3"}

		boxName        = "TestQueryActionBox"
		boxDesc        = "TestQueryActionBox is a test box"
		catalogRootUri = fmt.Sprintf("file://%v", integrationTestDir)
		digestType     = "TestQueryActionDigestType"
		digest         = "0xB00B1E5"
	)

	// Set up manager
	manager, err := getManager(catalogRootUri, boxName)
	if err != nil {
		log.Printf("Error getting a BackendManager")
		return
	}

	// Create the *input* boxes - that is, boxes that would come from packer-post-processor-vagrant
	if err = caryatid.CreateTestBoxFile(boxPath1, boxProvider1, true); err != nil {
		t.Fatalf("TestAddAction(): Error trying to create test box file: %v\n", err)
	}
	if err = caryatid.CreateTestBoxFile(boxPath2, boxProvider2, true); err != nil {
		t.Fatalf("TestAddAction(): Error trying to create test box file: %v\n", err)
	}

	// Now copy those boxes multiple times to the Catalog,
	// as if they were different versions each time
	for _, version := range boxVersions1 {
		boxArtifact = caryatid.BoxArtifact{Path: boxPath1, Name: boxName, Description: boxDesc, Version: version, Provider: boxProvider1, CatalogRootUri: catalogRootUri, ChecksumType: digestType, Checksum: digest}
		if err = manager.AddBox(&boxArtifact); err != nil {
			t.Fatalf("Error adding box metadata to catalog: %v\n", err)
			return
		}
	}
	for _, version := range boxVersions2 {
		boxArtifact = caryatid.BoxArtifact{Path: boxPath2, Name: boxName, Description: boxDesc, Version: version, Provider: boxProvider2, CatalogRootUri: catalogRootUri, ChecksumType: digestType, Checksum: digest}
		if err = manager.AddBox(&boxArtifact); err != nil {
			t.Fatalf("Error adding box metadata to catalog: %v\n", err)
			return
		}
	}

	type TestCase struct {
		VersionQuery   string
		ProviderQuery  string
		ExpectedResult caryatid.Catalog
	}

	testCases := []TestCase{
		TestCase{ // Expect all items in catalog
			"", "",
			caryatid.Catalog{boxName, boxDesc, []caryatid.Version{
				caryatid.Version{"0.3.5", []caryatid.Provider{
					caryatid.Provider{boxProvider1, "FAKEURI", digestType, digest},
				}},
				caryatid.Version{"0.3.5-BETA", []caryatid.Provider{
					caryatid.Provider{boxProvider1, "FAKEURI", digestType, digest},
					caryatid.Provider{boxProvider2, "FAKEURI", digestType, digest},
				}},
				caryatid.Version{"1.0.0", []caryatid.Provider{
					caryatid.Provider{boxProvider1, "FAKEURI", digestType, digest},
				}},
				caryatid.Version{"1.0.0-PRE", []caryatid.Provider{
					caryatid.Provider{boxProvider1, "FAKEURI", digestType, digest},
				}},
				caryatid.Version{"1.4.5", []caryatid.Provider{
					caryatid.Provider{boxProvider1, "FAKEURI", digestType, digest},
				}},
				caryatid.Version{"1.2.3", []caryatid.Provider{
					caryatid.Provider{boxProvider1, "FAKEURI", digestType, digest},
					caryatid.Provider{boxProvider2, "FAKEURI", digestType, digest},
				}},
				caryatid.Version{"1.2.4", []caryatid.Provider{
					caryatid.Provider{boxProvider1, "FAKEURI", digestType, digest},
				}},
				caryatid.Version{"0.3.4", []caryatid.Provider{
					caryatid.Provider{boxProvider2, "FAKEURI", digestType, digest},
				}},
				caryatid.Version{"1.0.1", []caryatid.Provider{
					caryatid.Provider{boxProvider2, "FAKEURI", digestType, digest},
				}},
				caryatid.Version{"2.0.0", []caryatid.Provider{
					caryatid.Provider{boxProvider2, "FAKEURI", digestType, digest},
				}},
				caryatid.Version{"2.10.0", []caryatid.Provider{
					caryatid.Provider{boxProvider2, "FAKEURI", digestType, digest},
				}},
				caryatid.Version{"2.11.1", []caryatid.Provider{
					caryatid.Provider{boxProvider2, "FAKEURI", digestType, digest},
				}},
			}},
		},
		TestCase{
			"", "rongSap",
			caryatid.Catalog{boxName, boxDesc, []caryatid.Version{
				caryatid.Version{"0.3.5", []caryatid.Provider{
					caryatid.Provider{boxProvider1, "FAKEURI", digestType, digest},
				}},
				caryatid.Version{"0.3.5-BETA", []caryatid.Provider{
					caryatid.Provider{boxProvider1, "FAKEURI", digestType, digest},
				}},
				caryatid.Version{"1.0.0", []caryatid.Provider{
					caryatid.Provider{boxProvider1, "FAKEURI", digestType, digest},
				}},
				caryatid.Version{"1.0.0-PRE", []caryatid.Provider{
					caryatid.Provider{boxProvider1, "FAKEURI", digestType, digest},
				}},
				caryatid.Version{"1.4.5", []caryatid.Provider{
					caryatid.Provider{boxProvider1, "FAKEURI", digestType, digest},
				}},
				caryatid.Version{"1.2.3", []caryatid.Provider{
					caryatid.Provider{boxProvider1, "FAKEURI", digestType, digest},
				}},
				caryatid.Version{"1.2.4", []caryatid.Provider{
					caryatid.Provider{boxProvider1, "FAKEURI", digestType, digest},
				}},
			}},
		},
		TestCase{
			"<1", "",
			caryatid.Catalog{boxName, boxDesc, []caryatid.Version{
				caryatid.Version{"0.3.5", []caryatid.Provider{
					caryatid.Provider{boxProvider1, "FAKEURI", digestType, digest},
				}},
				caryatid.Version{"0.3.5-BETA", []caryatid.Provider{
					caryatid.Provider{boxProvider1, "FAKEURI", digestType, digest},
					caryatid.Provider{boxProvider2, "FAKEURI", digestType, digest},
				}},
				caryatid.Version{"0.3.4", []caryatid.Provider{
					caryatid.Provider{boxProvider2, "FAKEURI", digestType, digest},
				}},
			}},
		},
		TestCase{
			"<1", ".*rongSap.*",
			caryatid.Catalog{boxName, boxDesc, []caryatid.Version{
				caryatid.Version{"0.3.5", []caryatid.Provider{
					caryatid.Provider{boxProvider1, "FAKEURI", digestType, digest},
				}},
				caryatid.Version{"0.3.5-BETA", []caryatid.Provider{
					caryatid.Provider{boxProvider1, "FAKEURI", digestType, digest},
				}},
			}},
		},
	}

	fuzzyEqualsParams := caryatid.CatalogFuzzyEqualsParams{SkipProviderUrl: true}

	for _, tc := range testCases {
		// Join the array into a multi-line string, and add a trailing newline
		result, err = queryAction(catalogRootUri, boxName, tc.VersionQuery, tc.ProviderQuery)
		if err != nil {
			t.Fatalf("queryAction(*, *, '%v', '%v') returned an unexpected error: %v\n", tc.VersionQuery, tc.ProviderQuery, err)
		} else if !result.FuzzyEquals(&tc.ExpectedResult, fuzzyEqualsParams) {
			t.Fatalf(
				"queryAction(*, *, '%v', '%v') returned result:\n%v\nBut we expected:\n%v\n",
				tc.VersionQuery, tc.ProviderQuery, result.DisplayString(), tc.ExpectedResult.DisplayString())
		}
	}
}

func TestDeleteAction(t *testing.T) {
	t.Logf("TODO: TestDeleteAction() HAS NO TESTS DEFINED\n")
}
