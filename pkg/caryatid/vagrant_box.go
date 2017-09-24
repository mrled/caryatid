package caryatid

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

	"github.com/hashicorp/packer/packer"
	"github.com/mrled/caryatid/internal/util"
)

// Determine the provider of a Vagrant box based on its metadata.json
// See also https://www.packer.io/docs/post-processors/vagrant.html
func DetermineProvider(boxFilePath string) (result string, err error) {
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

func DeriveArtifactInfoFromBoxFile(boxFile string) (digest string, provider string, err error) {
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

	provider, err = DetermineProvider(boxFile)
	if err != nil {
		log.Printf("Could not determine provider from the filename for box file '%v'; got error %v\n", boxFile, err)
		return
	}
	log.Println(fmt.Sprintf("Determined provider as '%v'", provider))

	return
}

// func DerivePackerArtifactInfo(artifact packer.Artifact) (boxFile string, digest string, provider string, err error) {
func DeriveArtifactInfoFromPackerArtifact(artifact packer.Artifact) (boxFile string, digest string, provider string, err error) {
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

	digest, provider, err = DeriveArtifactInfoFromBoxFile(boxFile)
	return
}
