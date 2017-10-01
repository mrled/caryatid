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
	"log"
	"regexp"
	"strings"

	"github.com/mrled/caryatid/internal/util"
)

// Provider represents part of the structure of a Vagrant catalog
// It holds a box name, as well as its URL, checksum, and checksum type
type Provider struct {
	Name         string `json:"name"`
	Url          string `json:"url"`
	ChecksumType string `json:"checksum_type"`
	Checksum     string `json:"checksum"`
}

// Equals will return true if all properties of both Provider structs match
func (p1 *Provider) Equals(p2 *Provider) bool {
	if p1 == nil || p2 == nil {
		return false
	}
	return *p1 == *p2
}

// Version represents part of the structure of a Vagrant catalog
// It holds a string representing the version, as well as an array of Provider structs
type Version struct {
	Version   string     `json:"version"`
	Providers []Provider `json:"providers"`
}

// Equals compares two Version structs - including each of their Providers - and returns true if they are equal
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

// Catalog represents a Vagrant Catalog
// It holds the box name, its description, and an array of Version structs
type Catalog struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Versions    []Version `json:"versions"`
}

func (c *Catalog) DisplayString() (s string) {
	s = fmt.Sprintf("%v (%v)\n", c.Name, c.Description)
	for _, v := range c.Versions {
		s += fmt.Sprintf("  v%v\n", v.Version)
		for _, p := range v.Providers {
			s += fmt.Sprintf("    %v %v:%v <%v>\n", p.Name, p.ChecksumType, p.Checksum, p.Url)
		}
	}
	return
}

// Equals compares two Catalog structs - including their Versions, and those Versions' Providers - and returns true if they are equal
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

