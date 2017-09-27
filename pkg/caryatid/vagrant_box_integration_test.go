package caryatid

import (
	"flag"
	"fmt"
	"os"
	"path"
	"runtime"
	"testing"
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

func TestDetermineProvider(t *testing.T) {
	var (
		err                error
		resultProviderName string
		testProviderName   = "TESTPROVIDER"
		testGzipArtifact   = path.Join(integrationTestDir, "testDetProvGz.box")
		testTarArtifact    = path.Join(integrationTestDir, "testDetProvTar.box")
	)

	err = CreateTestBoxFile(testGzipArtifact, testProviderName, true)
	if err != nil {
		t.Fatal(fmt.Sprintf("Error trying to write input artifact file: %v", err))
	}

	resultProviderName, err = DetermineProvider(testGzipArtifact)
	if err != nil {
		t.Fatal("Error trying to determine provider: ", err)
	}
	if resultProviderName != testProviderName {
		t.Fatal("Expected provider name does not match result provider name: ", testProviderName, resultProviderName)
	}

	err = CreateTestBoxFile(testTarArtifact, testProviderName, false)
	if err != nil {
		t.Fatal(fmt.Sprintf("Error trying to write input artifact file: %v", err))
	}

	resultProviderName, err = DetermineProvider(testTarArtifact)
	if err != nil {
		t.Fatal("Error trying to determine provider: ", err)
	}
	if resultProviderName != testProviderName {
		t.Fatal("Expected provider name does not match result provider name: ", testProviderName, resultProviderName)
	}
}
