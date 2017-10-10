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

func BoxUriFromCatalogUri(catalogUri string, name string, version string, provider string) (boxUri string, err error) {
	lastSlashIdx := strings.LastIndex(catalogUri, "/")
	if lastSlashIdx < 0 {
		err = fmt.Errorf("Invalid URI: %v\n", catalogUri)
		return
	}
	catalogParentUri := catalogUri[0:lastSlashIdx]
	boxUri = fmt.Sprintf("%v/%v/%v_%v_%v.box", catalogParentUri, name, name, version, provider)
	return
}

// AddBox updates the Catalog to include a new box file
// The artifact's Name must match the Catalog's Name, if the Catalog already exists in storage
// However, the artifact's Description always overwrites the Catalog's Description, even if they are different
// This minimizes painful end-of-build errors,
// and lets the user change their mind about the wording of the description
func (c *Catalog) AddBox(catalogUri string, name string, description string, version string, provider string, checksumType string, checksum string) (err error) {
	if c.Name != "" && name != "" && c.Name != name {
		err = fmt.Errorf("Catalog.AddBox(): Catalog name '%v' does not match input name '%v'\n", c.Name, name)
		return
	} else if name != "" && c.Name == "" {
		c.Name = name
	}
	if c.Name == "" {
		err = fmt.Errorf("Catalog.AddBox(): Catalog name could not be determined\n")
	}

	c.Description = description

	boxUri, err := BoxUriFromCatalogUri(catalogUri, name, version, provider)
	if err != nil {
		return
	}

	newProvider := Provider{provider, boxUri, checksumType, checksum}
	newVersion := Version{version, []Provider{newProvider}}

	foundVersion := false
	foundProvider := false

	for vidx, _ := range c.Versions {
		if c.Versions[vidx].Version == version {
			foundVersion = true
			for pidx, _ := range c.Versions[vidx].Providers {
				if c.Versions[vidx].Providers[pidx].Name == provider {
					c.Versions[vidx].Providers[pidx].Url = boxUri
					c.Versions[vidx].Providers[pidx].ChecksumType = checksumType
					c.Versions[vidx].Providers[pidx].Checksum = checksum
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
// The string must be a valid semantic version string, optionally preceded by a qualifier - one of < > <= or >=
// If a semver doesn't have a qualifier, such as "1.0.0", return BOTH VersionEquals and VersionEqualsPrereleaseMismatch
// However, if the semver has an equals qualifier, like "=1.0.0", return ONLY VersionEquals
func parseVersionQueryString(semver string) (version ComparableVersion, qualifier VersionComparatorList, err error) {
	if len(semver) == 0 {
		return
	}

	// WARNING: The two-character prefixes must come first!
	for _, prefix := range []string{">=", "<=", ">", "<", "="} {
		if strings.HasPrefix(semver, prefix) {
			if qualifier, err = NewVersionComparator(prefix); err != nil {
				return
			}
			if version, err = NewComparableVersion(semver[len(prefix):]); err != nil {
				log.Printf("FAILING HERE: %v, %v, %v\n%v\n", semver, prefix, qualifier, err)
				return
			}
			break
		}
	}
	if len(qualifier) == 0 {
		qualifier = VersionComparatorList{VersionEquals, VersionEqualsPrereleaseMismatch}
		if version, err = NewComparableVersion(semver); err != nil {
			return
		}
	}

	return
}

// CatalogQueryParams represents valid parameters for QueryCatalog(), below
type CatalogQueryParams struct {
	Version  string
	Provider string
}

// QueryCatalogVersions returns a new Catalog containing only Versions match the versionquery input string
// If the caller has provided an *exact* version like "=1.0.0",
// assume they do NOT want to find prerelease-mismatched versions;
// If the caller has provided a version *range* like "<=1.0.0",
// assume they DO want to find prerelease-mismatched versions.
func (catalog *Catalog) QueryCatalogVersions(versionquery string) (result Catalog, err error) {
	var (
		comparator VersionComparator
		cVers      ComparableVersion
		queryVers  ComparableVersion
		queryQual  VersionComparatorList
	)
	result.Name = catalog.Name
	result.Description = catalog.Description
	if queryVers, queryQual, err = parseVersionQueryString(versionquery); err != nil {
		return
	} else if len(queryVers.Version) == 0 {
		result = *catalog
		return
	}

	for _, version := range catalog.Versions {
		if cVers, err = NewComparableVersion(version.Version); err != nil {
			return
		}
		comparator = cVers.Compare(&queryVers)

		if queryQual.Contains(VersionComparatorList{comparator}) {
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

type BoxReference struct {
	Version      string
	ProviderName string
	Uri          string
}

// Compare the two key fields of a BoxReference: Version and ProviderName
// Within a given Catalog, these two values should be enough to uniquely identify a box
func (br1 *BoxReference) Equals(br2 BoxReference) bool {
	return br1.Version == br2.Version && br1.ProviderName == br2.ProviderName
}

type BoxReferenceList []BoxReference

func (list BoxReferenceList) Contains(br BoxReference) bool {
	for _, listItem := range list {
		if listItem.Equals(br) {
			return true
		}
	}
	return false
}

func (catalog *Catalog) BoxReferences() (result BoxReferenceList) {
	for _, v := range catalog.Versions {
		for _, p := range v.Providers {
			result = append(result, BoxReference{Version: v.Version, ProviderName: p.Name, Uri: p.Url})
		}
	}

	return
}

func (catalog *Catalog) DeleteReferences(references BoxReferenceList) (result Catalog) {
	result.Name = catalog.Name
	result.Description = catalog.Description

	for _, v := range catalog.Versions {
		newVersion := Version{Version: v.Version, Providers: []Provider{}}
		for _, p := range v.Providers {
			thisBox := BoxReference{Version: v.Version, ProviderName: p.Name}
			if !references.Contains(thisBox) {
				newVersion.Providers = append(newVersion.Providers, p)
			}

		}
		if len(newVersion.Providers) > 0 {
			result.Versions = append(result.Versions, newVersion)
		}
	}
	return

}

// TODO: Consider refactoring / removing
// Currently this is only used in tests, while the BackendManager has to call QueryCatalog() and DeleteReferences() itself
func (catalog *Catalog) DeleteQuery(param CatalogQueryParams) (result Catalog, err error) {
	var (
		deleteCatalog Catalog
	)

	if deleteCatalog, err = catalog.QueryCatalog(param); err != nil {
		return
	}

	refs := deleteCatalog.BoxReferences()
	result = catalog.DeleteReferences(refs)
	return
}