// AddBox updates the Catalog to include a new BoxArtifact
// The artifact's Name must match the Catalog's Name, if the Catalog already exists in storage
// However, the artifact's Description always overwrites the Catalog's Description, even if they are different
// This minimizes painful end-of-build errors,
// and lets the user change their mind about the wording of the description
func (c *Catalog) AddBox(artifact *BoxArtifact) (err error) {
	if c.Name == "" {
		c.Name = artifact.Name
	} else if c.Name != artifact.Name {
		err = fmt.Errorf("Catalog.AddBox(): Catalog name '%v' does not match artifact name '%v'", c.Name, artifact.Name)
		return
	}

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

// parseVersionQueryString parses a semver query string
// The string must be a valid semantic version string, optionally preceded by one of < > <= or >=
// TODO: Return a VersionComparator? Would have to extend VersionComparator to have GreaterThanOrEquals and LessThanOrEquals.
func parseVersionQueryString(semver string) (version ComparableVersion, qualifier string, err error) {
	if len(semver) == 0 {
		return
	}

	for _, prefix := range []string{">", "<", ">=", "<=", "="} {
		if strings.HasPrefix(semver, prefix) {
			qualifier = prefix
			version, err = NewComparableVersion(semver[len(prefix):])
		}
	}
	if qualifier == "" {
		version, err = NewComparableVersion(semver)
	}

	if err == nil {
		log.Printf("Parsed version query string '%v' into a version '%v' and a qualifier '%v'\n", semver, version, qualifier)
	} else {
		log.Printf("Error trying to create a ComparableVersion from input '%v': %v", semver, err)
	}

	return
}

// CatalogQueryParams represents valid parameters for QueryCatalog(), below
type CatalogQueryParams struct {
	Version  string
	Provider string
}

// QueryCatalogVersions returns a new Catalog containing only Versions that have a .Version property matching the versionquery input string
func (catalog *Catalog) QueryCatalogVersions(versionquery string) (result Catalog, err error) {
	var (
		comparator VersionComparator
		cVers      ComparableVersion
		pVers      ComparableVersion
		pVersQual  string
	)
	result.Name = catalog.Name
	result.Description = catalog.Description
	pVers, pVersQual, err = parseVersionQueryString(versionquery)
	if err != nil {
		return
	} else if len(pVers.Version) == 0 {
		result = *catalog
		return
	}
	for _, version := range catalog.Versions {
		if cVers, err = NewComparableVersion(version.Version); err != nil {
			return
		}
		comparator = cVers.Compare(&pVers)

		if pVersQual == "<" && comparator == VersionLessThan {
			result.Versions = append(result.Versions, version)

		} else if pVersQual == "<=" && (comparator == VersionLessThan || comparator == VersionEquals || comparator == VersionEqualsPrereleaseMismatch) {
			// Return prerelease-mismatched versions for <=
			result.Versions = append(result.Versions, version)

		} else if pVersQual == ">" && comparator == VersionGreaterThan {
			result.Versions = append(result.Versions, version)

		} else if pVersQual == ">=" && (comparator == VersionGreaterThan || comparator == VersionEquals || comparator == VersionEqualsPrereleaseMismatch) {
			// Return prerelease-mismatched versions for >=
			result.Versions = append(result.Versions, version)

		} else if pVersQual == "=" && comparator == VersionEquals {
			// If the versionquery qualifier is '=', return only an *exact* match
			result.Versions = append(result.Versions, version)

		} else if pVersQual == "" && (comparator == VersionEquals || comparator == VersionEqualsPrereleaseMismatch) {
			// If the versionquery qualifier is left off, but a version is passed,
			// return the version if it is an exact or prerelease-mismatched match
			result.Versions = append(result.Versions, version)
		}
	}
	return
}

// QueryCatalogProviders returns a new Catalog containing only Providers that have a .Name property matching the providerquery input string
func (catalog *Catalog) QueryCatalogProviders(providerquery string) (result Catalog, err error) {
	result.Name = catalog.Name
	result.Description = catalog.Description
	providerRegex := regexp.MustCompile(providerquery)
	for _, version := range catalog.Versions {
		newVersion := Version{version.Version, []Provider{}}
		for _, provider := range version.Providers {
			if providerRegex.Match([]byte(provider.Name)) {
				newVersion.Providers = append(newVersion.Providers, provider)
			}
		}
		if len(newVersion.Providers) > 0 {
			result.Versions = append(result.Versions, newVersion)
		}
	}
	return
}

// QueryCatalog returns a new catalog containing only matching boxes from a CatalogQueryParams input query
func (catalog *Catalog) QueryCatalog(params CatalogQueryParams) (result Catalog, err error) {
	var vResult, pResult Catalog
	if vResult, err = catalog.QueryCatalogVersions(params.Version); err != nil {
		return
	}
	if pResult, err = vResult.QueryCatalogProviders(params.Provider); err != nil {
		return
	}
	result = pResult
	result.Name = catalog.Name
	result.Description = catalog.Description
	return
}

// deleteBoxes deletes references to artifacts whose Version matches an item in vStrings or Provider matches an item in pStrings
// Note that this function *only* works with *exact* matches.
func (catalog *Catalog) deleteBoxes(vStrings []string, pStrings []string) (result Catalog) {
	result.Name = catalog.Name
	result.Description = catalog.Description

	for _, version := range catalog.Versions {

		if !util.StringInSlice(vStrings, version.Version) {
			newVersion := Version{version.Version, []Provider{}}
			for _, provider := range version.Providers {
				if !util.StringInSlice(pStrings, provider.Name) {
					newVersion.Providers = append(newVersion.Providers, provider)
				}
			}

			if len(newVersion.Providers) > 0 {
				result.Versions = append(result.Versions, newVersion)
			}
		}
	}

	return
}

type CatalogMetadata struct {
	Versions      []string
	Providers     []string
	ChecksumTypes []string
}

// Metadata returns a CatalogMetadata struct containing all the Version strings, Provider Names, and ChecksumTypes used in a Catalog
func (catalog *Catalog) Metadata() (metadata CatalogMetadata) {
	for _, v := range catalog.Versions {
		if !util.StringInSlice(metadata.Versions, v.Version) {
			metadata.Versions = append(metadata.Versions, v.Version)
		}

		for _, p := range v.Providers {
			if !util.StringInSlice(metadata.Providers, p.Name) {
				metadata.Providers = append(metadata.Providers, p.Name)
			}
			if !util.StringInSlice(metadata.ChecksumTypes, p.ChecksumType) {
				metadata.ChecksumTypes = append(metadata.ChecksumTypes, p.ChecksumType)
			}
		}
	}

	return metadata
}

// Delete removes items in the catalog that match CatalogQueryParams
func (catalog *Catalog) Delete(params CatalogQueryParams) (result Catalog, err error) {
	var queryResult Catalog
	var deleteMetadata CatalogMetadata

	if params.Version == "" {
		result = *catalog
	} else {
		if queryResult, err = catalog.QueryCatalogVersions(params.Version); err != nil {
			return
		}
		fmt.Printf("qr: %v\n", queryResult)
		deleteMetadata = queryResult.Metadata()
		result = catalog.deleteBoxes(deleteMetadata.Versions, []string{})
	}

	if params.Provider != "" {
		if queryResult, err = result.QueryCatalogProviders(params.Provider); err != nil {
			return
		}
		deleteMetadata = queryResult.Metadata()
		result = result.deleteBoxes([]string{}, deleteMetadata.Providers)
	}

	return
}
