/*
In architecture, an Atlas [https://atlas.hashicorp.com/] is a column or support sculpted in the form of a man; a Caryatid [https://github.com/mrled/packer-post-processor-caryatid] is such a support in the form of a woman.

Caryatid is a packer post-processor plugin that provides a way to host a (versioned) Vagrant catalog on systems without having to use (and pay for) Atlas, and without having to trust a third party unless you want to.

Here's the JSON of an example catalog:

	{
		"name": "testbox",
		"description": "Just an example",
		"versions": [{
			"version": "0.1.0",
			"providers": [{
				"name": "virtualbox",
				"url": "user@example.com/caryatid/boxes/testbox_0.1.0.box",
				"checksum_type": "sha1",
				"checksum": "d3597dccfdc6953d0a6eff4a9e1903f44f72ab94"
			}]
		}]
	}
*/
package caryatid

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
)

// Used in a Version, in a Catalog
type Provider struct {
	Name         string `json:"name"`
	Url          string `json:"url"`
	ChecksumType string `json:"checksum_type"`
	Checksum     string `json:"checksum"`
}

// Used in a Catalog
type Version struct {
	Version   string     `json:"version"`
	Providers []Provider `json:"providers"`
}

// A catalog keeps track of multiple versions and providers of a single box
type Catalog struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Versions    []Version `json:"versions"`
}

// Used to keep track of a box artifact that we have been passed
type BoxArtifact struct {
	// The local path for the box
	Path string
	// The box name, like "win10x64"
	Name string
	// A box description, like "Windows 10, 64-bit"
	Description string
	// The version of this artifact, like "1.0.0"
	Version string
	// The provider for this artifact, like "virtualbox" or "vmware"
	Provider string
	// The final URL for this artifact. May be blank. Will be the final URL that is saved in the catalog for Vagrant to use to fetch this artifact.
	Url string
	// The type of checksum e.g. "sha1"
	ChecksumType string
	// A hex checksum
	Checksum string
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

// func AddBoxToCatalog(catalog Catalog, boxUrl string, boxName string, boxDesc string, boxVers string, boxProvider string, checksumType string, checksum string) (newCatalog Catalog) {

func AddBoxToCatalog(catalog Catalog, artifact BoxArtifact) (newCatalog Catalog) {

	newCatalog = catalog

	if newCatalog.Name == "" {
		newCatalog.Name = artifact.Name
	}
	if newCatalog.Description == "" {
		newCatalog.Description = artifact.Description
	}

	var version *Version
	for _, v := range newCatalog.Versions {
		if v.Version == artifact.Version {
			version = &v
			break
		}
	}
	if version == nil {
		version = new(Version)
		version.Version = artifact.Version
		newCatalog.Versions = append(newCatalog.Versions, *version)
	}

	var provider *Provider
	for _, p := range version.Providers {
		if p.Name == artifact.Provider {
			provider = &p
			break
		}
	}
	if provider == nil {
		provider = new(Provider)
		provider.Name = artifact.Provider
		version.Providers = append(version.Providers, *provider)
	}
	provider.Url = artifact.Url
	provider.ChecksumType = artifact.ChecksumType
	provider.Checksum = artifact.Checksum

	return
}

func UnmarshalCatalog(catalogPath string) (catalog Catalog, err error) {
	if catalogBytes, readerr := ioutil.ReadFile(catalogPath); readerr != nil {
		if os.IsNotExist(readerr) {
			catalogBytes = []byte("{}")
			err = json.Unmarshal(catalogBytes, &catalog)
		} else {
			err = readerr
		}
	}
	return
}

func IngestBox(catalogRoot string, artifact BoxArtifact, backend string) (err error) {
	var catalog Catalog
	catalogPath := path.Join(catalogRoot, fmt.Sprintf("%v.json", artifact.Name))
	if catalog, err = UnmarshalCatalog(catalogPath); err != nil {
		return
	}

	switch backend {
	case "copy":
		artifact.Url = fmt.Sprintf("file://%v/%v/%v_%v_%v.box", catalogRoot, artifact.Name, artifact.Name, artifact.Version, artifact.Provider)
	default:
		panic("Unknown backend")
	}

	catalog = AddBoxToCatalog(catalog, artifact)

	return
}
