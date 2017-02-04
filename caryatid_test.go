package main

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestJsonDecodingProvider(t *testing.T) {
	jstring := `{"name":"testname","url":"http://example.com/whatever","checksum_type":"dummy","checksum":"dummy"}`
	var prov Provider
	err := json.Unmarshal([]byte(jstring), &prov)
	if err != nil {
		t.Fatal(fmt.Sprintf("Error unmarshalling JSON: %s", err))
	}
	if prov.Name != "testname" {
		t.Fatal(fmt.Sprintf("Decoded JSON object had bad Name property; should be 'testname' but was '%s'", prov.Name))
	}
}

func TestJsonDecodingCatalog(t *testing.T) {
	jstring := `{"name":"examplebox","description":"this is an example box","versions":[{"version":"12.34.56","providers":[{"name":"testname","url":"http://example.com/whatever","checksum_type":"dummy","checksum":"dummy"}]}]}`

	var cata Catalog
	err := json.Unmarshal([]byte(jstring), &cata)
	if err != nil {
		t.Fatal(fmt.Sprintf("Error unmarshalling JSON: %s", err))
	}
	if cata.Name != "examplebox" {
		t.Fatal(fmt.Sprintf("Decoded JSON had bad Name property; should be 'examplebox' but was '%s'", cata.Name))
	}
	if len(cata.Versions) != 1 {
		t.Fatal(fmt.Sprintf("Expected decoded JSON to have %v elements in its Versions property, but actually had %v", 1, len(cata.Versions)))
	}
	vers := cata.Versions[0]
	if vers.Version != "12.34.56" {
		t.Fatal(fmt.Sprintf("Expected decoded JSON to have a Version with a version of '%s', but actually had a version of '%s'", "12.34.56", vers.Version))
	}
	if len(vers.Providers) != 1 {
		t.Fatal(fmt.Sprintf("Expected first Version to have %v elements in its Providers property, but actually had %v", 1, len(vers.Providers)))
	}
	prov := vers.Providers[0]
	if prov.Name != "testname" {
		t.Fatal(fmt.Sprintf("Expected first Provider to have a Name of '%s', but actually had '%s'", "testname", prov.Name))
	}
}

func TestJsonDecodingEmptyCatalog(t *testing.T) {
	var cata Catalog
	err := json.Unmarshal([]byte("{}"), &cata)
	if err != nil {
		t.Fatal("Failed to unmarshal empty catalog with error:", err)
	}
}

func TestAddBoxToCatalog(t *testing.T) {
	addBoxSrcPath := "/packer/output/packer-TESTBOX-PROVIDER.box"
	addBoxName := "TESTBOX"
	addBoxDesc := "This is a description of TESTBOX"
	addBoxVers := "2.4.9"
	addBoxProv := "PROVIDER"
	addBoxCataRoot := "/catalog/root"
	addBoxExpectedUrl := fmt.Sprintf("file://%v/%v/%v_%v_%v.box", addBoxCataRoot, addBoxName, addBoxName, addBoxVers, addBoxProv)
	addBoxCheckType := "CHECKSUMTYPE"
	addBoxChecksum := "0xDECAFBAD"

	bxArt := BoxArtifact{addBoxSrcPath, addBoxName, addBoxDesc, addBoxVers, addBoxProv, addBoxCataRoot, addBoxCheckType, addBoxChecksum}

	var resultCata, expectedCata Catalog

	resultCata = AddBoxToCatalog(Catalog{}, bxArt)
	expectedCata = Catalog{addBoxName, addBoxDesc, []Version{
		Version{addBoxVers, []Provider{
			Provider{addBoxProv, addBoxExpectedUrl, addBoxCheckType, addBoxChecksum}}}}}
	if !resultCata.Equals(expectedCata) {
		t.Fatal(fmt.Sprintf("Result catalog did not match expected catalog\n\t%v\n\t%v", resultCata, expectedCata))
	}

	/*	Test to do:
		Empty catalog
		Catalog with empty Versions
		Catalog with other Versions but not the one you're adding
		Catalog with populated Versions, with the version you're adding, but empty Providers
		Catalog with populated Versions, with the version you're adding, and other Providers but not the one you're adding
		Catalog with populated Versions, with the version you're adding, and a Provider that you're adding
		Catalog with conflicting name/description
	*/

}

func TestDetermineProvider(t *testing.T) {
	inOutPairs := map[string]string{
		"/omg/wtf/bbq/packer_BUILDNAME_PROVIDER.box":     "PROVIDER",
		"/omg/wtf/bbq/packer_MY_BUILD_NAME_PROVIDER.box": "PROVIDER",
		"file:///C:/packer_BUILDNAME_PROVIDER.box":       "PROVIDER",
	}

	for input, expectedOutput := range inOutPairs {
		realOutput, _ := determineProvider(input)
		if realOutput != expectedOutput {
			t.Fatal(fmt.Sprintf("For input '%v', expected output to be '%v' but was actually '%v'", input, expectedOutput, realOutput))
		}
	}
}
