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

package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"regexp"

	"github.com/mitchellh/packer/common"
	"github.com/mitchellh/packer/helper/config"
	"github.com/mitchellh/packer/packer"
	"github.com/mitchellh/packer/template/interpolate"
)

//// Internal use only

// Hack version: pull it from the filename
// Final version should open the .box zipfile and extract it: https://www.vagrantup.com/docs/boxes/format.html
func determineProvider(boxFile string) string {
	re := regexp.MustCompile(`.*_\(a-zA-Z0-9+\)\.box$`)
	return re.FindString(boxFile)
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func sha1sum(filePath string) (result string, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer file.Close()

	hash := sha1.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		return
	}

	result = hex.EncodeToString(hash.Sum(nil))
	return
}

//// External interface

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
	// The *local* path for the box - not the final path after we copy the box to the server, but where the artifact is right now
	Path string
	// The box name, like "win10x64"
	Name string
	// A box description, like "Windows 10, 64-bit"
	Description string
	// The version of this artifact, like "1.0.0"
	Version string
	// The provider for this artifact, like "virtualbox" or "vmware"
	Provider string
	// The root path of the Vagrant catalog
	CatalogRoot string
	// The type of checksum e.g. "sha1"
	ChecksumType string
	// A hex checksum
	Checksum string
}

// Given a BoxArtifact metadata object and a Catalog object representing the contents of a Vagrant JSON catalog,
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
	provider.Url = fmt.Sprintf("file://%v/%v/%v_%v_%v.box", artifact.CatalogRoot, artifact.Name, artifact.Name, artifact.Version, artifact.Provider)
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

// // Add a reference to a box file to a catalog
// func IngestBox(catalogRoot string, artifact BoxArtifact, backend string) (err error) {
// 	var catalog Catalog
// 	catalogPath := path.Join(catalogRoot, fmt.Sprintf("%v.json", artifact.Name))
// 	if catalog, err = UnmarshalCatalog(catalogPath); err != nil {
// 		return
// 	}

// 	switch backend {
// 	case "copy":
// 		∑
// 	default:
// 		panic("Unknown backend")
// 	}

// 	catalog = AddBoxToCatalog(catalog, artifact)

// 	return
// }

//// Packer's PostProcessor interface methods

type Config struct {
	common.PackerConfig `mapstructure:",squash"`

	// A name for the Vagrant box
	Name string

	// The version for the current artifact
	Version string

	// A short description for the Vagrant box
	Description string

	// The root path for a Vagrant catalog
	// If the catalog URL is file:///tmp/mybox.json, CatalogRoot is "file:///tmp" and the Name is "mybox"
	CatalogRoot string

	ctx interpolate.Context
}

type PostProcessor struct {
	config Config
}

func (pp *PostProcessor) Configure(raws ...interface{}) error {
	err := config.Decode(
		&pp.config,
		&config.DecodeOpts{
			Interpolate:        true,
			InterpolateContext: &pp.config.ctx,
		},
		raws...)
	if err != nil {
		return err
	}

	if pp.config.Version == "" {
		return fmt.Errorf("Version equired")
	}
	if pp.config.CatalogRoot == "" {
		return fmt.Errorf("CatalogRoot equired")
	}

	return nil
}

func (pp *PostProcessor) PostProcess(ui packer.Ui, artifact packer.Artifact) (newArtifact packer.Artifact, keepOldArtifact bool, err error) {

	// Sanity check the artifact we were passed
	if len(artifact.Files()) != 1 {
		err = fmt.Errorf(
			"Wrong number of files in the input artifact; expected exactly 1 file but found %v",
			len(artifact.Files()))
		return
	}
	boxFile := artifact.Files()[0]

	digest, err := sha1sum(boxFile)
	if err != nil {
		return
	}

	boxArtifact := BoxArtifact{
		boxFile,
		pp.config.Name,
		pp.config.Description,
		pp.config.Version,
		determineProvider(boxFile),
		pp.config.CatalogRoot,
		"sha1",
		digest,
	}

	var catalog Catalog
	catalogPath := path.Join(pp.config.CatalogRoot, fmt.Sprintf("%v.json", boxArtifact.Name))
	if catalog, err = UnmarshalCatalog(catalogPath); err != nil {
		return
	}
	catalog = AddBoxToCatalog(catalog, boxArtifact)
	log.Println(catalog)

	// TODO: save the catalog and copy the artifact file

	// TODO: not sure how to handle keepOldArtifact (do I have to handle it myself?)
	// TODO: create a new packer.Artifact, don't just return the old one
	return artifact, true, err
}
