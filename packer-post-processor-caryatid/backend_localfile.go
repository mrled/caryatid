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
	VagrantCatalogPath string
	Manager            *BackendManager
}

func (clfb *CaryatidLocalFileBackend) SetManager(manager *BackendManager) (err error) {
	clfb.Manager = manager

	catalogRootPath, err := parseLocalPathFromUri(clfb.Manager.VagrantCatalogRootUri)
	if err != nil {
		fmt.Printf("Error trying to parse local catalog path from URI: %v\n", err)
		return
	}
	catalogFilename := fmt.Sprintf("%v.json", clfb.Manager.VagrantCatalogName)
	clfb.VagrantCatalogPath = path.Join(catalogRootPath, catalogFilename)

	return
}

func (cb *CaryatidLocalFileBackend) GetManager() (manager *BackendManager, err error) {
	manager = cb.Manager
	if manager == nil {
		err = fmt.Errorf("The Manager property was not set")
	}
	return
}

func (clfb *CaryatidLocalFileBackend) GetCatalogBytes() (catalogBytes []byte, err error) {
	catalogBytes, err = ioutil.ReadFile(clfb.VagrantCatalogPath)
	if os.IsNotExist(err) {
		log.Printf("No file at '%v'; starting with empty catalog\n", clfb.VagrantCatalogPath)
		catalogBytes = []byte("{}")
		err = nil
	} else if err != nil {
		log.Printf("Error trying to read catalog: %v\n", err)
	}
	return
}

func (clfb *CaryatidLocalFileBackend) SetCatalogBytes(serializedCatalog []byte) (err error) {
	err = ioutil.WriteFile(clfb.VagrantCatalogPath, serializedCatalog, 0666)
	if err != nil {
		log.Println("Error trying to write catalog: ", err)
		return
	}
	log.Println(fmt.Sprintf("Catalog updated on disk to reflect new value"))
	return
}

func (clfb *CaryatidLocalFileBackend) CopyBoxFile(box *BoxArtifact) (err error) {

	remoteBoxPath, err := parseLocalPathFromUri(box.GetUri())
	if err != nil {
		fmt.Printf("Error trying to parse local artifact path from URI: %v\n", err)
		return
	}

	remoteBoxParentPath, err := parseLocalPathFromUri(box.GetParentUri())
	if err != nil {
		fmt.Printf("Error trying to parse local artifact parent path from URI: %v\n", err)
		return
	}

	err = os.MkdirAll(remoteBoxParentPath, 0777)
	if err != nil {
		log.Println("Error trying to create the box directory: ", err)
		return
	}

	written, err := util.CopyFile(box.Path, remoteBoxPath)
	if err != nil {
		log.Println(fmt.Sprintf("Error trying to copy '%v' to '%v' file: %v", box.Path, remoteBoxPath, err))
		return
	}
	log.Println(fmt.Sprintf("Copied %v bytes from original path at '%v' to new location at '%v'", written, box.Path, remoteBoxPath))
	return
}
