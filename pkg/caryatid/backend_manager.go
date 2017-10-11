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
	CatalogUri string
	Backend    CaryatidBackend
}

// TODO: Should this also just call NewBackendFromUri()? Why split them out?
func NewBackendManager(catalogUri string, backend *CaryatidBackend) (bm *BackendManager) {
	bm = &BackendManager{
		catalogUri,
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
	if err != nil {
		log.Printf("Error unmashalling catalog: %v\ncatalogbytes:\n%v\n", err, catalogBytes)
		return
	}

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

func (bm *BackendManager) AddBox(localPath string, name string, description string, version string, provider string, checksumType string, checksum string) (err error) {

	catalog, err := bm.GetCatalog()
	if _, err = NewComparableVersion(version); err != nil {
		log.Printf("AddBox(): Invalid version '%v'\n", version)
	}
	if err != nil {
		log.Printf("AddBox(): Error retrieving catalog from backend: %v\n", err)
		return
	}

	err = catalog.AddBox(bm.CatalogUri, name, description, version, provider, checksumType, checksum)
	if err != nil {
		log.Printf("AddBox(): Error adding box to catalog metadata object: %v\n", err)
		return
	}
	if err = bm.SaveCatalog(catalog); err != nil {
		log.Printf("AddBox(): Error saving catalog: %v\n", err)
		return
	}
	if err = bm.Backend.CopyBoxFile(localPath, name, version, provider); err != nil {
		log.Printf("AddBox(): Error copying box file: %v\n", err)
		return
	}
	return
}

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
