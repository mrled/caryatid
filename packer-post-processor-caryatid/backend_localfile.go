/*
The localfile backend, for dealing with a Vagrant catalog on a local filesystem
*/

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/mrled/caryatid/packer-post-processor-caryatid/util"
)

type CaryatidLocalFileBackend struct {
	VagrantCatalogRootPath string
	VagrantCatalogPath     string
	Manager                *BackendManager
}

func (backend *CaryatidLocalFileBackend) SetManager(manager *BackendManager) (err error) {
	backend.Manager = manager

	catalogRootPath, err := util.ParseLocalPathFromUri(backend.Manager.VagrantCatalogRootUri)
	if err != nil {
		fmt.Printf("Error trying to parse local catalog path from URI: %v\n", err)
		return
	}
	catalogFilename := fmt.Sprintf("%v.json", backend.Manager.VagrantCatalogName)
	backend.VagrantCatalogRootPath = catalogRootPath
	backend.VagrantCatalogPath = path.Join(catalogRootPath, catalogFilename)

	return
}

func (cb *CaryatidLocalFileBackend) GetManager() (manager *BackendManager, err error) {
	manager = cb.Manager
	if manager == nil {
		err = fmt.Errorf("The Manager property was not set")
	}
	return
}

func (backend *CaryatidLocalFileBackend) GetCatalogBytes() (catalogBytes []byte, err error) {
	catalogBytes, err = ioutil.ReadFile(backend.VagrantCatalogPath)
	if os.IsNotExist(err) {
		log.Printf("No file at '%v'; starting with empty catalog\n", backend.VagrantCatalogPath)
		catalogBytes = []byte("{}")
		err = nil
	} else if err != nil {
		log.Printf("Error trying to read catalog: %v\n", err)
	}
	return
}

func (backend *CaryatidLocalFileBackend) SetCatalogBytes(serializedCatalog []byte) (err error) {

	err = os.MkdirAll(backend.VagrantCatalogRootPath, 0777)
	if err != nil {
		log.Printf("Error trying to create the catalog root path at '%v': %v\b", backend.VagrantCatalogRootPath, err)
		return
	}

	err = ioutil.WriteFile(backend.VagrantCatalogPath, serializedCatalog, 0666)
	if err != nil {
		log.Println("Error trying to write catalog: ", err)
		return
	}
	log.Println(fmt.Sprintf("Catalog updated on disk to reflect new value"))
	return
}

func (backend *CaryatidLocalFileBackend) CopyBoxFile(box *BoxArtifact) (err error) {

	remoteBoxPath, err := util.ParseLocalPathFromUri(box.GetUri())
	if err != nil {
		fmt.Printf("Error trying to parse local artifact path from URI: %v\n", err)
		return
	}

	remoteBoxParentPath, err := util.ParseLocalPathFromUri(box.GetParentUri())
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
