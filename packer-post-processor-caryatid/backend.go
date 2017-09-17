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
type BackendManager struct {
	VagrantCatalogRootUri string
	VagrantCatalogName    string
	VagrantCatalog        *Catalog
	Backend               CaryatidBackend
}

func (bm *BackendManager) Configure(catalogRootUri string, catalogName string, backend *CaryatidBackend) (err error) {
	bm.VagrantCatalogRootUri = catalogRootUri
	bm.VagrantCatalogName = catalogName
	bm.Backend = *backend
	bm.Backend.SetManager(bm)
	catalog, err := bm.GetCatalog()
	if err != nil {
		log.Printf("Error trying to get catalog: %v\n", err)
		return
	}
	bm.VagrantCatalog = &catalog
	return
}

func (bm *BackendManager) GetCatalog() (catalog Catalog, err error) {
	catalogBytes, err := bm.Backend.GetCatalogBytes()
	if err != nil {
		log.Printf("Error trying to get catalog bytes: %v\n", err)
		return
	}

	err = json.Unmarshal(catalogBytes, &catalog)

	return
}

func (bm *BackendManager) SaveCatalog() (err error) {
	jsonData, err := json.MarshalIndent(bm.VagrantCatalog, "", "  ")
	if err != nil {
		log.Println("Error trying to marshal catalog: ", err)
		return
	}
	err = bm.Backend.SetCatalogBytes(jsonData)
	if err != nil {
		log.Printf("Error saving catalog: %v\n", err)
	}
	return
}

func (bm *BackendManager) AddBoxMetadataToCatalog(box *BoxArtifact) (err error) {
	if err = bm.VagrantCatalog.AddBox(box); err != nil {
		log.Printf("AddBoxMetadataToCatalog(): Error adding box to catalog metadata object: %v", err)
		return
	}
	if err = bm.SaveCatalog(); err != nil {
		log.Printf("AddBoxMetadataToCatalog(): Error saving catalog: %v", err)
		return
	}
	return
}

/*
The interface we use to deal with Caryatid backends

It is intended that you put an anonymous CaryatidBaseBackend in each implemented Caryatid backend, which lets you take advantage of shared logic that doesn't change between backends.
*/
type CaryatidBackend interface {
	// Set the manager to an internal property so the backend can access its properties/methods
	// This is an appropriate place for setup code, since it's always called from BackendManager.Configure()
	SetManager(*BackendManager) error

	// Return the manager from an internal property
	// So far this is only used for testing
	GetManager() (*BackendManager, error)

	// Get the raw byte value held in the Vagrant catalog
	GetCatalogBytes() ([]byte, error)

	// Save a raw byte value to the Vagrant catalog
	SetCatalogBytes([]byte) error

	// Copy the Vagrant box to the location referenced in the Vagrant catalog
	CopyBoxFile(*BoxArtifact) error
}

// A stub implementation of CaryatidBackend
type CaryatidBaseBackend struct {
	Manager *BackendManager
}

func (cb *CaryatidBaseBackend) SetManager(manager *BackendManager) (err error) {
	cb.Manager = manager
	return
}

func (cb *CaryatidBaseBackend) GetManager() (manager *BackendManager, err error) {
	manager = cb.Manager
	if manager == nil {
		err = fmt.Errorf("The Manager property was not set")
	}
	return
}

func (cb *CaryatidBaseBackend) GetCatalogBytes() (catalogBytes []byte, err error) {
	err = fmt.Errorf("NOT IMPLEMENTED")
	return
}

func (cb *CaryatidBaseBackend) SetCatalogBytes(serializedCatalog []byte) (err error) {
	err = fmt.Errorf("NOT IMPLEMENTED")
	return
}

func (cb *CaryatidBaseBackend) CopyBoxFile(box *BoxArtifact) (err error) {
	err = fmt.Errorf("NOT IMPLEMENTED")
	return
}
