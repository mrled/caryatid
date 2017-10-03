/*
The interface for a Caryatid backend
*/

package caryatid

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
)

func NewBackend(name string) (backend CaryatidBackend, err error) {
	switch name {
	case "file":
		backend = &CaryatidLocalFileBackend{}
	case "s3":
		backend = &CaryatidS3Backend{}
	default:
		err = fmt.Errorf("No known backend with name '%v'", name)
	}
	return
}

func NewBackendFromUri(uri string) (backend CaryatidBackend, err error) {
	u, err := url.Parse(uri)
	if err != nil {
		err = fmt.Errorf("Error trying to parse URI '%v': %v\n", uri, err)
		return
	}
	backend, err = NewBackend(u.Scheme)
	return
}

// Manages Vagrant catalogs via various backends
type BackendManager struct {
	VagrantCatalogRootUri string
	VagrantCatalogName    string
	Backend               CaryatidBackend
}

func NewBackendManager(catalogRootUri string, catalogName string, backend *CaryatidBackend) (bm *BackendManager) {
	bm = &BackendManager{
		catalogRootUri,
		catalogName,
		*backend,
	}
	bm.Backend.SetManager(bm)
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

func (bm *BackendManager) SaveCatalog(catalog Catalog) (err error) {
	jsonData, err := json.MarshalIndent(catalog, "", "  ")
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

func (bm *BackendManager) AddBox(box *BoxArtifact) (err error) {
	catalog, err := bm.GetCatalog()
	if err != nil {
		log.Printf("AddBox(): Error retrieving catalog from backend: %v\n", err)
		return
	}
	if err = catalog.AddBox(box); err != nil {
		log.Printf("AddBox(): Error adding box to catalog metadata object: %v\n", err)
		return
	}
	if err = bm.SaveCatalog(catalog); err != nil {
		log.Printf("AddBox(): Error saving catalog: %v\n", err)
		return
	}
	if err = bm.Backend.CopyBoxFile(box); err != nil {
		log.Printf("AddBox(): Error copying box file: %v\n", err)
		return
	}
	return
}

// TODO: Want this to call Backend.DeleteBoxFile also
func (bm *BackendManager) DeleteBox(params CatalogQueryParams) (err error) {
	var (
		catalog       Catalog
		deleteCatalog Catalog
		refs          BoxReferenceList
	)

	if catalog, err = bm.GetCatalog(); err != nil {
		log.Printf("DeleteBox(): Error retrieving catalog from backend: %v\n", err)
		return
	}
	if deleteCatalog, err = catalog.QueryCatalog(params); err != nil {
		log.Printf("DeleteBox(): Error querying catalog: %v\n", err)
		return
	}

	refs = deleteCatalog.BoxReferences()
	catalog = catalog.DeleteReferences(refs)
	if err = bm.SaveCatalog(catalog); err != nil {
		log.Printf("DeleteBox(): Error saving catalog: %v\n", err)
		return
	}

	for _, ref := range refs {
		if err = bm.Backend.DeleteFile(ref.Uri); err != nil {
			log.Printf("DeleteBox(): Error copying box file: %v\n", err)
			return
		}
	}

	return
}

/*
The interface we use to deal with Caryatid backends

It is intended that you put an anonymous CaryatidBaseBackend in each implemented Caryatid backend, which lets you take advantage of shared logic that doesn't change between backends.
*/
type CaryatidBackend interface {
	// Set the manager to an internal property so the backend can access its properties/methods
	// This is an appropriate place for setup code, since it's always called from NewBackendManager()
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

	// Delete a file with a given URI
	// If the URI's .Scheme doesn't match the value of .Scheme(), error
	DeleteFile(uri string) error

	// Return the scheme as would be used in the URI for the backend,
	// such as "file" for a "file:///tmp/catalog.json" catalog
	Scheme() string
}

// A stub implementation of CaryatidBackend
type CaryatidBaseBackend struct {
	Manager *BackendManager
}

func (backend *CaryatidBaseBackend) SetManager(manager *BackendManager) (err error) {
	backend.Manager = manager
	return
}

func (backend *CaryatidBaseBackend) GetManager() (manager *BackendManager, err error) {
	manager = backend.Manager
	if manager == nil {
		err = fmt.Errorf("The Manager property was not set")
	}
	return
}

func (backend *CaryatidBaseBackend) GetCatalogBytes() (catalogBytes []byte, err error) {
	err = fmt.Errorf("NOT IMPLEMENTED")
	return
}

func (backend *CaryatidBaseBackend) SetCatalogBytes(serializedCatalog []byte) (err error) {
	err = fmt.Errorf("NOT IMPLEMENTED")
	return
}

func (backend *CaryatidBaseBackend) CopyBoxFile(box *BoxArtifact) (err error) {
	err = fmt.Errorf("NOT IMPLEMENTED")
	return
}

func (backend *CaryatidBaseBackend) DeleteFile(uri string) error {
	u, err := url.Parse(uri)
	if err != nil {
		return fmt.Errorf("Could not parse '%v' as URI: %v", uri, err)
	}
	if u.Scheme != backend.Scheme() {
		return fmt.Errorf("Expected scheme '%v' but was given a URI with scheme '%v'", backend.Scheme(), u.Scheme)
	}
	return fmt.Errorf("NOT IMPLEMENTED")
}

func (backend *CaryatidBaseBackend) Scheme() string {
	return ""
}
