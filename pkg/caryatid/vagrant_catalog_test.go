package caryatid

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

func TestProviderEquals(t *testing.T) {
	matchingp1 := Provider{"TestProviderX", "http://example.com/pX", "TestChecksum", "0xB00B135"}
	matchingp2 := Provider{"TestProviderX", "http://example.com/pX", "TestChecksum", "0xB00B135"}
	unmatchingp := []Provider{
		Provider{"TestProviderYaaaas", "http://example.com/pX", "TestChecksum", "0xB00B135"},
		Provider{"TestProviderX", "http://example.com/pother", "TestChecksum", "0xB00B135"},
		Provider{"TestProviderX", "http://example.com/pX", "DifferentChecksum", "0xB00B135"},
		Provider{"TestProviderX", "http://example.com/pX", "TestChecksum", "0xDECAFBADxxxxx"},
	}
	if !matchingp1.Equals(&matchingp2) {
		t.Fatal("Providers that should have matched do not match")
	}
	for idx := 0; idx < len(unmatchingp); idx += 1 {
		if matchingp1.Equals(&unmatchingp[idx]) {
			t.Fatal(fmt.Sprintf("Providers that should not have matched did match. Failing non-matching provider: %v", unmatchingp[idx]))
		}
	}
}

func TestVersionEquals(t *testing.T) {
	p1 := Provider{"TestProviderOne", "http://example.com/One", "TestChecksum", "0xB00B135"}
	p2 := Provider{"TestProviderTwo", "http://example.com/Two", "TestChecksum", "0xB00B135"}

	matchingv1 := Version{"1.2.3", []Provider{p1, p2}}
	matchingv2 := Version{"1.2.3", []Provider{p1, p2}}
	unmatchingv := []Version{
		Version{"1.2.3", []Provider{p2}},
		Version{"1.2.4", []Provider{p1}},
		Version{"1.2.3", []Provider{p1, p2, p2}},
	}
	if !matchingv1.Equals(&matchingv2) {
		t.Fatal("Versions that should have matched did not match")
	}
	for idx := 0; idx < len(unmatchingv); idx += 1 {
		if matchingv1.Equals(&unmatchingv[idx]) {
			t.Fatal(fmt.Sprintf("Versions that should not have matched did match. Failing non-matching version: %v", unmatchingv[idx]))
		}
	}
}

func TestCatalogEquals(t *testing.T) {
	p1 := Provider{"TestProvider", "http://example.com/Provider", "TestChecksum", "0xB00B135"}
	v1 := Version{"1.2.3", []Provider{p1}}
	v2 := Version{"1.2.4", []Provider{p1}}
	matchingc1 := Catalog{"SomeName", "This is a desc", []Version{v1, v2}}
	matchingc2 := Catalog{"SomeName", "This is a desc", []Version{v1, v2}}
	unmatchingc := []Catalog{
		Catalog{"SomeOtherName", "This is a desc", []Version{v1, v2}},
		Catalog{"SomeName", "This is a completely different desc", []Version{v1, v2}},
		Catalog{"SomeName", "This is a desc", []Version{v1}},
		Catalog{"SomeName", "This is a desc", []Version{v1, v2, v2}},
		Catalog{"SomeName", "This is a desc", []Version{v2, v1}},
	}

	if !matchingc1.Equals(&matchingc2) {
		t.Fatal("Catalogs that should have matched did not match")
	}
	for idx := 0; idx < len(unmatchingc); idx += 1 {
		if matchingc1.Equals(&unmatchingc[idx]) {
			t.Fatal(fmt.Sprintf("Catalogs that should not have matched did match. Failing non-matching version: %v", unmatchingc[idx]))
		}
	}
}

