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
	if matched && err != nil {
		return true
	}
	return false
}

func convertLocalPathToUri(path string) (uri string, err error) {
	abspath, err := filepath.Abs(path)
	uri = fmt.Sprintf("file://%v", abspath)
	return
}

// func ensure

func main() {

	// Flags with default arguments
	actionFlag := flag.String(
		"action",
		"show",
		"One of 'show', 'query', 'add', or 'delete'.")

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

	globalRequiredFlags := []string{
		"catalog",
	}
	createTestBoxRequiredFlags := []string{
		"box",
		"provider",
	}
	showRequiredFlags := []string{}
	queryRequiredFlags := []string{}
	addRequiredFlags := []string{
		"box",
		"description",
		"version",
		"name",
	}
	deleteRequiredFlags := []string{
		"box",
		"version",
		"provider",
	}

	var err error

	// Create an array of all flags passed by the user
	// Note that this will not include flags with default values
	passedFlags := make([]string, 0)
	flag.Visit(func(f *flag.Flag) { passedFlags = append(passedFlags, f.Name) })
	// fmt.Printf("Passed flags: %v\n", passedFlags)

	strEnsureArrayContainsAll(passedFlags, globalRequiredFlags, "Missing required flag: '-%v'")

	// Handle a special case where the -catalog is a local path, rather than a file:// URI
	var catalogUri string
	if testValidUri(*catalogFlag) {
		catalogUri = *catalogFlag
	} else {
		catalogUri, err = convertLocalPathToUri(*catalogFlag)
		if err != nil {
			log.Printf("Error converting catalog path '%v' to URI: %v", *catalogFlag, err)
			os.Exit(1)
		}
	}
	log.Printf("Using catalog URI of '%v'", catalogUri)

	backend, err := caryatid.NewBackendFromUri(catalogUri)
	if err != nil {
		log.Printf("Error retrieving backend: %v\n", err)
		os.Exit(1)
	}

	manager := caryatid.NewBackendManager(catalogUri, *nameFlag, &backend)
	cata, err := manager.GetCatalog()
	if err != nil {
		log.Printf("Error getting catalog: %v\n", err)
		os.Exit(1)
	}

	switch *actionFlag {

	case "show":
		strEnsureArrayContainsAll(passedFlags, showRequiredFlags, "Missing required flag for '-action show': '-%v'")
		fmt.Printf("%v\n", cata)

	case "create-test-box":
		strEnsureArrayContainsAll(passedFlags, createTestBoxRequiredFlags, "Missing required flag for '-action create-test-box': '-%v'")
		caryatid.CreateTestBoxFile(*boxFlag, *providerFlag, true)
		log.Printf("Box file created at '%v'", *boxFlag)

	case "add":
		// TODO: Reduce code duplication between here and packer-post-processor-caryatid

		strEnsureArrayContainsAll(passedFlags, addRequiredFlags, "Missing required flag for '-action add': '-%v'")

		digestType, digest, provider, err := caryatid.DeriveArtifactInfoFromBoxFile(*boxFlag)
		if err != nil {
			panic(fmt.Sprintf("Could not determine artifact info: %v", err))
		}

		boxArtifact := caryatid.BoxArtifact{
			*boxFlag,
			*nameFlag,
			*descriptionFlag,
			*versionFlag,
			provider,
			catalogUri,
			digestType,
			digest,
		}

		err = manager.AddBoxMetadataToCatalog(&boxArtifact)
		if err != nil {
			log.Printf("Error adding box metadata to catalog: %v\n", err)
			os.Exit(1)
		}
		log.Println("Catalog saved to backend")

		catalog, err := manager.GetCatalog()
		if err != nil {
			log.Printf("Error getting catalog: %v\n", err)
			os.Exit(1)
		}
		log.Printf("New catalog is:\n%v\n", catalog)

		err = backend.CopyBoxFile(&boxArtifact)
		if err != nil {
			os.Exit(1)
		}
		log.Println("Box file copied successfully to backend")

	case "query":
		strEnsureArrayContainsAll(passedFlags, queryRequiredFlags, "Missing required flag for '-action query': '-%v'")
		catalog, err := manager.GetCatalog()
		if err != nil {
			log.Printf("Error getting catalog: %v\n", err)
			os.Exit(1)
		}
		queryParams := caryatid.CatalogQueryParams{*versionFlag, *providerFlag}
		for _, box := range catalog.QueryCatalog(queryParams) {
			fmt.Printf("%v\n", box.String())
		}

	case "delete":
		strEnsureArrayContainsAll(passedFlags, deleteRequiredFlags, "Missing required flag for '-action delete': '-%v'")
		panic("NOT IMPLEMENTED")

	default:
		panic(fmt.Sprintf("No such action '%v'\n", *actionFlag))
	}

	os.Exit(0)
}
