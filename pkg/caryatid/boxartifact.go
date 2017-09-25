/*
Used to keep track of a box artifact that we have been passed
Implements the packer.Artifact interface
*/

package caryatid

import (
	"fmt"
)

const BuilderId = "com.micahrl.caryatid"

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
	// The root URI of the Vagrant catalog
	CatalogRootUri string
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

func (artifact *BoxArtifact) String() string {
	return fmt.Sprintf("%v/%v v%v %v:%v (%v)", artifact.Name, artifact.Provider, artifact.Version, artifact.ChecksumType, artifact.Checksum, artifact.Description)
}

func (*BoxArtifact) State(name string) interface{} {
	return nil
}

func (art *BoxArtifact) Destroy() error {
	return nil
}

func (artifact *BoxArtifact) GetParentUri() string {
	return fmt.Sprintf("%v/%v", artifact.CatalogRootUri, artifact.Name)
}

func (artifact *BoxArtifact) GetUri() string {
	return fmt.Sprintf("%v/%v_%v_%v.box", artifact.GetParentUri(), artifact.Name, artifact.Version, artifact.Provider)
}

func (ba1 *BoxArtifact) Equals(ba2 *BoxArtifact) bool {
	if ba1 == nil || ba2 == nil {
		return false
	}
	if ba1 == ba2 {
		return true
	}
	return (ba1.Path == ba2.Path &&
		ba1.Name == ba2.Name &&
		ba1.Description == ba2.Description &&
		ba1.Version == ba2.Version &&
		ba1.Provider == ba2.Provider &&
		ba1.CatalogRootUri == ba2.CatalogRootUri &&
		ba1.ChecksumType == ba2.ChecksumType &&
		ba1.Checksum == ba2.Checksum)
}
