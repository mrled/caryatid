/*
Common functions used for testing that need to exist outside of individual *_test.go files
*/

package caryatid

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
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
