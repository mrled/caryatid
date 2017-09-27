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

// assembleZip creates a separate zipfile for each supported platform for packer-post-processor-caryatid
// DEPRECATED: This was written back when I just had one command in the root of my repo, and no longer works
// TODO: rework this into something that can build the entire source tree
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
	environment = updateEnv(environment, "GOARCH", goarch)
	environment = updateEnv(environment, "GOOS", goos)
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

// goBuildCmd builds a single "cmd" project
func goBuildCmd(projectRoot string, cmdName string, outDir string, plat platform) (err error) {
	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)
	if projectRoot == "" {
		return fmt.Errorf("Missing required parameter projectRoot")
	}
	if cmdName == "" {
		return fmt.Errorf("Missing required parameter cmdName")
	}
	projectDir := path.Join(projectRoot, "cmd", cmdName)
	if outDir == "" {
		outDir = projectDir
	}

	cmdFilename := cmdName
	if plat.Os == "windows" {
		cmdFilename = fmt.Sprintf("%v.exe", cmdName)
	}
	cmdOutPath := path.Join(outDir, cmdFilename)

	environment := os.Environ()
	environment = updateEnv(environment, "GOARCH", plat.Arch)
	environment = updateEnv(environment, "GOOS", plat.Os)

	cmd := exec.Command("go", "build", "-o", cmdOutPath)
	cmd.Dir = projectDir
	cmd.Env = environment
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		errmsg := strings.Join(
			[]string{
				fmt.Sprintf("Error running 'go build' for %v/%v platform:", plat.Os, plat.Arch),
				"STDOUT:", stdout.String(),
				"STDERR:", stderr.String(),
				"Go error:", fmt.Sprintf("%v", err),
			}, "\n",
		)
		return fmt.Errorf("%v\n", errmsg)
	}
	return
}

type platform struct {
	Os   string
	Arch string
}

func (plat *platform) String() string {
	return fmt.Sprintf("%v/%v", plat.Os, plat.Arch)
}

func myPlatform() platform {
	return platform{runtime.GOOS, runtime.GOARCH}
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

// readDirFullPath takes in path components and returns the absolute path of the input path's children
func readDirFullPath(pathComponents ...string) (fullPaths []string, err error) {
	basePath := path.Join(pathComponents...)
	pathSubItems, err := ioutil.ReadDir(basePath)
	if err != nil {
		return
	}
	for _, subItem := range pathSubItems {
		fullPaths = append(fullPaths, path.Join(basePath, subItem.Name()))
	}
	return
}

func main() {
	var (
		err    error
		outDir string

		_, thisFile, _, rcOk = runtime.Caller(0)
		thisDir              = filepath.Dir(thisFile)
		projectRootDir       = filepath.Dir(thisDir)
		releaseDir           = path.Join(projectRootDir, "release")

		// cmdProjs, _      = ioutil.ReadDir(path.Join(projectRootDir, "cmd"))
		// internalProjs, _ = ioutil.ReadDir(path.Join(projectRootDir, "internal"))
		// pkgProjs, _      = ioutil.ReadDir(path.Join(projectRootDir, "pkg"))
	)

	if !rcOk {
		panic("Could not determine build script file path")
	}

	actionFlag := flag.String("action", "build", "The action to perform. One of build, test, release")
	outDirFlag := flag.String("outDir", "", "The output directory. If empty, binaries will be built in their project directories.")
	versionFlag := flag.String("version", "devel", "A version number")
	flag.Parse()

	outDir = *outDirFlag
	if outDir != "" {
		if outDir, err = filepath.Abs(outDir); err != nil {
			fmt.Printf("Tried to set the outDir to '%v', but could not determine its absolute path. Building all binaries in their respective project directories instead.\n", outDirFlag)
			outDir = ""
		}
	}

	fmt.Printf("thisDir: %v\nprojectRootDir: %v\n", thisDir, projectRootDir)
	fmt.Printf("Performing action: %v\n", *actionFlag)
	switch *actionFlag {
	case "build":
		err = goBuildCmd(projectRootDir, "caryatid", outDir, myPlatform())
		if err != nil {
			panic(err)
		}
		err = goBuildCmd(projectRootDir, "packer-post-processor-caryatid", outDir, myPlatform())
		if err != nil {
			panic(err)
		}
		fmt.Printf("Successfully built all projects under cmd/\n")
		if outDir == "" {
			fmt.Printf("All files output to their respective project directories\n")
		} else {
			fmt.Printf("All files output to '%v'\n", outDir)
		}
	case "test":
		panic("-action test NOT IMPLEMENTED")
	case "release":
		panic("-action release NOT IMPLEMENTED")
	}
}
