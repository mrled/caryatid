package main

import (
	"fmt"
	"log"
	"path/filepath"
	"regexp"

	"github.com/mrled/caryatid/pkg/caryatid"
)

// Test whether a string is a valid URI
func testValidUri(uri string) bool {
	matched, err := regexp.MatchString("^[a-zA-Z0-9]+://", uri)
	if err != nil {
		matched = false
	}
	return matched
}

func convertLocalPathToUri(path string) (uri string, err error) {
	abspath, err := filepath.Abs(path)
	uri = fmt.Sprintf("file://%v", abspath)
	return
}

func getManager(catalogRootUri string, boxName string) (manager *caryatid.BackendManager, err error) {
	var uri string
	if testValidUri(catalogRootUri) {
		uri = catalogRootUri
	} else {
		// Handle a special case where the -catalog is a local path, rather than a file:// URI
		uri, err = convertLocalPathToUri(catalogRootUri)
		if err != nil {
			log.Printf("Error converting catalog path '%v' to URI: %v", catalogRootUri, err)
			return
		}
	}
	log.Printf("Using catalog URI of '%v'", uri)

	backend, err := caryatid.NewBackendFromUri(uri)
	if err != nil {
		log.Printf("Error retrieving backend: %v\n", err)
		return
	}

	manager = caryatid.NewBackendManager(uri, boxName, &backend)
	return
}

func showAction(catalogRootUri string, boxName string) (result string, err error) {
	manager, err := getManager(catalogRootUri, boxName)
	if err != nil {
		return "", err
	}
	catalog, err := manager.GetCatalog()
	if err != nil {
		return "", err
	}
	result = fmt.Sprintf("%v\n", catalog)
	return
}

func createTestBoxAction(boxName string, providerName string) (err error) {
	err = caryatid.CreateTestBoxFile(boxName, providerName, true)
	if err != nil {
		log.Printf("Error creating a test box file: %v", err)
		return
	} else {
		log.Printf("Box file created at '%v'", boxName)
	}
	return
}

func addAction(boxPath string, boxName string, boxDescription string, boxVersion string, catalogRootUri string) (err error) {
	// TODO: Reduce code duplication between here and packer-post-processor-caryatid
	digestType, digest, provider, err := caryatid.DeriveArtifactInfoFromBoxFile(boxPath)
	if err != nil {
		panic(fmt.Sprintf("Could not determine artifact info: %v", err))
	}

	boxArtifact := caryatid.BoxArtifact{
		boxPath,
		boxName,
		boxDescription,
		boxVersion,
		provider,
		catalogRootUri,
		digestType,
		digest,
	}

	manager, err := getManager(catalogRootUri, boxName)
	if err != nil {
		log.Printf("Error getting a BackendManager")
		return
	}

	err = manager.AddBox(&boxArtifact)
	if err != nil {
		log.Printf("Error adding box metadata to catalog: %v\n", err)
		return
	}
	log.Println("Box successfully added to backend")

	catalog, err := manager.GetCatalog()
	if err != nil {
		log.Printf("Error getting catalog: %v\n", err)
		return
	}
	log.Printf("New catalog is:\n%v\n", catalog)

	return
}

func queryAction(catalogRootUri string, boxName string, versionQuery string, providerQuery string) (result caryatid.Catalog, err error) {
	manager, err := getManager(catalogRootUri, boxName)
	if err != nil {
		log.Printf("Error getting a BackendManager")
		return
	}

	catalog, err := manager.GetCatalog()
	if err != nil {
		log.Printf("Error getting catalog: %v\n", err)
		return
	}

	queryParams := caryatid.CatalogQueryParams{Version: versionQuery, Provider: providerQuery}
	result, err = catalog.QueryCatalog(queryParams)
	if err != nil {
		log.Printf("Error querying catalog: %v\n", err)
		return
	}

	return
}

func deleteAction(catalogRootUri string, boxName string, versionQuery string, providerQuery string) (err error) {
	manager, err := getManager(catalogRootUri, boxName)
	if err != nil {
		log.Printf("Error getting a BackendManager")
		return
	}

	queryParams := caryatid.CatalogQueryParams{Version: versionQuery, Provider: providerQuery}
	if err = manager.DeleteBox(queryParams); err != nil {
		return
	}

	return
}
