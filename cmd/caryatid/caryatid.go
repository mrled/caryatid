/*
Caryatid standalone program

A command line application for managing Vagrant catalogs

caryatid add --uri uri:///path/to/catalog.json --name "testbox" --box /local/path/to/name.box --version 1.2.5
caryatid query --uri uri:///path/to/catalog.json --version ">=1.2.5" --provider "-iso" --name "asdf"
caryatid delete --uri uri:///path/to/catalog.json --version "<1.0.0" --provider "-iso" --name "asdf"
*/

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/mrled/caryatid/pkg/caryatid"
)

var (
	cFlag = flag.NewFlagSet("Caryatid", flag.PanicOnError)

	actionFlag      string
	catalogFlag     string
	boxFlag         string
	versionFlag     string
	descriptionFlag string
	providerFlag    string
	nameFlag        string
)

func init() {
	cFlag.Usage = func() {
		// What the fuck, people https://github.com/golang/go/issues/16955
		fmt.Printf("Caryatid usage:\n")
		cFlag.PrintDefaults()
	}

	cFlag.StringVar(
		&actionFlag, "action", "",
		"One of 'show', 'create-test-box', 'query', 'add', or 'delete'.")
	cFlag.StringVar(
		&catalogFlag, "catalog", "",
		"URI for the Vagrant Catalog to operate on")
	cFlag.StringVar(
		&boxFlag, "box", "", "Local path to a box file")
	cFlag.StringVar(
		&versionFlag, "version", "",
		"A version specifier. When querying boxes or deleting a box, this restricts the query to only the versions matched, and its value may include specifiers such as less-than signs, like '<=1.2.3'. When adding a box, the version must be exact, and such specifiers are not supported.")
	cFlag.StringVar(
		&descriptionFlag, "description", "",
		"A description for a box in the Vagrant catalog")
	cFlag.StringVar(
		&providerFlag, "provider", "",
		"The name of a provider. When querying boxes or deleting a box, this restricts the query to only the providers matched, and its value may include asterisks to glob such as '*-iso'. When adding a box, globbing is not supported and an asterisk will be interpreted literally.")
	cFlag.StringVar(
		&nameFlag, "name", "",
		"The name of the box tracked in the Vagrant catalog. When deleting a box, this restricts the query to only boxes matching this name, and may include asterisks for globbing. When adding a box, globbing is not supported and an asterisk will be interpreted literally.")
}

func main() {
	var (
		err    error
		result string
	)

	if err = cFlag.Parse(os.Args[1:]); err != nil {
		panic(fmt.Sprintf("Flag parsing error: %v\n", err))
	}

	switch actionFlag {
	case "show":
		result, err = showAction(catalogFlag, boxFlag)
		fmt.Printf("%v\n", result)
	case "create-test-box":
		err = createTestBoxAction(boxFlag, providerFlag)
	case "add":
		// TODO: Validate -version when adding a box
		// (Should also be done in the packer post-processor, I guess)
		err = addAction(boxFlag, nameFlag, descriptionFlag, versionFlag, catalogFlag)
	case "query":
		var resultCata caryatid.Catalog
		resultCata, err = queryAction(catalogFlag, nameFlag, versionFlag, providerFlag)
		fmt.Printf(resultCata.DisplayString())
	case "delete":
		err = deleteAction(catalogFlag, nameFlag, versionFlag, providerFlag)
	default:
		fmt.Printf("Unknown action: %v\n", actionFlag)
		cFlag.Usage()
	}

	if err != nil {
		fmt.Printf("Error running '%v' action:\n%v\n", actionFlag, err)
		os.Exit(1)
	}

	os.Exit(0)
}
