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

	// The URI for a Vagrant catalog
	// This is decoded separately by each backend
	CatalogUri string `mapstructure:"catalog_uri"`
	
	// A name for the Vagrant box
	Name string `mapstructure:"name"`

	// The version for the current artifact
	Version string `mapstructure:"version"`

	// A short description for the Vagrant box
	Description string `mapstructure:"description"`

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
	if pp.config.CatalogUri == "" {
		return fmt.Errorf("CatalogUri required")
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

	var backend caryatid.CaryatidBackend
	backend, err = caryatid.NewBackendFromUri(pp.config.CatalogUri)
	if err != nil {
		log.Printf("PostProcess(): Error trying to get backend: %v\n", err)
		return
	}
	manager := caryatid.NewBackendManager(pp.config.CatalogUri, &backend)

	err = manager.AddBox(inBoxFile, pp.config.Name, pp.config.Description, pp.config.Version, provider, digestType, digest)
	if err != nil {
		log.Printf("PostProcess(): Error adding box metadata to catalog: %v\n", err)
		return
	}
	log.Println("PostProcess(): New box added to backend")

	catalog, err := manager.GetCatalog()
	if err != nil {
		log.Printf("PostProcess(): Error getting catalog: %v\n", err)
		return
	}
	log.Printf("PostProcess(): New catalog is:\n%v\n", catalog)

	packerArtifact = &CaryatidOutputArtifact{
		CatalogUri: fmt.Sprintf("%v/%v.json", pp.config.CatalogUri, pp.config.Name),
		Description: pp.config.Description,
		Version: pp.config.Version,
		Provider: provider ,
		ChecksumType: digestType,
		Checksum: digest,
	}

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
