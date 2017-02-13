package main

import (
	"bytes"
	"fmt"
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

func main() {
	var (
		err                  error
		_, thisFile, _, rcOk = runtime.Caller(0)
		thisDir, _           = path.Split(thisFile)
		ppDir                = path.Join(thisDir, "packer-post-processor-caryatid")
		releaseDir           = path.Join(thisDir, "release")
	)

	if !rcOk {
		panic("Could not determine build script file path")
	}

	platforms := [][]string{
		[]string{"amd64", "darwin"},
		[]string{"amd64", "freebsd"},
		[]string{"386", "freebsd"},
		[]string{"arm", "freebsd"},
		[]string{"amd64", "linux"},
		[]string{"386", "linux"},
		[]string{"arm", "linux"},
		[]string{"amd64", "windows"},
		[]string{"386", "windows"},
	}

	for _, plat := range platforms {
		goarch := plat[0]
		goos := plat[1]
		platOutDir := path.Join(releaseDir, goos, goarch)

		filename := "packer-post-processor-caryatid"
		if goos == "windows" {
			filename = fmt.Sprintf("%v.exe", filename)
		}

		if err = os.MkdirAll(platOutDir, 0777); err != nil {
			panic(err)
		}

		cmd := exec.Command("go", "build", "-o", path.Join(platOutDir, filename))
		cmd.Dir = ppDir

		environment := os.Environ()
		environment, err = updateEnv(environment, "GOARCH", goarch)
		if err != nil {
			panic(err)
		}
		environment, err = updateEnv(environment, "GOOS", goos)
		if err != nil {
			panic(err)
		}
		cmd.Env = environment

		var stdout bytes.Buffer
		var stderr bytes.Buffer

		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			panic(fmt.Errorf("Error running command '%v':\nSTDOUT: %v\nSTDERR: %v\nGo error: %v\n", cmd, stdout.String(), stderr.String(), err))
		}
	}
}
