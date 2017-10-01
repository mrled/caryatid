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

// TODO: Add a func (*Catalog) String() string for use in the cli caryatid tool

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

// QueryCatalog returns matching BoxArtifact structs from a Catalog based on a CatalogQueryParams input query
func (catalog *Catalog) QueryCatalog(params CatalogQueryParams) (boxes []BoxArtifact) {
	var (
		err              error
		matchingVersions []Version
		newCatalog       Catalog
	)
	if params.Version == "" {
		matchingVersions = catalog.Versions
	} else {
		newCatalog, err = catalog.QueryCatalogVersions(params.Version)
		matchingVersions = newCatalog.Versions
		if err != nil {
			fmt.Printf("Invalid version query '%v' resulted in error '%v'; will return results for *all* versions\n", params.Version, err)
			matchingVersions = catalog.Versions
		}
	}

	if params.Provider == "" {
		params.Provider = ".*"
	}
	providerRegex := regexp.MustCompile(params.Provider)
	for _, version := range matchingVersions {
		for _, provider := range version.Providers {

			// TODO: should the regex matching be in a function like version.QueryVersionProviders instead?
			if providerRegex.Match([]byte(provider.Name)) {

				// TODO: is BoxArtifact the appropriate return type for this function?
				// Showing information to the user was not its intended purpose
				// Consider building up a new Catalog,
				// then using Catalog.String() or similar to display info to the user
				box := BoxArtifact{
					provider.Url, // Not quite right; this is supposed to be a local path, not a URI
					catalog.Name,
					catalog.Description,
					version.Version,
					provider.Name,
					"", // CatalogRootUri is useless and unknowable from here
					provider.ChecksumType,
					provider.Checksum,
				}
				boxes = append(boxes, box)

			}

		}
	}

	return
}
