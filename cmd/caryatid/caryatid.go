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
	"os"

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
	backendFlag := flag.String(
		"backend",
		"",
		fmt.Sprintf("The name of the backend to use, of %v", "FIXME"))

	/*boxFlag :=*/ flag.String(
		"box", "", "Local path to a box file")
	/*versionFlag :=*/ flag.String(
		"version", "",
		"A version specifier. When querying boxes or deleting a box, this restricts the query to only the versions matched, and its value may include specifiers such as less-than signs, like '<=1.2.3'. When adding a box, the version must be exact, and such specifiers are not supported.")

	/*providerFlag :=*/ flag.String(
		"provider", "",
		"The name of a provider. When querying boxes or deleting a box, this restricts the query to only the providers matched, and its value may include asterisks to glob such as '*-iso'. When adding a box, globbing is not supported and an asterisk will be interpreted literally.")

	nameFlag := flag.String(
		"name",
		"",
		"The name of the box tracked in the Vagrant catalog. When querying boxes or deleting a box, this restricts the query to only boxes matching this name, and may include asterisks for globbing. When adding a box, globbing is not supported and an asterisk will be interpreted literally.")
	flag.Parse()

	globalRequiredFlags := []string{
		"catalog",
		"backend",
	}
	showRequiredFlags := []string{}
	queryRequiredFlags := []string{}
	addRequiredFlags := []string{
		"box",
		"version",
	}
	deleteRequiredFlags := []string{
		"box",
		"version",
		"provider",
	}

	// Create an array of all flags passed by the user
	// Note that this will not include flags with default values
	passedFlags := make([]string, 0)
	flag.Visit(func(f *flag.Flag) { passedFlags = append(passedFlags, f.Name) })
	// fmt.Printf("Passed flags: %v\n", passedFlags)

	strEnsureArrayContainsAll(passedFlags, globalRequiredFlags, "Missing required flag: '-%v'")

	backend, err := caryatid.NewBackend(*backendFlag)
	if err != nil {
		fmt.Printf("Error retrieving backend '%v': %v\n", *backendFlag, err)
	}
	manager := caryatid.NewBackendManager(*catalogFlag, *nameFlag, &backend)

	switch *actionFlag {
	case "show":
		strEnsureArrayContainsAll(passedFlags, showRequiredFlags, "Missing required flag for '-action show': '-%v'")
		cata, err := manager.GetCatalog()
		if err != nil {
			fmt.Printf("Error getting catalog: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("%v\n", cata)
	case "query":
		strEnsureArrayContainsAll(passedFlags, queryRequiredFlags, "Missing required flag for '-action query': '-%v'")
		panic("NOT IMPLEMENTED")
	case "add":
		strEnsureArrayContainsAll(passedFlags, addRequiredFlags, "Missing required flag for '-action add': '-%v'")
		panic("NOT IMPLEMENTED")
	case "delete":
		strEnsureArrayContainsAll(passedFlags, deleteRequiredFlags, "Missing required flag for '-action delete': '-%v'")
		panic("NOT IMPLEMENTED")
	default:
		panic(fmt.Sprintf("No such action '%v'\n", *actionFlag))
	}

	os.Exit(0)
}
