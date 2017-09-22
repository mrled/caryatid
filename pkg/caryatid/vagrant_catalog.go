/*
A Vagrant catalog is a JSON file for managing Vagrant boxes
A catalog references for one or more versions which reference one or more providers which each point to a single Vagrant box file

Here's the JSON of an example catalog:

	{
		"name": "testbox",
		"description": "Just an example",
		"versions": [
			{
				"version": "0.1.0",
				"providers": [
					{
						"name": "virtualbox",
						"url": "user@example.com/caryatid/boxes/testbox_0.1.0.box",
						"checksum_type": "sha1",
						"checksum": "d3597dccfdc6953d0a6eff4a9e1903f44f72ab94"
					}
				]
			}
		]
	}
*/

package caryatid

import (
	"fmt"
)

type Provider struct {
	Name         string `json:"name"`
	Url          string `json:"url"`
	ChecksumType string `json:"checksum_type"`
	Checksum     string `json:"checksum"`
}

func (p1 *Provider) Equals(p2 *Provider) bool {
	if p1 == nil || p2 == nil {
		return false
	}
	return *p1 == *p2
}

type Version struct {
	Version   string     `json:"version"`
	Providers []Provider `json:"providers"`
}

func (v1 *Version) Equals(v2 *Version) bool {
	if v1 == nil || v2 == nil {
		return false
	}
	if v1 == v2 {
		return true
	}
	if v1.Version != v2.Version || len(v1.Providers) != len(v2.Providers) {
		return false
	}
	for idx := 0; idx < len(v1.Providers); idx += 1 {
		if !v1.Providers[idx].Equals(&v2.Providers[idx]) {
			return false
		}
	}
	return true
}

type Catalog struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Versions    []Version `json:"versions"`
}

func (c1 *Catalog) Equals(c2 *Catalog) bool {
	if c1 == nil || c2 == nil {
		return false
	}
	if c1 == c2 {
		return true
	}
	if c1.Name != c2.Name || c1.Description != c2.Description || len(c1.Versions) != len(c2.Versions) {
		return false
	}
	for idx := 0; idx < len(c1.Versions); idx += 1 {
		if !c1.Versions[idx].Equals(&c2.Versions[idx]) {
			return false
		}
	}
	return true
}

func (c *Catalog) AddBox(artifact *BoxArtifact) (err error) {
	if c.Name == "" {
		c.Name = artifact.Name
	} else if c.Name != artifact.Name {
		err = fmt.Errorf("Catalog.AddBox(): Catalog name '%v' does not match artifact name '%v'", c.Name, artifact.Name)
		return
	}

	// What to do with conflicting descriptions is a bit uncertain.
	// We have decided to overwritew a conflicting description in the existing catalog,
	// rather than failing with an error or ignoring a conflicting description in the incoming artifact
	// This minimizes painful end-of-build errors,
	// and lets the user change their mind about the wording of the description
	c.Description = artifact.Description

	newProvider := Provider{artifact.Provider, artifact.GetUri(), artifact.ChecksumType, artifact.Checksum}
	newVersion := Version{artifact.Version, []Provider{newProvider}}

	foundVersion := false
	foundProvider := false

	for vidx, _ := range c.Versions {
		if c.Versions[vidx].Version == artifact.Version {
			foundVersion = true
			for pidx, _ := range c.Versions[vidx].Providers {
				if c.Versions[vidx].Providers[pidx].Name == artifact.Provider {
					c.Versions[vidx].Providers[pidx].Url = artifact.GetUri()
					c.Versions[vidx].Providers[pidx].ChecksumType = artifact.ChecksumType
					c.Versions[vidx].Providers[pidx].Checksum = artifact.Checksum
					foundProvider = true
					break
				}
			}
			if !foundProvider {
				c.Versions[vidx].Providers = append(c.Versions[vidx].Providers, newProvider)
			}
			break
		}
	}
	if !foundVersion {
		c.Versions = append(c.Versions, newVersion)
	}

	return
}
