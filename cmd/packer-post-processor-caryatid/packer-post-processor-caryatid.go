/*
In architecture, an Atlas [https://atlas.hashicorp.com/] is a column or support sculpted in the form of a man; a Caryatid [https://github.com/mrled/packer-post-processor-caryatid] is such a support in the form of a woman.

Caryatid is a packer post-processor plugin that provides a way to host a (versioned) Vagrant catalog on systems without having to use (and pay for) Atlas, and without having to trust a third party unless you want to.
*/

package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/hashicorp/packer/common"
	"github.com/hashicorp/packer/helper/config"
	"github.com/hashicorp/packer/packer"
	"github.com/hashicorp/packer/packer/plugin"
	"github.com/hashicorp/packer/template/interpolate"

	"github.com/mrled/caryatid/internal/util"
	"github.com/mrled/caryatid/pkg/caryatid"
)

// Determine the provider of a Vagrant box based on its metadata.json
// See also https://www.packer.io/docs/post-processors/vagrant.html
func determineProvider(boxFilePath string) (result string, err error) {
	file, err := os.Open(boxFilePath)
	defer file.Close()
	if err != nil {
		return
	}

	magic := make([]byte, 2, 2)
	_, err = file.Read(magic)
	if err != nil {
		return
	}

	// "Rewind" the file reader. If we don't do this after reading the magic number,
	// then gzip.NewReader() will see the file starting at the third byte, and panic()
	_, err = file.Seek(0, 0)
	if err != nil {
		return
	}

	// The magic number for Gzip files is 0xF1 0x8B or, in decimal, 31 139.
	var tarReader tar.Reader
	if magic[0] == 31 && magic[1] == 139 {
		gzReader, err := gzip.NewReader(file)
		defer gzReader.Close()
		if err != nil {
			e := fmt.Errorf("Failed to create gzip reader for file '%v': %v", boxFilePath, err)
			fmt.Printf("%v\n", e)
			return result, e
		}
		tr := tar.NewReader(gzReader)
		tarReader = *tr
	} else {
		tr := tar.NewReader(file)
		tarReader = *tr
	}

	var metadataContents []byte
	done := false
	for done == false {
		header, err := tarReader.Next()
		if err == io.EOF {
			return result, fmt.Errorf("Could not find metadata.json file in %v", boxFilePath)
		} else if err != nil {
			return result, err
		}

		if strings.ToLower(header.Name) == "metadata.json" {
			done = true
			metadataContents, err = ioutil.ReadAll(&tarReader)
			if err != nil {
				return result, err
			}
		}
	}

	var metadata struct {
		Provider string `json:"provider"`
	}
	if err = json.Unmarshal(metadataContents, &metadata); err != nil {
		return
	}

	result = metadata.Provider
	return
}

func deriveArtifactInfo(artifact packer.Artifact) (boxFile string, digest string, provider string, err error) {
	if len(artifact.Files()) != 1 {
		err = fmt.Errorf(
			"Wrong number of files in the input artifact; expected exactly 1 file but found %v:\n%v",
			len(artifact.Files()), strings.Join(artifact.Files(), ", "))
		return
	}

	boxFile = artifact.Files()[0]
	if !strings.HasSuffix(boxFile, ".box") {
		err = fmt.Errorf("Input artifact '%v' doesn't have a '.box' file extension, and is therefore not a valid Vagrant box", boxFile)
		return
	}
	log.Println(fmt.Sprintf("Found input Vagrant .box file: '%v'", boxFile))

	digest, err = util.Sha1sum(boxFile)
	if err != nil {
		log.Printf("sha1sum failed for box file '%v' with error %v\n", boxFile, err)
		return
	}
	log.Println(fmt.Sprintf("Found SHA1 hash for file: '%v'", digest))

	provider, err = determineProvider(boxFile)
	if err != nil {
		log.Printf("Could not determine provider from the filename for box file '%v'; got error %v\n", boxFile, err)
		return
	}
	log.Println(fmt.Sprintf("Determined provider as '%v'", provider))

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

	// Name of the backend to use, such as "localfile"
	Backend string `mapstructure:"backend"`

	// The root URI for a Vagrant catalog
	// This is decoded separately by each backend
	// If the catalog URL is file:///tmp/mybox.json, CatalogRootUri is "file:///tmp" (and the Name is "mybox")
	CatalogRootUri string `mapstructure:"catalog_root_uri"`

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
	if pp.config.CatalogRootUri == "" {
		return fmt.Errorf("CatalogRoot required")
	}

	return nil
}

func (pp *CaryatidPostProcessor) PostProcess(ui packer.Ui, artifact packer.Artifact) (packerArtifact packer.Artifact, keepInputArtifact bool, err error) {

	keepInputArtifact = pp.config.KeepInputArtifact

	inBoxFile, digest, provider, err := deriveArtifactInfo(artifact)
	if err != nil {
		log.Printf("PostProcess(): Error deriving artifact information: %v", err)
		return
	}

	boxArtifact := caryatid.BoxArtifact{
		inBoxFile,
		pp.config.Name,
		pp.config.Description,
		pp.config.Version,
		provider,
		pp.config.CatalogRootUri,
		"sha1",
		digest,
	}
	packerArtifact = &boxArtifact

	var backend caryatid.CaryatidBackend
	switch pp.config.Backend {
	case "file":
		backend = &caryatid.CaryatidLocalFileBackend{}
	default:
		backend = &caryatid.CaryatidBaseBackend{}
	}
	manager := caryatid.NewBackendManager(pp.config.CatalogRootUri, pp.config.Name, &backend)

	err = manager.AddBoxMetadataToCatalog(&boxArtifact)
	if err != nil {
		log.Printf("PostProcess(): Error adding box metadata to catalog: %v", err)
		return
	}
	log.Println("PostProcess(): Catalog saved to backend")

	catalog, err := manager.GetCatalog()
	if err != nil {
		log.Printf("PostProcess(): Error getting catalog: %v", err)
		return
	}
	log.Printf("PostProcess(): New catalog is:\n%v\n", catalog)

	err = backend.CopyBoxFile(&boxArtifact)
	if err != nil {
		return
	}
	log.Println("PostProcess(): Box file copied successfully to backend")

	return
}

func main() {
	server, err := plugin.Server()
	if err != nil {
		panic(err)
	}

	server.RegisterPostProcessor(&CaryatidPostProcessor{})
	server.Serve()
}
