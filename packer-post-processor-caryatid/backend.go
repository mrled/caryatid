/*
The interface for a Caryatid backend
*/

package main

import (
	"encoding/json"
	"fmt"
	"log"
)

// Manages Vagrant catalogs via various backends
type VagrantCatalogManager struct {
	VagrantCatalogRootUri string
	VagrantBox            BoxArtifact
	VagrantCatalog        Catalog
	Backend               CaryatidBackend
}

func (vcm VagrantCatalogManager) Configure(catalogRootUri string, boxName string, backend CaryatidBackend) (err error) {
	vcm.VagrantCatalogRootUri = catalogRootUri
	vcm.Backend = backend
	vcm.Backend.SetManager(vcm)
	catalog, err := vcm.GetCatalog()
	if err != nil {
		return
	}
	vcm.VagrantCatalog = catalog
	return
}

func (vcm VagrantCatalogManager) GetCatalog() (catalog Catalog, err error) {
	catalogBytes, err := vcm.Backend.GetCatalogBytes()
	if err != nil {
		return
	}

	err = json.Unmarshal(catalogBytes, &catalog)

	return
}

func (vcm VagrantCatalogManager) SaveCatalog() (err error) {
	jsonData, err := json.MarshalIndent(vcm.VagrantCatalog, "", "  ")
	if err != nil {
		log.Println("Error trying to marshal catalog: ", err)
		return
	}
	err = vcm.Backend.SetCatalogBytes(jsonData)
	return
}

func (vcm VagrantCatalogManager) AddBoxMetadataToCatalog(box BoxArtifact) (err error) {
	vcm.VagrantCatalog.AddBox(box)
	vcm.SaveCatalog()
	return
}

/*
The interface we use to deal with Caryatid backends

It is intended that you put an anonymous CaryatidBaseBackend in each implemented Caryatid backend, which lets you take advantage of shared logic that doesn't change between backends.
*/
type CaryatidBackend interface {
	//// Functions that you *must* override

	// Get the raw byte value held in the Vagrant catalog
	GetCatalogBytes() ([]byte, error)

	// Save a raw byte value to the Vagrant catalog
	SetCatalogBytes([]byte) error

	// Copy the Vagrant box to the location referenced in the Vagrant catalog
	CopyBoxFile(string) error

	// Set the manager to an internal property so the backend can access its properties/methods
	// This is an appropriate place for setup code, since it's always called from VagrantCatalogManager.Configure()
	SetManager(VagrantCatalogManager) error
}

// A stub implementation of CaryatidBackend
type CaryatidBaseBackend struct {
	Manager VagrantCatalogManager
}

func (cb CaryatidBaseBackend) SetManager(manager VagrantCatalogManager) (err error) {
	cb.Manager = manager
	return
}

func (cb CaryatidBaseBackend) GetCatalogBytes() (catalogBytes []byte, err error) {
	err = fmt.Errorf("NOT IMPLEMENTED")
	return
}

func (cb CaryatidBaseBackend) SetCatalogBytes(serializedCatalog []byte) (err error) {
	err = fmt.Errorf("NOT IMPLEMENTED")
	return
}

func (cb CaryatidBaseBackend) CopyBoxFile(boxPath string) (err error) {
	err = fmt.Errorf("NOT IMPLEMENTED")
	return
}
