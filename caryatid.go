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
	"archive/zip"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"

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

func determineProviderFromMetadata(boxFilePath string) (result string, err error) {
	zipReader, err := zip.OpenReader(boxFilePath)
	if err != nil {
		return
	}
	defer zipReader.Close()

	var openedMetadataFile io.ReadCloser
	for _, zippedFile := range zipReader.File {
		if strings.ToLower(zippedFile.Name) == "metadata.json" {
			openedMetadataFile, err = zippedFile.Open()
			defer openedMetadataFile.Close()
			if err != nil {
				return
			}
			break
		}
	}
	metadataContents, err := ioutil.ReadAll(openedMetadataFile)
	if err != nil {
		return
	}

	var metadata struct {
		Provider string `json:provider`
	}
	if err = json.Unmarshal([]byte(metadataContents), &metadata); err != nil {
		return
	}

	result = metadata.Provider
	return
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

func copyFile(src string, dst string) (written int64, err error) {
	in, err := os.Open(src)
	defer in.Close()
	if err != nil {
		return
	}
	out, err := os.Create(dst)
	defer out.Close()
	if err != nil {
		return
	}
	written, err = io.Copy(out, in)
	if err != nil {
		return
	}
	err = out.Close()
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

const BuilderId = "com.micahrl.caryatid"

// Used to keep track of a box artifact that we have been passed
// Implements the packer.Artifact interface
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

func (*BoxArtifact) BuilderId() string {
	return BuilderId
}

func (bxart *BoxArtifact) Files() []string {
	return nil
}

func (bxart *BoxArtifact) Id() string {
	return fmt.Sprintf("%s/%s/%s", bxart.Name, bxart.Provider, bxart.Version)
}

func (bxart *BoxArtifact) String() string {
	return fmt.Sprintf("%s/%s (v. %d)", bxart.Name, bxart.Provider, bxart.Version)
}

func (*BoxArtifact) State(name string) interface{} {
	return nil
}

func (art *BoxArtifact) Destroy() error {
	return nil
}

// Given a BoxArtifact metadata object and a Catalog object representing the contents of a Vagrant JSON catalog,
func AddBoxToCatalog(catalog Catalog, artifact BoxArtifact) (newCatalog Catalog) {

	newCatalog = catalog
	newCatalog.Name = artifact.Name
	newCatalog.Description = artifact.Description

	artifactUrl := fmt.Sprintf("%v/%v/%v_%v_%v.box", artifact.CatalogRoot, artifact.Name, artifact.Name, artifact.Version, artifact.Provider)
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

//// Packer's PostProcessor interface methods

type Config struct {
	common.PackerConfig `mapstructure:",squash"`

	// A name for the Vagrant box
	Name string `mapstructure:"name"`

	// The version for the current artifact
	Version string `mapstructure:"version"`

	// A short description for the Vagrant box
	Description string `mapstructure:"description"`

	// The root path for a Vagrant catalog
	// If the catalog URL is file:///tmp/mybox.json, CatalogRoot is "file:///tmp" and the Name is "mybox"
	CatalogRoot string `mapstructure:"catalog_root_url"`

	// Whether to keep the input artifact
	KeepInputArtifact bool `mapstructure:"keep_input_artifact"`

	ctx interpolate.Context
}

type CaryatidPostProcessor struct {
	config Config
}

func (pp *CaryatidPostProcessor) Configure(raws ...interface{}) error {
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
		return fmt.Errorf("Version required")
	}
	if pp.config.CatalogRoot == "" {
		return fmt.Errorf("CatalogRoot required")
	}

	return nil
}

func (pp *CaryatidPostProcessor) PostProcess(ui packer.Ui, artifact packer.Artifact) (outArtifact packer.Artifact, keepInputArtifact bool, err error) {

	keepInputArtifact = pp.config.KeepInputArtifact

	// Sanity check the artifact we were passed
	if len(artifact.Files()) != 1 {
		err = fmt.Errorf(
			"Wrong number of files in the input artifact; expected exactly 1 file but found %v",
			len(artifact.Files()))
		return
	}
	inBoxFile := artifact.Files()[0]
	if !strings.HasSuffix(inBoxFile, ".box") {
		err = fmt.Errorf("Box file '%v' doesn't have a '.box' file extension, and is therefore not a valid Vagrant box", inBoxFile)
		return
	}
	log.Println(fmt.Sprintf("Found input Vagrant .box file: '%v'", inBoxFile))

	var digest string
	digest, err = sha1sum(inBoxFile)
	if err != nil {
		log.Println("sha1sum failed for box file '%v' with error %v", inBoxFile, err)
		return
	}
	log.Println(fmt.Sprintf("Found SHA1 hash for file: '%v'", digest))

	provider, err := determineProvider(inBoxFile)
	if err != nil {
		log.Println("Could not determine provider from the filename for box file '%v'; got error %v", inBoxFile, err)
		return
	}
	log.Println(fmt.Sprintf("Determined provider as '%v'", provider))

	catalogRootUrl, err := url.Parse(pp.config.CatalogRoot)
	if err != nil {
		log.Println("Could not parse CatalogRoot URL of '%v'", pp.config.CatalogRoot)
		return
	}
	catalogRootPath := catalogRootUrl.Path
	boxDir := path.Join(catalogRootPath, pp.config.Name)

	// TODO: should do something more sensible than an unchangeable world-readable directory here
	err = os.MkdirAll(boxDir, 0777)
	if err != nil {
		log.Println("Error trying to create the box directory: ", err)
		return
	}

	boxPath := path.Join(boxDir, fmt.Sprintf("%v_%v_%v.box", pp.config.Name, pp.config.Version, provider))
	boxUrl, err := url.Parse(catalogRootUrl.String())
	if err != nil {
		log.Println("Unexpected error trying to copy catalog URL")
		return
	}
	boxUrl.Path = boxPath
	// catalogUrl = fmt.Sprintf("%v/", catalogRootUrl.String())

	boxArtifact := BoxArtifact{
		inBoxFile,
		pp.config.Name,
		pp.config.Description,
		pp.config.Version,
		provider,
		pp.config.CatalogRoot,
		"sha1",
		digest,
	}
	outArtifact = &boxArtifact

	var catalog Catalog
	catalogPath := path.Join(catalogRootPath, fmt.Sprintf("%v.json", boxArtifact.Name))
	log.Println(fmt.Sprintf("Using catalog path of '%v'", catalogPath))

	catalogBytes, err := ioutil.ReadFile(catalogPath)
	if os.IsNotExist(err) {
		log.Println(fmt.Sprintf("No file at '%v'; starting with empty catalog", catalogPath))
		catalogBytes = []byte("{}")
	} else if err != nil {
		log.Println("Error trying to read catalog: ", err)
		return
	}

	if err = json.Unmarshal(catalogBytes, &catalog); err != nil {
		log.Println("Error trying to unmarshal catalog: ", err)
		return
	}

	catalog = AddBoxToCatalog(catalog, boxArtifact)
	log.Println(fmt.Sprintf("Catalog updated; new value is:\n%v", catalog))

	jsonData, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		log.Println("Error trying to marshal catalog: ", err)
		return
	}
	err = ioutil.WriteFile(catalogPath, jsonData, 0666)
	if err != nil {
		log.Println("Error trying to write catalog: ", err)
		return
	}
	log.Println(fmt.Sprintf("Catalog updated on disk to reflect new value"))

	written, err := copyFile(inBoxFile, boxPath)
	if err != nil {
		log.Println(fmt.Sprintf("Error trying to copy '%v' to '%v' file: %v", inBoxFile, boxPath, err))
		return
	}
	log.Println(fmt.Sprintf("Copied %v bytes from original path at '%v' to new location at '%v'", written, inBoxFile, boxPath))

	return
}
