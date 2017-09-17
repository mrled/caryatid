/*
The localfile backend, for dealing with a Vagrant catalog on a local filesystem
*/

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path"
	"regexp"

	"github.com/mrled/caryatid/packer-post-processor-caryatid/util"
)

// Get a local path like /tmp/asdf or C:\temp\asdf a file:/// URI
func parseLocalPathFromUri(uristring string) (path string, err error) {
	uri, err := url.Parse(uristring)
	if err != nil {
		return
	}

	// On Windows, a URI will sometimes be in the form 'file:///C:\\path\\to\\something'
	// and uri.Path will have a leading slash, like '/C:\\path\\to\\something'.
	// If it does, strip it out
	matched, err := regexp.MatchString("^/[a-zA-Z]:", uri.Path)
	if err != nil {
		log.Printf("regexp.MatchString error: '%v'\n", err)
	} else if matched {
		path = uri.Path[1:len(uri.Path)]
	} else {
		path = uri.Path
	}
	return
}

type CaryatidLocalFileBackend struct {
	VagrantCatalogPath   string
	VagrantBoxPath       string
	VagrantBoxParentPath string
	Manager              VagrantCatalogManager
}

func (clfb CaryatidLocalFileBackend) SetManager(manager VagrantCatalogManager) (err error) {
	clfb.Manager = manager

	catalogRootPath, err := parseLocalPathFromUri(clfb.Manager.VagrantCatalogRootUri)
	if err != nil {
		return
	}
	clfb.VagrantCatalogPath = path.Join(catalogRootPath, fmt.Sprintf("%v.json", clfb.Manager.VagrantBox.Name))

	clfb.VagrantBoxPath, err = parseLocalPathFromUri(clfb.Manager.VagrantBox.GetUri())
	if err != nil {
		return
	}

	clfb.VagrantBoxParentPath, err = parseLocalPathFromUri(clfb.Manager.VagrantBox.GetParentUri())
	if err != nil {
		return
	}

	return
}

func (clfb CaryatidLocalFileBackend) GetCatalogBytes() (catalogBytes []byte, err error) {
	catalogBytes, err = ioutil.ReadFile(clfb.VagrantCatalogPath)
	if os.IsNotExist(err) {
		log.Printf("No file at '%v'; starting with empty catalog\n", clfb.VagrantCatalogPath)
		catalogBytes = []byte("{}")
	} else if err != nil {
		log.Printf("Error trying to read catalog: %v\n", err)
	}
	return
}

func (clfb CaryatidLocalFileBackend) SetCatalogBytes(serializedCatalog []byte) (err error) {
	err = ioutil.WriteFile(clfb.VagrantCatalogPath, serializedCatalog, 0666)
	if err != nil {
		log.Println("Error trying to write catalog: ", err)
		return
	}
	log.Println(fmt.Sprintf("Catalog updated on disk to reflect new value"))
	return
}

func (clfb CaryatidLocalFileBackend) CopyBoxFile(remotePath string) (err error) {
	err = os.MkdirAll(clfb.VagrantBoxParentPath, 0777)
	if err != nil {
		log.Println("Error trying to create the box directory: ", err)
		return
	}

	written, err := util.CopyFile(remotePath, clfb.VagrantBoxPath)
	if err != nil {
		log.Println(fmt.Sprintf("Error trying to copy '%v' to '%v' file: %v", remotePath, clfb.VagrantBoxPath, err))
		return
	}
	log.Println(fmt.Sprintf("Copied %v bytes from original path at '%v' to new location at '%v'", written, remotePath, clfb.VagrantBoxPath))
	return
}