func TestCatalogAddBox(t *testing.T) {
	addBoxSrcPath := "/packer/output/packer-TESTBOX-PROVIDER.box"
	addBoxName := "TESTBOX"
	addBoxDesc := "This is a description of TESTBOX"
	addBoxVers := "2.4.9"
	addBoxProv := "PROVIDER"
	addBoxCataRoot := "file:///catalog/root"
	addBoxExpectedUrl := fmt.Sprintf("%v/%v/%v_%v_%v.box", addBoxCataRoot, addBoxName, addBoxName, addBoxVers, addBoxProv)
	addBoxCheckType := "CHECKSUMTYPE"
	addBoxChecksum := "0xDECAFBAD"

	bxArt := BoxArtifact{addBoxSrcPath, addBoxName, addBoxDesc, addBoxVers, addBoxProv, addBoxCataRoot, addBoxCheckType, addBoxChecksum}

	addAndCompareCata := func(description string, initial *Catalog, expected *Catalog, addition *BoxArtifact) {
		if err := initial.AddBox(addition); err != nil {
			t.Fatal(fmt.Sprintf("Error calling AddBox in test '%v': %v", description, err))
		}
		if !initial.Equals(expected) {
			t.Fatal(fmt.Sprintf("Test '%v' failed\nInitial catalog:\n%v\nExpected catalog:\n%v\n", description, initial, expected))
		}
	}

	addAndCompareCata(
		"Add box to empty catalog",
		&Catalog{},
		&Catalog{addBoxName, addBoxDesc, []Version{
			Version{addBoxVers, []Provider{
				Provider{addBoxProv, addBoxExpectedUrl, addBoxCheckType, addBoxChecksum},
			}},
		}},
		&bxArt,
	)

	addAndCompareCata(
		"Add box to catalog where it's already present",
		&Catalog{addBoxName, addBoxDesc, []Version{
			Version{addBoxVers, []Provider{
				Provider{addBoxProv, addBoxExpectedUrl, addBoxCheckType, addBoxChecksum},
			}},
		}},
		&Catalog{addBoxName, addBoxDesc, []Version{
			Version{addBoxVers, []Provider{
				Provider{addBoxProv, addBoxExpectedUrl, addBoxCheckType, addBoxChecksum},
			}},
		}},
		&bxArt,
	)

	addAndCompareCata(
		"Add box to catalog with empty version",
		&Catalog{addBoxName, addBoxDesc, []Version{}},
		&Catalog{addBoxName, addBoxDesc, []Version{
			Version{addBoxVers, []Provider{
				Provider{addBoxProv, addBoxExpectedUrl, addBoxCheckType, addBoxChecksum},
			}},
		}},
		&bxArt,
	)

	addAndCompareCata(
		"Add box to catalog with different version",
		&Catalog{addBoxName, addBoxDesc, []Version{
			Version{"2.3.0", []Provider{
				Provider{addBoxProv, addBoxExpectedUrl, addBoxCheckType, addBoxChecksum},
			}},
		}},
		&Catalog{addBoxName, addBoxDesc, []Version{
			Version{"2.3.0", []Provider{
				Provider{addBoxProv, addBoxExpectedUrl, addBoxCheckType, addBoxChecksum},
			}},
			Version{addBoxVers, []Provider{
				Provider{addBoxProv, addBoxExpectedUrl, addBoxCheckType, addBoxChecksum},
			}},
		}},
		&bxArt,
	)

	addAndCompareCata(
		"Add box to catalog with empty provider",
		&Catalog{addBoxName, addBoxDesc, []Version{
			Version{addBoxVers, []Provider{
				Provider{},
			}},
		}},
		&Catalog{addBoxName, addBoxDesc, []Version{
			Version{addBoxVers, []Provider{
				Provider{},
				Provider{addBoxProv, addBoxExpectedUrl, addBoxCheckType, addBoxChecksum},
			}},
		}},
		&bxArt,
	)

	addAndCompareCata(
		"Add box to catalog with different provider",
		&Catalog{addBoxName, addBoxDesc, []Version{
			Version{addBoxVers, []Provider{
				Provider{"differentProvider", addBoxExpectedUrl, addBoxCheckType, addBoxChecksum},
			}},
		}},
		&Catalog{addBoxName, addBoxDesc, []Version{
			Version{addBoxVers, []Provider{
				Provider{"differentProvider", addBoxExpectedUrl, addBoxCheckType, addBoxChecksum},
				Provider{addBoxProv, addBoxExpectedUrl, addBoxCheckType, addBoxChecksum},
			}},
		}},
		&bxArt,
	)
}

type TestParameters struct {
	ProviderNames []string
	BoxUri        string
	BoxName       string
	BoxDesc       string
	DigestType    string
	Digest        string
}

var testParameters = TestParameters{
	[]string{"StrongSapling", "FeebleFungus"},
	"http://example.com/this/is/my/box",
	"vagrant_catalog_test_box",
	"Vagrant Catalog Test Box is a test box",
	"CRC32",
	"0xB00B1E5",
}

