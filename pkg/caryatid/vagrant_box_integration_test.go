package caryatid

import (
	"fmt"
	"path"
	"testing"

	"github.com/mrled/caryatid/pkg/caryatid"
)

func TestDetermineProvider(t *testing.T) {
	var (
		err                error
		resultProviderName string
		testProviderName   = "TESTPROVIDER"
		testGzipArtifact   = path.Join(integrationTestDir, "testDetProvGz.box")
		testTarArtifact    = path.Join(integrationTestDir, "testDetProvTar.box")
	)

	err = createTestBoxFile(testGzipArtifact, testProviderName, true)
	if err != nil {
		t.Fatal(fmt.Sprintf("Error trying to write input artifact file: %v", err))
	}

	resultProviderName, err = caryatid.DetermineProvider(testGzipArtifact)
	if err != nil {
		t.Fatal("Error trying to determine provider: ", err)
	}
	if resultProviderName != testProviderName {
		t.Fatal("Expected provider name does not match result provider name: ", testProviderName, resultProviderName)
	}

	err = createTestBoxFile(testTarArtifact, testProviderName, false)
	if err != nil {
		t.Fatal(fmt.Sprintf("Error trying to write input artifact file: %v", err))
	}

	resultProviderName, err = determineProvider(testTarArtifact)
	if err != nil {
		t.Fatal("Error trying to determine provider: ", err)
	}
	if resultProviderName != testProviderName {
		t.Fatal("Expected provider name does not match result provider name: ", testProviderName, resultProviderName)
	}
}
