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

func getManager(catalogUri string, backendCredential string) (manager *caryatid.BackendManager, err error) {
	var uri string
	if testValidUri(catalogUri) {
		uri = catalogUri
	} else {
		// Handle a special case where the -catalog is a local path, rather than a file:// URI
		uri, err = convertLocalPathToUri(catalogUri)
		if err != nil {
			log.Printf("Error converting catalog path '%v' to URI: %v", catalogUri, err)
			return
		}
	}
	log.Printf("Using catalog URI of '%v'", uri)

	manager, err = caryatid.NewBackendManager(uri, backendCredential)
	if err != nil {
		log.Printf("Error creating backend manager: %v\n", err)
		return
	}

	return
}

func showAction(catalogUri string, backendCredential string) (result string, err error) {
	manager, err := getManager(catalogUri)
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

func addAction(boxPath string, boxName string, boxDescription string, boxVersion string, catalogUri string, backendCredential string) (err error) {
	// TODO: Reduce code duplication between here and packer-post-processor-caryatid
	digestType, digest, provider, err := caryatid.DeriveArtifactInfoFromBoxFile(boxPath)
	if err != nil {
		panic(fmt.Sprintf("Could not determine artifact info: %v", err))
	}

	manager, err := getManager(catalogUri)
	if err != nil {
		log.Printf("Error getting a BackendManager")
		return
	}

	err = manager.AddBox(boxPath, boxName, boxDescription, boxVersion, provider, digestType, digest)
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

func queryAction(catalogUri string, backendCredential string, versionQuery string, providerQuery string) (result caryatid.Catalog, err error) {
	manager, err := getManager(catalogUri)
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

func deleteAction(catalogUri string, backendCredential string, versionQuery string, providerQuery string) (err error) {
	manager, err := getManager(catalogUri)
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
