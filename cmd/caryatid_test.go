package main

import (
	"fmt"
	// "os"
	"path"
	"testing"
)

func TestArgumentParsing(t *testing.T) {
	var (
		cmdInv     = "/some/path/to/caryatid.test"
		_, cmdName = path.Split(cmdInv)
		boxName    = "ExampleBox"
		boxDesc    = "This is an example box"
		boxVers    = "1.2.3"
		boxProv    = "virtualbox"
		artifact   = fmt.Sprintf("%v_%v_%v.box", boxName, boxVers, boxProv)
		backend    = "copy"
		dest       = "/some/path/somewhere"
	)
	var (
		parsed ParsedArguments
		err    error
	)

	parsed, err = ParseArguments(cmdInv, boxName, boxDesc, boxVers, boxProv, artifact, backend, dest)
	if err != nil {
		t.Fatal(fmt.Sprintf("ParseArguments failed with error %v", err))
	}
	if parsed.CommandInvocation != cmdInv {
		t.Fatal(fmt.Sprintf("Expected .CommandInvocation to be '%v' but it was '%v'", cmdInv, parsed.CommandInvocation))
	}
	if parsed.CommandName != cmdName {
		t.Fatal(fmt.Sprintf("Expected .CommandName to be '%v' but it was '%v'", cmdName, parsed.CommandName))
	}
	if parsed.BoxName != boxName {
		t.Fatal(fmt.Sprintf("Expected .BoxName to be '%v' but it was '%v'", boxName, parsed.BoxName))
	}
	if parsed.BoxDescription != boxDesc {
		t.Fatal(fmt.Sprintf("Expected .BoxDescription to be '%v' but it was '%v'", boxDesc, parsed.BoxDescription))
	}
	if parsed.BoxVersion != boxVers {
		t.Fatal(fmt.Sprintf("Expected .BoxVersion to be '%v' but it was '%v'", boxVers, parsed.BoxVersion))
	}
	if parsed.BoxProvider != boxProv {
		t.Fatal(fmt.Sprintf("Expected .BoxProvider to be '%v' but it was '%v'", boxProv, parsed.BoxProvider))
	}
	if parsed.ArtifactPath != artifact {
		t.Fatal(fmt.Sprintf("Expected .ArtifactPath to be '%v' but it was '%v'", artifact, parsed.ArtifactPath))
	}
	if parsed.Backend != backend {
		t.Fatal(fmt.Sprintf("Expected .Backend to be '%v' but it was '%v'", backend, parsed.Backend))
	}
	if parsed.Destination != dest {
		t.Fatal(fmt.Sprintf("Expected .Destination to be '%v' but it was '%v'", dest, parsed.Destination))
	}

	parsed, err = ParseArguments(cmdInv, boxName, boxDesc, boxVers, boxProv, artifact, backend, dest)
	if err != nil {
		t.Fatal(fmt.Sprintf("Expected ParseArguments to succeed, but it failed: %v", err))
	}

	badArtifact := fmt.Sprintf("%v.NOTBOX", artifact)
	parsed, err = ParseArguments(cmdInv, boxName, boxDesc, boxVers, boxProv, badArtifact, backend, dest)
	if err == nil {
		t.Fatal(fmt.Sprintf("ParseArguments should have failed with an artifact name of %v but it did not", badArtifact))
	}

	// TODO: add more tests for all the failure modes of ParseArguments
}
