package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/mitchellh/packer/packer"
)

// Integration tests
// NOTE: Call as 'go test -k' to keep the integration_tests directory around after the tests finish... otherwise it will be deleted

const integrationTestDirName = "integration_tests"

var (
	_, thisfile, _, runtimeCallerOk = runtime.Caller(0)
	thisdir, _                      = path.Split(thisfile)
	integrationTestDir              = path.Join(thisdir, integrationTestDirName)
	keepFlag                        = flag.Bool("k", false, fmt.Sprintf("Keep the %v directory after running integration tests", integrationTestDir))
)

// A somewhat silly test that exists because we can't put logic outside of a function
func TestThatIntegrationTestingIsSetUpCorrectly(t *testing.T) {
	if !runtimeCallerOk {
		t.Fatal("Failed to detect thisdir using runtime.Caller()")
	}
	fmt.Println(fmt.Sprintf("Detected running the test directory as '%v'", thisdir))
}

func TestPostProcess(t *testing.T) {
	var (
		err error
	)

	testBoxName := "TestBoxName"
	testArtifactContents := "This is a test artifact"
	testProviderName := "TestProvider"
	testArtifactFilename := fmt.Sprintf("%v_%v.box", testBoxName, testProviderName)
	testArtifactPath := path.Join(integrationTestDir, testArtifactFilename)
	ui := &packer.BasicUi{}
	inartifact := &packer.MockArtifact{FilesValue: []string{testArtifactPath}}
	pp := CaryatidPostProcessor{}
	inkeepinput := false

	pp.config.CatalogRoot = fmt.Sprintf("file://%v", integrationTestDir)
	pp.config.Description = "Test box description"
	pp.config.KeepInputArtifact = inkeepinput
	pp.config.Name = testBoxName
	pp.config.Version = "6.6.6"

	// Set up test: write files etc
	err = os.MkdirAll(integrationTestDir, 0700)
	if err != nil {
		t.Fatal("Error trying to create test directory: ", err)
	}
	if *keepFlag {
		fmt.Printf("Will not remove integraion test dir '%v' after tests complete", integrationTestDir)
	} else {
		defer os.RemoveAll(integrationTestDir)
	}

	err = ioutil.WriteFile(testArtifactPath, []byte(testArtifactContents), 0600)
	if err != nil {
		t.Fatal(fmt.Sprintf("Error trying to write file: ", err))
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

	// Can't test these because we aren't getting what Packer thinks is a real BoxArtifact, just a packer.Artifact
	// testArtifactSha1Sum := "78bc8a542fa84494ff14ae412196d134c603960c"
	// if outArt.Checksum != testArtifactSha1Sum {
	// 	t.Fatal(fmt.Sprintf("Expected checksum of '%v' but got checksum of '%v'", testArtifactSha1Sum, outArt.Checksum))
	// }

	expectedCatalogData := `{"name":"TestBoxName","description":"Test box description","versions":[{"version":"6.6.6","providers":[{"name":"TestProvider","url":"file:///Users/mrled/Documents/Go/src/github.com/mrled/packer-post-processor-caryatid/integration_test/TestBoxName/TestBoxName_6.6.6_TestProvider.box","checksum_type":"sha1","checksum":"78bc8a542fa84494ff14ae412196d134c603960c"}]}]}`
	testCatalogPath := path.Join(integrationTestDir, fmt.Sprintf("%v.json", testBoxName))
	testCatalogData, err := ioutil.ReadFile(testCatalogPath)
	if err != nil {
		t.Fatal("Error trying to read the test catalog: ", err)
	}
	if string(testCatalogData) != expectedCatalogData {
		t.Fatal("Catalog data did not match expectations", testCatalogData, expectedCatalogData)
	}

	origDigest, err := sha1sum(testArtifactPath)
	if err != nil {
		t.Fatal("Failed to calculate sha1sum of ", testArtifactPath)
	}
	copiedBoxPath := path.Join(integrationTestDir, testBoxName, fmt.Sprintf("%v_%v_%v.box", testBoxName, pp.config.Version, testProviderName))
	copiedDigest, err := sha1sum(copiedBoxPath)
	if err != nil {
		t.Fatal("Failed to calculate sha1sum of ", copiedBoxPath)
	}
	if copiedDigest != origDigest {
		t.Fatal(fmt.Sprintf("Copying %v to %v failed... files are not identical", testArtifactPath, copiedBoxPath))
	}
}
