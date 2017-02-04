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

	var initialCata, resultCata, expectedCata Catalog

	// The next few tests have the same expected result:
	expectedCata = Catalog{addBoxName, addBoxDesc, []Version{
		Version{addBoxVers, []Provider{
			Provider{addBoxProv, addBoxExpectedUrl, addBoxCheckType, addBoxChecksum}}}}}

	// Empty initial catalog
	resultCata = AddBoxToCatalog(Catalog{}, bxArt)
	if !resultCata.Equals(expectedCata) {
		t.Fatal(fmt.Sprintf("Result catalog did not match expected catalog\n\t%v\n\t%v", resultCata, expectedCata))
	}

	// Initial catalog with different name/desc, but otherwise empty
	resultCata = AddBoxToCatalog(Catalog{"different box name", "different box desc", []Version{}}, bxArt)
	if !resultCata.Equals(expectedCata) {
		t.Fatal(fmt.Sprintf("Result catalog did not match expected catalog\n\t%v\n\t%v", resultCata, expectedCata))
	}

	// Initial catalog with same name/desc, but otherwise empty
	resultCata = AddBoxToCatalog(Catalog{addBoxName, addBoxDesc, []Version{}}, bxArt)
	if !resultCata.Equals(expectedCata) {
		t.Fatal(fmt.Sprintf("Result catalog did not match expected catalog\n\t%v\n\t%v", resultCata, expectedCata))
	}

	// Initial catalog with same name/desc and version, but otherwise empty
	resultCata = AddBoxToCatalog(
		Catalog{addBoxName, addBoxDesc, []Version{
			Version{addBoxVers, []Provider{}}}}, bxArt)
	if !resultCata.Equals(expectedCata) {
		t.Fatal(fmt.Sprintf("Result catalog did not match expected catalog\n\t%v\n\t%v", resultCata, expectedCata))
	}

	// Initial catalog with same name/desc and version and provider
	resultCata = AddBoxToCatalog(
		Catalog{addBoxName, addBoxDesc, []Version{
			Version{addBoxVers, []Provider{
				Provider{addBoxProv, addBoxExpectedUrl, addBoxCheckType, addBoxChecksum}}}}}, bxArt)
	if !resultCata.Equals(expectedCata) {
		t.Fatal(fmt.Sprintf("Result catalog did not match expected catalog\n\t%v\n\t%v", resultCata, expectedCata))
	}

	// Initial catalog with same name/desc and version and same provider with different url/checktype/checksum
	resultCata = AddBoxToCatalog(
		Catalog{addBoxName, addBoxDesc, []Version{
			Version{addBoxVers, []Provider{
				Provider{addBoxProv, "different box URL", "different checksum type", "different checksum"}}}}}, bxArt)
	if !resultCata.Equals(expectedCata) {
		t.Fatal(fmt.Sprintf("Result catalog did not match expected catalog\n\t%v\n\t%v", resultCata, expectedCata))
	}

	// The following set of tests will have a different expected catalog

	// Initial catalog with same name/desc and version, and 2 existing different providers
	initialCata = Catalog{addBoxName, addBoxDesc, []Version{
		Version{addBoxVers, []Provider{
			Provider{"anProvider", "anUrl", "anCheckType", "anCheckSum"},
			Provider{"otherProvider", "otherUrl", "otherCheckType", "otherCheckSum"}}}}}
	expectedCata = Catalog{addBoxName, addBoxDesc, []Version{
		Version{addBoxVers, []Provider{
			Provider{"anProvider", "anUrl", "anCheckType", "anCheckSum"},
			Provider{"otherProvider", "otherUrl", "otherCheckType", "otherCheckSum"},
			Provider{addBoxProv, addBoxExpectedUrl, addBoxCheckType, addBoxChecksum}}}}}
	resultCata = AddBoxToCatalog(initialCata, bxArt)
	if !resultCata.Equals(expectedCata) {
		t.Fatal(fmt.Sprintf("Result catalog did not match expected catalog\n\t%v\n\t%v", resultCata, expectedCata))
	}

	// Initial catalog with same name/desc and 2 different versions
	initialCata = Catalog{addBoxName, addBoxDesc, []Version{
		Version{"1.2.3", []Provider{Provider{addBoxProv, addBoxExpectedUrl, addBoxCheckType, addBoxChecksum}}},
		Version{"5.2.4", []Provider{Provider{addBoxProv, addBoxExpectedUrl, addBoxCheckType, addBoxChecksum}}}}}
	expectedCata = Catalog{addBoxName, addBoxDesc, []Version{
		Version{"1.2.3", []Provider{Provider{addBoxProv, addBoxExpectedUrl, addBoxCheckType, addBoxChecksum}}},
		Version{"5.2.4", []Provider{Provider{addBoxProv, addBoxExpectedUrl, addBoxCheckType, addBoxChecksum}}},
		Version{addBoxVers, []Provider{Provider{addBoxProv, addBoxExpectedUrl, addBoxCheckType, addBoxChecksum}}}}}
	resultCata = AddBoxToCatalog(initialCata, bxArt)
	if !resultCata.Equals(expectedCata) {
		t.Fatal(fmt.Sprintf("Result catalog did not match expected catalog\n\t%v\n\t%v", resultCata, expectedCata))
	}
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
