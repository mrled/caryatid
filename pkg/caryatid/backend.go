/*
The interface for a Caryatid backend
*/

package caryatid

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
	CopyBoxFile(string, string, string, string) error

	// Delete a file with a given URI
	// If the URI's .Scheme doesn't match the value of .Scheme(), error
	DeleteFile(uri string) error

	// Return the scheme as would be used in the URI for the backend,
	// such as "file" for a "file:///tmp/catalog.json" catalog
	Scheme() string
}
