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
// By default, artifacts emited from the Vagrant post-processor are named packer_{{.BuildName}}_{{.Provider}}.box
// according to https://www.packer.io/docs/post-processors/vagrant.html
func determineProvider(boxFile string) (result string, err error) {
	re := regexp.MustCompile(".*_([[:alnum:]]+).box$")
	matches := re.FindStringSubmatch(boxFile)
	if len(matches) != 2 { // matches[0] is always the whole input, if there are any submatches
		err = fmt.Errorf("Wrong number of matches; expected 1, but found '%v'", len(matches))
		return
	} else {
		result = matches[1]
		return
	}
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

func (v1 Version) Equals(v2 Version) bool {
	if &v1 == &v2 {
		return true
	}
	if v1.Version != v2.Version {
		return false
	}
	if len(v1.Providers) != len(v2.Providers) {
		return false
	}
	for idx := 0; idx < len(v1.Providers); idx += 1 {
		if v1.Providers[idx] != v2.Providers[idx] {
			return false
		}
	}
	return true
}

// A catalog keeps track of multiple versions and providers of a single box
type Catalog struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Versions    []Version `json:"versions"`
}

func (c1 Catalog) Equals(c2 Catalog) bool {
	if &c1 == &c2 {
		return true
	}
	if c1.Name != c2.Name {
		return false
	}
	if c1.Description != c2.Description {
		return false
	}
	if len(c1.Versions) != len(c2.Versions) {
		return false
	}
	for idx := 0; idx < len(c1.Versions); idx += 1 {
		if !c1.Versions[idx].Equals(c2.Versions[idx]) {
			return false
		}
	}
	return true
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
	newCatalog.Name = artifact.Name
	newCatalog.Description = artifact.Description

	artifactUrl := fmt.Sprintf("file://%v/%v/%v_%v_%v.box", artifact.CatalogRoot, artifact.Name, artifact.Name, artifact.Version, artifact.Provider)
	newProvider := Provider{artifact.Provider, artifactUrl, artifact.ChecksumType, artifact.Checksum}
	newVersion := Version{artifact.Version, []Provider{newProvider}}

	foundVersion := false
	foundProvider := false

	for vidx, _ := range newCatalog.Versions {
		if newCatalog.Versions[vidx].Version == artifact.Version {
			foundVersion = true
			for pidx, _ := range newCatalog.Versions[vidx].Providers {
				if newCatalog.Versions[vidx].Providers[pidx].Name == artifact.Provider {
					newCatalog.Versions[vidx].Providers[pidx].Url = artifactUrl
					newCatalog.Versions[vidx].Providers[pidx].ChecksumType = artifact.ChecksumType
					newCatalog.Versions[vidx].Providers[pidx].Checksum = artifact.Checksum
					foundProvider = true
					break
				}
			}
			if !foundProvider {
				newCatalog.Versions[vidx].Providers = append(newCatalog.Versions[vidx].Providers, newProvider)
			}
			break
		}
	}
	if !foundVersion {
		newCatalog.Versions = append(newCatalog.Versions, newVersion)
	}

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

	provider, err := determineProvider(boxFile)
	if err != nil {
		return
	}

	boxArtifact := BoxArtifact{
		boxFile,
		pp.config.Name,
		pp.config.Description,
		pp.config.Version,
		provider,
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