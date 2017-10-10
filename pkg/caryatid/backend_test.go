package caryatid

import (
	"fmt"
	"testing"
)

func TestCaryatidBaseBackend_ImplementsCaryatidBackend(t *testing.T) {
	var _ CaryatidBackend = new(CaryatidBaseBackend)
}

type CaryatidTestBackend struct {
	Manager     *BackendManager
	CatalogData []byte
}

func (cb *CaryatidTestBackend) SetManager(manager *BackendManager) (err error) {
	cb.Manager = manager
	return nil
}

func (cb *CaryatidTestBackend) GetManager() (manager *BackendManager, err error) {
	manager = cb.Manager
	if manager == nil {
		err = fmt.Errorf("The Manager property was not set")
	}
	return
}

func (cb *CaryatidTestBackend) GetCatalogBytes() (catalogBytes []byte, err error) {
	catalogBytes = cb.CatalogData
	if len(catalogBytes) == 0 {
		catalogBytes = []byte("{}")
	}
	return
}

func (cb *CaryatidTestBackend) SetCatalogBytes(serializedCatalog []byte) (err error) {
	cb.CatalogData = serializedCatalog
	return nil
}

func (cb *CaryatidTestBackend) CopyBoxFile(path string, box *BoxArtifact) error {
	return nil
}

func (bc *CaryatidTestBackend) DeleteFile(uri string) error {
	return nil
}

func (backend *CaryatidTestBackend) Scheme() string {
	return "Test"
}

func TestCaryatidTestBackend_ImplementsCaryatidBackend(t *testing.T) {
	var _ CaryatidBackend = new(CaryatidTestBackend)
}

func TestBackendManagerConfigure(t *testing.T) {
	cataRootUri := "http://example.com/cata"
	boxPath := "/tmp/path/to/example.box"
	testBox := BoxArtifact{
		"ExampleBox",
		"ExampleBox description",
		"192.168.0.1",
		"ExampleProvider",
		cataRootUri,
		"sha1",
		"0xDECAFBAD",
	}
	var backend CaryatidBackend = &CaryatidTestBackend{}
	manager := NewBackendManager(cataRootUri, testBox.Name, &backend)

	if manager.VagrantCatalogRootUri != cataRootUri {
		t.Fatal("VagrantCatalogRootUri property not set correctly")
	}

	if manager.VagrantCatalogName != testBox.Name {
		t.Fatal("VagrantCatalogName property not set properly")
	}

	expectedCata := Catalog{
		testBox.Name, testBox.Description, []Version{
			Version{testBox.Version, []Provider{
				Provider{testBox.Provider, boxPath, testBox.ChecksumType, testBox.Checksum},
			}},
		},
	}

	cata, err := manager.GetCatalog()
	if err != nil {
		t.Fatalf("Could not retrieve catalog from backend: %v", err)
	}
	if !cata.Equals(&Catalog{}) {
		t.Fatalf("VagrantCatalog property not set properly; result was\n%v\nbut we expected\n%v\n", cata, expectedCata)
	}

	if manager.Backend != backend {
		t.Fatal("Backend property not set properly")
	}

	if backendManager, err := manager.Backend.GetManager(); err != nil || backendManager == nil {
		t.Fatal(fmt.Sprintf("Backend Manager property not set properly; value was '%v'; error was '%v'", backendManager, err))
	}
}
