package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
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
		panic(fmt.Sprintf("Error trying to create test directory: ", err))
	}

	testRv := m.Run()

	// os.Exit() doesn't respect defer, so we can't have defered the call to os.RemoveAll() at creation time
	if *keepFlag {
		fmt.Printf("Will not remove integraion test dir '%v' after tests complete", integrationTestDir)
	} else {
		os.RemoveAll(integrationTestDir)
	}

	os.Exit(testRv)
}

func TestDetermineProviderFromMetadata(t *testing.T) {
	var err error

	// Create a buffer to write our archive to.
	buf := new(bytes.Buffer)

	// Create a new zip archive.
	w := zip.NewWriter(buf)
	md, err := w.Create("metadata.json")
	if err != nil {
		t.Fatal("Failed to create zipped metadata.json file: ", err)
	}
	testProviderName := "TESTPROVIDER"
	_, err = md.Write([]byte(fmt.Sprintf(`{"provider": "%v"}`, testProviderName)))

	// Make sure to check the error on Close.
	err = w.Close()
	if err != nil {
		log.Fatal(err)
	}

	zipFilePath := path.Join(integrationTestDir, "test.zip")
	zipFile, err := os.Create(zipFilePath)
	if err != nil {
		t.Fatal("Failed to create zipfile: ", err)
	}
	if _, err = zipFile.Write(buf.Bytes()); err != nil {
		t.Fatal("Error writing zipfile: ", err)
	}

	resultProviderName, err := determineProviderFromMetadata(zipFilePath)
	if err != nil {
		t.Fatal("Error trying to determine provider: ", err)
	}
	if resultProviderName != testProviderName {
		t.Fatal("Expected provider name does not match result provider name: ", testProviderName, resultProviderName)
	}
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
	err = ioutil.WriteFile(testArtifactPath, []byte(testArtifactContents), 0666)
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

	expectedCatalogStr := fmt.Sprintf(`{"name":"TestBoxName","description":"Test box description","versions":[{"version":"6.6.6","providers":[{"name":"TestProvider","url":"file://%v/TestBoxName/TestBoxName_6.6.6_TestProvider.box","checksum_type":"sha1","checksum":"78bc8a542fa84494ff14ae412196d134c603960c"}]}]}`, integrationTestDir)
	resultCatalogPath := path.Join(integrationTestDir, fmt.Sprintf("%v.json", testBoxName))
	resultCatalogData, err := ioutil.ReadFile(resultCatalogPath)
	if err != nil {
		t.Fatal("Error trying to read the test catalog: ", err)
	}
	var (
		expectedCatalog Catalog
		resultCatalog   Catalog
	)
	if err = json.Unmarshal([]byte(expectedCatalogStr), &expectedCatalog); err != nil {
		t.Fatal("Unable to unmarshal expected catalog")
	}
	if err = json.Unmarshal(resultCatalogData, &resultCatalog); err != nil {
		t.Fatal("Unable to unmarshal result catalog")
	}
	if !expectedCatalog.Equals(resultCatalog) {
		t.Fatal(fmt.Sprintf("Catalog data did not match expectations\n\tExpected: %v\n\tResult:   %v", expectedCatalog, resultCatalog))
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
