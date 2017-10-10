/*
The localfile backend, for dealing with a Vagrant catalog on a local filesystem
*/

package caryatid

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"

	"github.com/mrled/caryatid/internal/util"
)

type CaryatidLocalFileBackend struct {
	VagrantCatalogRootPath string
	VagrantCatalogPath     string
	Manager                *BackendManager
}

func (backend *CaryatidLocalFileBackend) SetManager(manager *BackendManager) (err error) {
	backend.Manager = manager

	backend.VagrantCatalogPath, err = getValidLocalPath(backend.Manager.CatalogUri)
	if err != nil {
		fmt.Printf("Error trying to parse local catalog path from URI: %v\n", err)
		return
	}

	backend.VagrantCatalogRootPath, _ = path.Split(backend.VagrantCatalogPath)

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

func (backend *CaryatidLocalFileBackend) CopyBoxFile(localPath string, boxName string, boxVersion string, boxProvider string) (err error) {
	var boxUri string

	boxUri, err = BoxUriFromCatalogUri(backend.VagrantCatalogPath, boxName, boxVersion, boxProvider)
	if err != nil {
		fmt.Printf("Error trying to determine box URI: %v\n", err)
		return
	}

	remoteBoxPath, err := getValidLocalPath(boxUri)
	if err != nil {
		fmt.Printf("Error trying to parse local artifact path from URI: %v\n", err)
		return
	}

	remoteBoxParentPath, _ := path.Split(remoteBoxPath)
	err = os.MkdirAll(remoteBoxParentPath, 0777)
	if err != nil {
		log.Println("Error trying to create the box directory: ", err)
		return
	}
	log.Printf("Successfully created directory at %v\n", remoteBoxParentPath)

	written, err := util.CopyFile(localPath, remoteBoxPath)
	if err != nil {
		log.Printf("Error trying to copy '%v' to '%v' file: %v\n", localPath, remoteBoxPath, err)
		return
	}
	log.Printf("Copied %v bytes from original path at '%v' to new location at '%v'\n", written, localPath, remoteBoxPath)
	return
}

func (backend *CaryatidLocalFileBackend) DeleteFile(uri string) (err error) {
	var (
		u    *url.URL
		path string
	)
	u, err = url.Parse(uri)
	if err != nil {
		return fmt.Errorf("Could not parse '%v' as URI: %v", uri, err)
	}
	if u.Scheme != backend.Scheme() {
		return fmt.Errorf("Expected scheme '%v' but was given a URI with scheme '%v'", backend.Scheme(), u.Scheme)
	}

	if path, err = getValidLocalPath(uri); err != nil {
		return
	}
	if err = os.Remove(path); err != nil {
		return
	}

	return
}

func (backend *CaryatidLocalFileBackend) Scheme() string {
	return "file"
}

// Get a valid local path from a URI
// Converts URI paths (with '/' separator) to Windows paths (with '\' separator) when on Windows
// On Windows, a URI will sometimes be in the form 'file:///C:/path/to/something' (or 'file:///C:\\path\\to\\something')
func getValidLocalPath(uri string) (outpath string, err error) {
	u, err := url.Parse(uri)
	if err != nil {
		return
	}

	outpath = u.Path
	if u.Path == "" {
		err = fmt.Errorf("No valid path information was provided in the URI '%v'", uri)
		return
	}

	// On Windows, valid URIs look like file:///C:/whatever or file:///C:\\whatever
	// The naivePath variable will contain that leading slash, like "/C:/whatever" or "/C:\\whatever"
	// If the path looks like that, strip the leading slash
	matched, err := regexp.MatchString("^/[a-zA-Z]:", outpath)
	if err != nil {
		return
	} else if matched {
		outpath = outpath[1:len(outpath)]
	}

	// Get an absolute path
	// If on Windows, replaces any forward slashes in the URI with backslashes
	outpath = filepath.Clean(outpath)

	return
}
