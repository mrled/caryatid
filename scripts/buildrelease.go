/*
Caryatid build script

Goals:
- Build single-platform binaries
- Build separate binaries for each supported architecture
- Assemble zipfiles for each supported architecture for release

Run with "go run scripts/buildrelease.go"
*/

package main

import (
	"archive/zip"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

func updateEnv(inEnv []string, name string, value string) (outEnv []string) {
	for _, entry := range inEnv {
		eqIdx := strings.Index(entry, "=")
		if eqIdx != -1 && entry[0:eqIdx] != name {
			outEnv = append(outEnv, entry)
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

// copyFileToZip copies a file on the filesystem inside an archive represented by zipWriter
func copyFileToZip(zipWriter *zip.Writer, fsPath string, zipPath string) (err error) {
	zippedFileHandle, err := zipWriter.Create(zipPath)
	if err != nil {
		return err
	}

	fsFileHandle, err := os.Open(fsPath)
	defer fsFileHandle.Close()
	if err != nil {
		return
	}

	if _, err = io.Copy(zippedFileHandle, fsFileHandle); err != nil {
		return
	}

	return
}

// readDirFullPath takes in path components and returns the absolute path of the input path's children
func readDirFullPath(pathComponents ...string) (fullPaths []string, err error) {
	basePath := path.Join(pathComponents...)
	pathSubItems, err := ioutil.ReadDir(basePath)
	if err != nil {
		return
	}
	for _, subItem := range pathSubItems {
		p := path.Join(basePath, subItem.Name())
		// fmt.Printf("- %v\n", p)
		fullPaths = append(fullPaths, p)
	}
	return
}

// execGo executes go
func execGo(arguments []string, environment []string, pwd string) (err error) {
	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)
	cmd := exec.Command("go", arguments...)
	cmd.Dir = pwd
	cmd.Env = environment
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		errmsg := strings.Join(
			[]string{
				fmt.Sprintf("Error running 'go %v'", arguments),
				"Environment:", strings.Join(environment, "\n"),
				"STDOUT:", stdout.String(),
				"STDERR:", stderr.String(),
				"Go error:", fmt.Sprintf("%v", err),
			}, "\n",
		)
		return fmt.Errorf("%v\n", errmsg)
	}
	return
}

// assembleZip creates a separate zipfile for each supported platform
func assembleZip(projectRoot string, zipOutDir string, version string) (err error) {

	cmdDirList, err := readDirFullPath(projectRoot, "cmd")
	if err != nil {
		return err
	}

	for _, plat := range allPlatforms {
		projectName := path.Base(projectRoot)
		zipBaseName := fmt.Sprintf("%v_%v_%v_%v", projectName, plat.Os, plat.Arch, version)
		zipOutPath := path.Join(zipOutDir, fmt.Sprintf("%v.zip", zipBaseName))

		zipOutFile, err := os.Create(zipOutPath)
		defer zipOutFile.Close()
		if err != nil {
			return err
		}
		zipWriter := zip.NewWriter(zipOutFile)
		defer zipWriter.Close()

		copyFileToZip(zipWriter, path.Join(projectRoot, "readme.markdown"), fmt.Sprintf("%v/readme.markdown", zipBaseName))

		for _, cmdDir := range cmdDirList {
			cmdName := path.Base(cmdDir)
			if plat.Os == "windows" {
				cmdName = fmt.Sprintf("%v.exe", cmdName)
			}
			log.Printf("Building %v for %v...\n", cmdName, plat.String())

			tempBuildOutputFile, err := getTempFilePath(projectRoot)
			if err != nil {
				return err
			}

			err = execGo([]string{"build", "-o", tempBuildOutputFile}, plat.GetEnv(), cmdDir)
			defer os.Remove(tempBuildOutputFile)
			if err != nil {
				return err
			}

			if err = copyFileToZip(zipWriter, tempBuildOutputFile, fmt.Sprintf("%v/%v", zipBaseName, cmdName)); err != nil {
				return err
			}
		}
	}

	return
}

// platform represents an operating system / processor architecture pair
type platform struct {
	Os   string
	Arch string
}

// String() returns a human-readable string for a platform
func (plat *platform) String() string {
	return fmt.Sprintf("%v/%v", plat.Os, plat.Arch)
}

// GetEnv() returns os.Environ() + GOOS and GOARCH based on its Os and Arch properties
func (plat *platform) GetEnv() (outEnv []string) {
	return updateEnv(updateEnv(os.Environ(), "GOOS", plat.Os), "GOARCH", plat.Arch)
}

var allPlatforms = []platform{
	platform{"darwin", "amd64"},
	platform{"freebsd", "amd64"},
	platform{"freebsd", "386"},
	platform{"freebsd", "arm"},
	platform{"linux", "amd64"},
	platform{"linux", "386"},
	platform{"linux", "arm"},
	platform{"windows", "amd64"},
	platform{"windows", "386"},
}

func main() {
	var (
		err    error
		outDir string

		_, thisFile, _, rcOk = runtime.Caller(0)
		thisDir              = filepath.Dir(thisFile)
		projectRootDir       = filepath.Dir(thisDir)
	)

	if !rcOk {
		panic("Could not determine build script file path")
	}

	outDirFlag := flag.String("outDir", path.Join(projectRootDir, "release"), "The output directory.")
	versionFlag := flag.String("version", "devel", "A version number")
	flag.Parse()

	if outDir, err = filepath.Abs(*outDirFlag); err != nil {
		panic(err)
	}
	if err = os.MkdirAll(outDir, 0700); err != nil {
		panic(err)
	}

	log.Printf("Project root directory: %v\n", projectRootDir)
	log.Printf("Output directory: %v\n", outDir)

	if err = assembleZip(projectRootDir, outDir, *versionFlag); err != nil {
		panic(err)
	}
}