var testCatalog = Catalog{testParameters.BoxName, testParameters.BoxDesc, []Version{
	Version{"0.3.5", []Provider{
		Provider{testParameters.ProviderNames[0], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
	}},
	Version{"0.3.4", []Provider{
		Provider{testParameters.ProviderNames[1], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
	}},
	Version{"0.3.5-BETA", []Provider{
		Provider{testParameters.ProviderNames[0], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
		Provider{testParameters.ProviderNames[1], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
	}},
	Version{"1.0.0", []Provider{
		Provider{testParameters.ProviderNames[0], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
	}},
	Version{"1.0.1", []Provider{
		Provider{testParameters.ProviderNames[1], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
	}},
	Version{"1.4.5", []Provider{
		Provider{testParameters.ProviderNames[0], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
	}},
	Version{"1.2.3", []Provider{
		Provider{testParameters.ProviderNames[0], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
		Provider{testParameters.ProviderNames[1], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
	}},
	Version{"1.2.4", []Provider{
		Provider{testParameters.ProviderNames[0], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
	}},
	Version{"2.11.1", []Provider{
		Provider{testParameters.ProviderNames[1], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
	}},
}}

func TestQueryCatalogVersions(t *testing.T) {
	testQueryVers := func(initial *Catalog, query string, expectedResult *Catalog) {
		result, err := initial.QueryCatalogVersions(query)
		if err != nil {
			t.Fatalf("QueryCatalogVersions() returned an error: %v\n", err)
		} else if !expectedResult.Equals(&result) {
			t.Fatalf("QueryCatalogVersions() returned unexpected value(s). Actual:\n%v\nExpected:\n%v\n", result, expectedResult)
		}
	}

	testQueryVers(&testCatalog, ">2", &Catalog{testParameters.BoxName, testParameters.BoxDesc, []Version{
		Version{"2.11.1", []Provider{
			Provider{testParameters.ProviderNames[1], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
		}},
	}})
	testQueryVers(&testCatalog, "<=0.3.5", &Catalog{testParameters.BoxName, testParameters.BoxDesc, []Version{
		Version{"0.3.5", []Provider{
			Provider{testParameters.ProviderNames[0], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
		}},
		Version{"0.3.4", []Provider{
			Provider{testParameters.ProviderNames[1], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
		}},
		Version{"0.3.5-BETA", []Provider{
			Provider{testParameters.ProviderNames[0], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
			Provider{testParameters.ProviderNames[1], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
		}},
	}})
	testQueryVers(&testCatalog, "0.3.5", &Catalog{testParameters.BoxName, testParameters.BoxDesc, []Version{
		Version{"0.3.5", []Provider{
			Provider{testParameters.ProviderNames[0], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
		}},
		Version{"0.3.5-BETA", []Provider{
			Provider{testParameters.ProviderNames[0], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
			Provider{testParameters.ProviderNames[1], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
		}},
	}})
	testQueryVers(&testCatalog, "=0.3.5", &Catalog{testParameters.BoxName, testParameters.BoxDesc, []Version{
		Version{"0.3.5", []Provider{
			Provider{testParameters.ProviderNames[0], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
		}},
	}})
	testQueryVers(&testCatalog, "=0.3.6", &Catalog{testParameters.BoxName, testParameters.BoxDesc, []Version{}})
}

func TestQueryCatalogProviders(t *testing.T) {
	testQueryProv := func(initial Catalog, query string, expectedResult Catalog) {
		result, err := initial.QueryCatalogProviders(query)
		if err != nil {
			t.Fatalf("QueryCatalogProviders() returned an error: %v\n", err)
		} else if !expectedResult.Equals(&result) {
			t.Fatalf("QueryCatalogProviders() returned unexpected value(s). Actual:\n%v\nExpected:\n%v\n", result, expectedResult)
		}
	}
	testQueryProv(testCatalog, "^Strong", Catalog{testParameters.BoxName, testParameters.BoxDesc, []Version{
		Version{"0.3.5", []Provider{
			Provider{testParameters.ProviderNames[0], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
		}},
		Version{"0.3.5-BETA", []Provider{
			Provider{testParameters.ProviderNames[0], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
		}},
		Version{"1.0.0", []Provider{
			Provider{testParameters.ProviderNames[0], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
		}},
		Version{"1.4.5", []Provider{
			Provider{testParameters.ProviderNames[0], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
		}},
		Version{"1.2.3", []Provider{
			Provider{testParameters.ProviderNames[0], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
		}},
		Version{"1.2.4", []Provider{
			Provider{testParameters.ProviderNames[0], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
		}},
	}})
	testQueryProv(testCatalog, "Sapling$", Catalog{testParameters.BoxName, testParameters.BoxDesc, []Version{
		Version{"0.3.5", []Provider{
			Provider{testParameters.ProviderNames[0], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
		}},
		Version{"0.3.5-BETA", []Provider{
			Provider{testParameters.ProviderNames[0], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
		}},
		Version{"1.0.0", []Provider{
			Provider{testParameters.ProviderNames[0], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
		}},
		Version{"1.4.5", []Provider{
			Provider{testParameters.ProviderNames[0], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
		}},
		Version{"1.2.3", []Provider{
			Provider{testParameters.ProviderNames[0], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
		}},
		Version{"1.2.4", []Provider{
			Provider{testParameters.ProviderNames[0], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
		}},
	}})
	testQueryProv(testCatalog, "F", Catalog{testParameters.BoxName, testParameters.BoxDesc, []Version{
		Version{"0.3.4", []Provider{
			Provider{testParameters.ProviderNames[1], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
		}},
		Version{"0.3.5-BETA", []Provider{
			Provider{testParameters.ProviderNames[1], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
		}},
		Version{"1.0.1", []Provider{
			Provider{testParameters.ProviderNames[1], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
		}},
		Version{"1.2.3", []Provider{
			Provider{testParameters.ProviderNames[1], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
		}},
		Version{"2.11.1", []Provider{
			Provider{testParameters.ProviderNames[1], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
		}},
	}})
}

func TestDelete(t *testing.T) {
	testDelete := func(initial Catalog, query CatalogQueryParams, expectedResult Catalog) {
		result, err := initial.Delete(query)
		if err != nil {
			t.Fatalf("Delete(%v) returned an error: %v\n", query, err)
		} else if !expectedResult.Equals(&result) {
			t.Fatalf("Delete(%v) returned unexpected value(s). Actual:\n%v\nExpected:\n%v\n", query, result.DisplayString(), expectedResult.DisplayString())
		}
	}

	testDelete(testCatalog, CatalogQueryParams{Version: "", Provider: ""}, Catalog{
		testParameters.BoxName, testParameters.BoxDesc, []Version{
			Version{"0.3.5", []Provider{
				Provider{testParameters.ProviderNames[0], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
			}},
			Version{"0.3.4", []Provider{
				Provider{testParameters.ProviderNames[1], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
			}},
			Version{"0.3.5-BETA", []Provider{
				Provider{testParameters.ProviderNames[0], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
				Provider{testParameters.ProviderNames[1], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
			}},
			Version{"1.0.0", []Provider{
				Provider{testParameters.ProviderNames[0], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
			}},
			Version{"1.0.1", []Provider{
				Provider{testParameters.ProviderNames[1], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
			}},
			Version{"1.4.5", []Provider{
				Provider{testParameters.ProviderNames[0], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
			}},
			Version{"1.2.3", []Provider{
				Provider{testParameters.ProviderNames[0], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
				Provider{testParameters.ProviderNames[1], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
			}},
			Version{"1.2.4", []Provider{
				Provider{testParameters.ProviderNames[0], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
			}},
			Version{"2.11.1", []Provider{
				Provider{testParameters.ProviderNames[1], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
			}},
		},
	})
	testDelete(testCatalog, CatalogQueryParams{Version: "<=1", Provider: "Feeb"}, Catalog{
		testParameters.BoxName, testParameters.BoxDesc, []Version{
			Version{"1.4.5", []Provider{
				Provider{testParameters.ProviderNames[0], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
			}},
			Version{"1.2.3", []Provider{
				Provider{testParameters.ProviderNames[0], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
			}},
			Version{"1.2.4", []Provider{
				Provider{testParameters.ProviderNames[0], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
			}},
		},
	})
	testDelete(testCatalog, CatalogQueryParams{Version: "<=1.0.0", Provider: "Feeb"}, Catalog{
		testParameters.BoxName, testParameters.BoxDesc, []Version{
			Version{"1.4.5", []Provider{
				Provider{testParameters.ProviderNames[0], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
			}},
			Version{"1.2.3", []Provider{
				Provider{testParameters.ProviderNames[0], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
			}},
			Version{"1.2.4", []Provider{
				Provider{testParameters.ProviderNames[0], testParameters.BoxUri, testParameters.DigestType, testParameters.Digest},
			}},
		},
	})
}
