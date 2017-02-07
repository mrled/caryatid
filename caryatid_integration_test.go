package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/mitchellh/packer/packer"
)

func TestPostProcess(t *testing.T) {
	var (
		err error
	)

	_, thisfile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("Failed to detect thisdir using runtime.Caller()")
	}
	thisdir, _ := path.Split(thisfile)
	fmt.Println(fmt.Sprintf("Detected running the test directory as '%v'", thisdir))
	testdir := path.Join(thisdir, "integration_test")

	testBoxName := "TestBoxName"
	testArtifactContents := "This is a test artifact"
	testArtifactSha1Sum := "78bc8a542fa84494ff14ae412196d134c603960c"
	testProviderName := "TestProvider"
	testArtifactFilename := fmt.Sprintf("%v_%v.box", testBoxName, testProviderName)
	testArtifactPath := path.Join(testdir, testArtifactFilename)
	ui := &packer.BasicUi{}
	inartifact := &packer.MockArtifact{FilesValue: []string{testArtifactPath}}
	pp := PostProcessor{}
	inkeepinput := false

	pp.config.CatalogRoot = fmt.Sprintf("file://%v", testdir)
	pp.config.Description = "Test box description"
	pp.config.KeepInputArtifact = inkeepinput
	pp.config.Name = testBoxName
	pp.config.Version = "6.6.6"

	// Set up test: write files etc
	err = os.MkdirAll(testdir, 0700)
	if err != nil {
		t.Fatal("Error trying to create test directory: ", err)
	}
	defer os.RemoveAll(testdir)
	err = ioutil.WriteFile(testArtifactPath, []byte(testArtifactContents), 0600)
	if err != nil {
		t.Fatal(fmt.Sprintf("Error trying to write file: ", err))
	}

	// Run the tests
	boxArt, keepinputresult, err := pp.PostProcess(ui, inartifact)
	if err != nil {
		t.Fatal(fmt.Sprintf("Error during PostProcess(): %v", err))
	}
	if keepinputresult != inkeepinput {
		t.Fatal(fmt.Sprintf("Failed to keep input consistently"))
	}
	if boxArt.Checksum != testArtifactSha1Sum {
		// t.Fatal(fmt.Sprint("Expected checksum of '%v' but got checksum of '%v'", testArtifactSha1Sum, boxArt.Checksum))
		t.Fatal("Expected checksum of '%v' but got checksum of '%v'", testArtifactSha1Sum, boxArt.Checksum)
	}

	expectedCatalogData := `{"name":"TestBoxName","description":"Test box description","versions":[{"version":"6.6.6","providers":[{"name":"TestProvider","url":"file:///Users/mrled/Documents/Go/src/github.com/mrled/packer-post-processor-caryatid/integration_test/TestBoxName/TestBoxName_6.6.6_TestProvider.box","checksum_type":"sha1","checksum":"78bc8a542fa84494ff14ae412196d134c603960c"}]}]}`
	testCatalogPath := path.Join(testdir, fmt.Sprintf("%v.json", testBoxName))
	testCatalogData, err := ioutil.ReadFile(testCatalogPath)
	if err != nil {
		t.Fatal("Error trying to read the test catalog: ", err)
	}
	if string(testCatalogData) != expectedCatalogData {
		t.Fatal("Catalog data did not match expectations", testCatalogData, expectedCatalogData)
	}
}
