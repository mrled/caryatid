/*
This package contains generic functions used by caryatid. It should not be used for functions that contain caryatid specific logic.
*/

package util

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"os"
)

// PathExists tests whether path exists
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

// Sha1sum returns the SHA1 hash for a file on the filesystem
func Sha1sum(filePath string) (result string, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer file.Close()

	hash := sha1.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		return
	}

	result = hex.EncodeToString(hash.Sum(nil))
	return
}

// CopyFile copies a file
func CopyFile(src string, dst string) (written int64, err error) {
	in, err := os.Open(src)
	defer in.Close()
	if err != nil {
		return
	}
	out, err := os.Create(dst)
	defer out.Close()
	if err != nil {
		return
	}
	written, err = io.Copy(out, in)
	if err != nil {
		return
	}
	err = out.Close()
	return
}

// StringInSlice tests whether a string is in a slice of strings
func StringInSlice(slice []string, str string) bool {
	for _, item := range slice {
		if str == item {
			return true
		}
	}
	return false
}
