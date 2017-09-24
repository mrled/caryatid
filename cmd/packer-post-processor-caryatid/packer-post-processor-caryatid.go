/*
In architecture, an Atlas [https://atlas.hashicorp.com/] is a column or support sculpted in the form of a man; a Caryatid [https://github.com/mrled/packer-post-processor-caryatid] is such a support in the form of a woman.

Caryatid is a packer post-processor plugin that provides a way to host a (versioned) Vagrant catalog on systems without having to use (and pay for) Atlas, and without having to trust a third party unless you want to.
*/

package main

import (
	"fmt"
	"log"

	"github.com/hashicorp/packer/common"
	"github.com/hashicorp/packer/helper/config"
	"github.com/hashicorp/packer/packer"
	"github.com/hashicorp/packer/packer/plugin"
	"github.com/hashicorp/packer/template/interpolate"

	"github.com/mrled/caryatid/pkg/caryatid"
)

//// Packer's PostProcessor interface methods

type Config struct {
	common.PackerConfig `mapstructure:",squash"`

	// A name for the Vagrant box
	Name string `mapstructure:"name"`

	// The version for the current artifact
	Version string `mapstructure:"version"`

	// A short description for the Vagrant box
	Description string `mapstructure:"description"`

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

	inBoxFile, digestType, digest, provider, err := caryatid.DeriveArtifactInfoFromPackerArtifact(artifact)
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
		digestType,
		digest,
	}
	packerArtifact = &boxArtifact

	var backend caryatid.CaryatidBackend
	backend, err = caryatid.NewBackendFromUri(pp.config.CatalogRootUri)
	if err != nil {
		log.Printf("PostProcess(): Error trying to get backend: %v\n", err)
		return
	}
	manager := caryatid.NewBackendManager(pp.config.CatalogRootUri, pp.config.Name, &backend)

	err = manager.AddBoxMetadataToCatalog(&boxArtifact)
	if err != nil {
		log.Printf("PostProcess(): Error adding box metadata to catalog: %v\n", err)
		return
	}
	log.Println("PostProcess(): Catalog saved to backend")

	catalog, err := manager.GetCatalog()
	if err != nil {
		log.Printf("PostProcess(): Error getting catalog: %v\n", err)
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
