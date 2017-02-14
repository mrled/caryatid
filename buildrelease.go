package main

import (
	"archive/zip"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
)

// Build a release for all supported architectures
// Run with "go run buildrelease.go"

func updateEnv(inEnv []string, name string, value string) (outEnv []string, err error) {
	if name == "" || value == "" {
		err = fmt.Errorf("Input name (%v) or value (%v) was empty", name, value)
	}
	for _, variable := range inEnv {
		var (
			split     = strings.Split(variable, "=")
			thisName  = split[0]
			thisValue = split[1]
		)
		if thisName == "" || thisValue == "" {
			err = fmt.Errorf("Couldn't find name (%v) or value (%v)", thisName, thisValue)
			return
		}
		if name != thisName {
			outEnv = append(outEnv, variable)
		}
	}
	outEnv = append(outEnv, fmt.Sprintf("%v=%v", name, value))
	return
}

func getTempFilePath(directory string) (tempFilePath string, err error) {
	tempFile, err := ioutil.TempFile(directory, "")
	if err != nil {
		return
	}
	tempFilePath = tempFile.Name()
	tempFile.Close()
	err = os.Remove(tempFilePath)
	return
}

func assembleZip(goos string, goarch string, thisDir string, zipOutDir string, zipBaseName string) (err error) {
	zipOutPath := path.Join(zipOutDir, fmt.Sprintf("%v.zip", zipBaseName))
	srcDir := path.Join(thisDir, "packer-post-processor-caryatid")
	readmePath := path.Join(thisDir, "readme.markdown")

	exeName := "packer-post-processor-caryatid"
	if goos == "windows" {
		exeName = fmt.Sprintf("%v.exe", exeName)
	}

	tmpExeName, err := getTempFilePath(thisDir)
	if err != nil {
		return
	}

	cmd := exec.Command("go", "build", "-o", tmpExeName)
	cmd.Dir = srcDir

	environment := os.Environ()
	environment, err = updateEnv(environment, "GOARCH", goarch)
	if err != nil {
		return
	}
	environment, err = updateEnv(environment, "GOOS", goos)
	if err != nil {
		return
	}
	cmd.Env = environment

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	fmt.Printf("Building %v/%v binary\n", goos, goarch)
	err = cmd.Run()
	defer os.Remove(tmpExeName)
	if err != nil {
		return fmt.Errorf("Error running command '%v':\nSTDOUT: %v\nSTDERR: %v\nGo error: %v\n", cmd, stdout.String(), stderr.String(), err)
	}

	fmt.Printf("Creating zipfile for %v/%v binary at %v\n", goos, goarch, zipOutPath)
	zipOutFile, err := os.Create(zipOutPath)
	defer zipOutFile.Close()
	if err != nil {
		return
	}
	zipWriter := zip.NewWriter(zipOutFile)
	defer zipWriter.Close()

	zExeFile, err := zipWriter.Create(fmt.Sprintf("%v/%v", zipBaseName, exeName))
	if err != nil {
		return
	}

	fsExeFile, err := os.Open(tmpExeName)
	defer fsExeFile.Close()
	if err != nil {
		return
	}

	_, err = io.Copy(zExeFile, fsExeFile)
	if err != nil {
		return
	}

	zReadmeFile, err := zipWriter.Create(fmt.Sprintf("%v/readme.markdown", zipBaseName))
	if err != nil {
		return
	}

	fsReadmeFile, err := os.Open(readmePath)
	defer fsReadmeFile.Close()
	if err != nil {
		return
	}

	_, err = io.Copy(zReadmeFile, fsReadmeFile)
	if err != nil {
		return
	}

	return
}

func main() {
	var (
		err                  error
		_, thisFile, _, rcOk = runtime.Caller(0)
		thisDir, _           = path.Split(thisFile)
		releaseDir           = path.Join(thisDir, "release")
	)

	if !rcOk {
		panic("Could not determine build script file path")
	}

	versionFlag := flag.String("version", "devel", "A version number")
	flag.Parse()

	platforms := [][]string{
		[]string{"darwin", "amd64"},
		[]string{"freebsd", "amd64"},
		[]string{"freebsd", "386"},
		[]string{"freebsd", "arm"},
		[]string{"linux", "amd64"},
		[]string{"linux", "386"},
		[]string{"linux", "arm"},
		[]string{"windows", "amd64"},
		[]string{"windows", "386"},
	}

	err = os.MkdirAll(releaseDir, 0777)
	if err != nil {
		panic(err)
	}

	for _, plat := range platforms {
		goos := plat[0]
		goarch := plat[1]
		zipBaseName := fmt.Sprintf("caryatid_%v_%v_%v", goos, goarch, *versionFlag)

		err = assembleZip(goos, goarch, thisDir, releaseDir, zipBaseName)
		if err != nil {
			fmt.Printf("Error assembling zip file for %v/%v: %v", goos, goarch, err)
		}
	}
}
