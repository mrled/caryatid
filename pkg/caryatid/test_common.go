/*
Common functions used for testing that need to exist outside of individual *_test.go files
*/

package caryatid

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"log"
	"os"
)

func CreateTestBoxFile(filePath string, providerName string, compress bool) (err error) {
	outFile, err := os.Create(filePath)
	if err != nil {
		fmt.Printf("Error trying to create the test box file at '%v': %v\n", filePath, err)
		return
	}
	defer outFile.Close()

	var tarWriter *tar.Writer
	if compress {
		gzipWriter := gzip.NewWriter(outFile)
		defer gzipWriter.Close()
		tarWriter = tar.NewWriter(gzipWriter)
	} else {
		tarWriter = tar.NewWriter(outFile)
	}
	defer tarWriter.Close()

	metaDataContents := fmt.Sprintf(`{"provider": "%v"}`, providerName)
	header := &tar.Header{
		Name: "metadata.json",
		Mode: 0666,
		Size: int64(len(metaDataContents)),
	}

	if err = tarWriter.WriteHeader(header); err != nil {
		fmt.Printf("Error trying to write the header for the test box file: %v\n", err)
		return
	}
	if _, err = tarWriter.Write([]byte(metaDataContents)); err != nil {
		fmt.Printf("Error trying to write metadata contents for the test box file: %v\n", err)
		return
	}
	return
}

type CatalogFuzzyEqualsParams struct {
	SkipName                 bool
	SkipDescription          bool
	SkipVersions             bool
	SkipVersionString        bool
	SkipProviders            bool
	SkipProviderName         bool
	SkipProviderUrl          bool
	SkipProviderChecksumType bool
	SkipProviderChecksum     bool
	LogMismatch              bool
}

// FuzzyEquals tests whether two Catalogs are equal, but allows skipping comparison of any property via CatalogFuzzyEqualsParams
func (c1 *Catalog) FuzzyEquals(c2 *Catalog, params CatalogFuzzyEqualsParams) bool {
	logMismatch := func(property string) {
		if params.LogMismatch {
			log.Printf("FuzzyEquals() for '%v ?= %v' Catalog failed to match property '%v'\n", c1.Name, c2.Name, property)
		}
	}

	if !params.SkipName && c1.Name != c2.Name {
		logMismatch("Name")
		return false
	}
	if !params.SkipDescription && c1.Description != c2.Description {
		logMismatch("Description")
		return false
	}
	if !params.SkipVersions == false {
		return true
	} else if len(c1.Versions) != len(c2.Versions) {
		logMismatch("Versions")
		return false
	}

	for idx, v1 := range c1.Versions {
		v2 := c2.Versions[idx]
		if !params.SkipVersionString && v1.Version != v2.Version {
			logMismatch("VersionString")
			return false
		}
		if !params.SkipProviders == false {
			continue
		} else if len(v1.Providers) != len(v2.Providers) {
			logMismatch("Providers")
			return false
		}

		for idx, p1 := range v1.Providers {
			p2 := v2.Providers[idx]
			if !params.SkipProviderName && p1.Name != p2.Name {
				logMismatch("ProviderName")
				return false
			}
			if !params.SkipProviderUrl && p1.Url != p2.Url {
				logMismatch("ProviderUrl")
				return false
			}
			if !params.SkipProviderChecksumType && p1.ChecksumType != p2.ChecksumType {
				logMismatch("ProviderChecksumType")
				return false
			}
			if !params.SkipProviderChecksum && p1.Checksum != p2.Checksum {
				logMismatch("ProviderChecksum")
				return false
			}
		}
	}

	return true
}
