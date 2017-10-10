/*
An artifact, implementing the packer.Artifact interface
This is what we return
*/

package main

import (
	"fmt"
)

const BuilderId = "com.micahrl.caryatid"

type CaryatidOutputArtifact struct {
	// The URI of the Vagrant catalog
	CatalogUri string
	// A box description, like "Windows 10, 64-bit"
	Description string
	// The version of this artifact, like "1.0.0"
	Version string
	// The provider for this artifact, like "virtualbox" or "vmware"
	Provider string
	// The type of checksum e.g. "sha1"
	ChecksumType string
	// A hex checksum
	Checksum string
}

func (*CaryatidOutputArtifact) BuilderId() string {
	return BuilderId
}

func (*CaryatidOutputArtifact) Files() []string {
	return nil
}

func (artifact *CaryatidOutputArtifact) Id() string {
	return fmt.Sprintf("%s#?version=%s&provider=%s&checksum=%s:%s", artifact.CatalogUri, artifact.Version, artifact.Provider, artifact.ChecksumType, artifact.Checksum)
}

func (artifact *CaryatidOutputArtifact) String() string {
	return fmt.Sprintf("%v\n(%v)", artifact.Id(), artifact.Description)
}

func (*CaryatidOutputArtifact) State(name string) interface{} {
	return nil
}

func (art *CaryatidOutputArtifact) Destroy() error {
	return nil
}

func (a1 *CaryatidOutputArtifact) Equals(a2 *CaryatidOutputArtifact) bool {
	if a1 == nil || a2 == nil {
		return false
	}
	if a1 == a2 {
		return true
	}
	return (a1.CatalogUri == a2.CatalogUri &&
		a1.Description == a2.Description &&
		a1.Version == a2.Version &&
		a1.Provider == a2.Provider &&
		a1.ChecksumType == a2.ChecksumType &&
		a1.Checksum == a2.Checksum)
}
