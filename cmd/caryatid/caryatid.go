/*
Caryatid standalone program

A command line application for managing Vagrant catalogs

caryatid add --uri uri:///path/to/catalog.json --name "testbox" --box /local/path/to/name.box --version 1.2.5
caryatid query --uri uri:///path/to/catalog.json --version ">=1.2.5" --provider "*-iso" --name "*asdf*"
caryatid delete --uri uri:///path/to/catalog.json --version "<1.0.0" --provider "*-iso" --name "*asdf*"
*/

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"github.com/mrled/caryatid/pkg/caryatid"
)

type IoPair struct {
	Input  string
	Output bool
}

func strArrayContains(array []string, testItem string) bool {
	for _, item := range array {
		if item == testItem {
			return true
		}
	}
	return false
}

/* Ensure an array contains all the items of another array. If it doesn't, panic().
refArray: The reference array
mustContain: An array, all items of which refArray must also contain
panicFormatString: A string that can be passed to fmt.Sprintf() which contains exactly one '%v'
*/
func strEnsureArrayContainsAll(refArray []string, mustContain []string, panicFormatString string) {
	for _, mcItem := range mustContain {
		if !strArrayContains(refArray, mcItem) {
			panic(fmt.Sprintf(panicFormatString, mcItem))
		}
	}
}

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

func main() {

	// Flags with default arguments
	actionFlag := flag.String(
		"action",
		"show",
		"One of 'show', 'create-test-box', 'query', 'add', or 'delete'.")

	// Globally required flags
	catalogFlag := flag.String(
		"catalog",
		"",
		"URI for the Vagrant Catalog to operate on")

	boxFlag := flag.String(
		"box", "", "Local path to a box file")

	// TODO: Validate -version when adding a box
	// (Should also be done in the packer post-processor, I guess)
	versionFlag := flag.String(
		"version",
		"",
		"A version specifier. When querying boxes or deleting a box, this restricts the query to only the versions matched, and its value may include specifiers such as less-than signs, like '<=1.2.3'. When adding a box, the version must be exact, and such specifiers are not supported.")
	descriptionFlag := flag.String(
		"description",
		"",
		"A description for a box in the Vagrant catalog")

	providerFlag := flag.String(
		"provider",
		"",
		"The name of a provider. When querying boxes or deleting a box, this restricts the query to only the providers matched, and its value may include asterisks to glob such as '*-iso'. When adding a box, globbing is not supported and an asterisk will be interpreted literally.")

	nameFlag := flag.String(
		"name",
		"",
		"The name of the box tracked in the Vagrant catalog. When deleting a box, this restricts the query to only boxes matching this name, and may include asterisks for globbing. When adding a box, globbing is not supported and an asterisk will be interpreted literally.")
	flag.Parse()

	var (
		err    error
		result string
	)
	switch *actionFlag {
	case "show":
		result, err = showAction(*catalogFlag, *boxFlag)
		fmt.Printf("%v\n", result)
	case "create-test-box":
		err = createTestBoxAction(*boxFlag, *providerFlag)
	case "add":
		err = addAction(*boxFlag, *nameFlag, *descriptionFlag, *versionFlag, *catalogFlag)
	case "query":
		var resultCata caryatid.Catalog
		resultCata, err = queryAction(*catalogFlag, *nameFlag, *versionFlag, *providerFlag)
		fmt.Printf(resultCata.DisplayString())
	case "delete":
		err = deleteAction(*catalogFlag, *nameFlag, *versionFlag, *providerFlag)
	default:
		err = fmt.Errorf("No such action '%v'\n", *actionFlag)
	}

	if err != nil {
		fmt.Printf("Error running '%v' action:\n%v\n", *actionFlag, err)
		os.Exit(1)
	}

	os.Exit(0)
}
