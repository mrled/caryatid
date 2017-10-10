package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/hashicorp/packer/packer"

	"github.com/mrled/caryatid/internal/util"
	"github.com/mrled/caryatid/pkg/caryatid"
)

// Integration tests
// NOTE: Call as 'go test -k' to keep the integration_tests directory around after the tests finish... otherwise it will be deleted

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
	fmt.Println(fmt.Sprintf("Detected running the test directory as '%v'", thisdir))

	err = os.MkdirAll(integrationTestDir, 0777)
	if err != nil {
		panic(fmt.Sprintf("Error trying to create test directory: %v", err))
	}

	testRv := m.Run()

	// os.Exit() doesn't respect defer, so we can't have defered the call to os.RemoveAll() at creation time
	if *keepFlag {
		fmt.Println(fmt.Sprintf("Will not remove integraion test dir after tests complete\n%v", integrationTestDir))
	} else {
		os.RemoveAll(integrationTestDir)
	}

	os.Exit(testRv)
}

func TestPostProcess(t *testing.T) {
	var (
		err                  error
		testBoxName          = "TestBoxName"
		testProviderName     = "TestProvider"
		testArtifactFilename = fmt.Sprintf("%v_%v.box", testBoxName, testProviderName)
		testArtifactPath     = path.Join(integrationTestDir, testArtifactFilename)
		ui                   = &packer.BasicUi{}
		inartifact           = &packer.MockArtifact{FilesValue: []string{testArtifactPath}}
		pp                   = CaryatidPostProcessor{}
		inkeepinput          = false
	)

	pp.config.CatalogUri = fmt.Sprintf("file://%v/%v.json", integrationTestDir, testBoxName)
	pp.config.Description = "Test box description"
	pp.config.KeepInputArtifact = inkeepinput
	pp.config.Name = testBoxName
	pp.config.Version = "6.6.6"

	// Set up test: write files etc
	err = caryatid.CreateTestBoxFile(testArtifactPath, testProviderName, true)
	if err != nil {
		t.Fatal(fmt.Sprintf("Error trying to write input artifact file: %v", err))
	}

	// Run the tests
	outArt, keepinputresult, err := pp.PostProcess(ui, inartifact)
	if err != nil {
		t.Fatal(fmt.Sprintf("Error during PostProcess(): %v", err))
	}
	if keepinputresult != inkeepinput {
		t.Fatal(fmt.Sprintf("Failed to keep input consistently"))
	}
	if outArt.BuilderId() != BuilderId {
		t.Fatal("BuildId does not match")
	}

	expectedCatalogStr := fmt.Sprintf(`{"name":"TestBoxName","description":"Test box description","versions":[{"version":"6.6.6","providers":[{"name":"TestProvider","url":"file://%v/TestBoxName/TestBoxName_6.6.6_TestProvider.box","checksum_type":"sha1","checksum":"2cca98d0ecfd03d57a3106950e14d724797f0836"}]}]}`, integrationTestDir)
	resultCatalogPath := path.Join(integrationTestDir, fmt.Sprintf("%v.json", testBoxName))
	resultCatalogData, err := ioutil.ReadFile(resultCatalogPath)
	if err != nil {
		t.Fatal("Error trying to read the test catalog: ", err)
	}
	var (
		expectedCatalog caryatid.Catalog
		resultCatalog   caryatid.Catalog
	)
	if err = json.Unmarshal([]byte(expectedCatalogStr), &expectedCatalog); err != nil {
		t.Fatal("Unable to unmarshal expected catalog")
	}
	if err = json.Unmarshal(resultCatalogData, &resultCatalog); err != nil {
		t.Fatal("Unable to unmarshal result catalog")
	}
	if !expectedCatalog.Equals(&resultCatalog) {
		t.Fatal(fmt.Sprintf("Catalog data did not match expectations\n\tExpected: %v\n\tResult:   %v", expectedCatalog, resultCatalog))
	}

	origDigest, err := util.Sha1sum(testArtifactPath)
	if err != nil {
		t.Fatal("Failed to calculate sha1sum of ", testArtifactPath)
	}
	copiedBoxPath := path.Join(integrationTestDir, testBoxName, fmt.Sprintf("%v_%v_%v.box", testBoxName, pp.config.Version, testProviderName))
	copiedDigest, err := util.Sha1sum(copiedBoxPath)
	if err != nil {
		t.Fatal("Failed to calculate sha1sum of ", copiedBoxPath)
	}
	if copiedDigest != origDigest {
		t.Fatal(fmt.Sprintf("Copying %v to %v failed... files are not identical", testArtifactPath, copiedBoxPath))
	}
}
