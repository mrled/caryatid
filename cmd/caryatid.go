package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"strings"
)

type ParsedArguments struct {
	CommandInvocation string
	CommandName       string
	BoxName           string
	BoxDescription    string
	BoxVersion        string
	BoxProvider       string
	ArtifactPath      string
	Backend           string
	Destination       string
}

func ParseArguments(arguments ...string) (parsed ParsedArguments, err error) {
	os.Args = arguments

	extension := flag.String("extension", ".box", "The file extension of the artifact. Passing a file with any other extension will cause the script to fail. For use with packer and vagrant this must be '.box'. (This option is intended mostly for testing.)")
	flag.Parse()

	reqArgCount := 7
	allowedBackends := []string{"copy", "scp"}

	backendIsAllowed := func(testBackend string) bool {
		for _, allowedBackend := range allowedBackends {
			if testBackend == allowedBackend {
				return true
			}
		}
		return false
	}

	if flag.NArg() != reqArgCount {
		err = errors.New(fmt.Sprintf("%v args were required but %v args were passed", reqArgCount, len(flag.Args())))
		return
	}

	parsed = ParsedArguments{os.Args[0], path.Base(os.Args[0]), flag.Arg(0), flag.Arg(1), flag.Arg(2), flag.Arg(3), flag.Arg(4), flag.Arg(5), flag.Arg(6)}
	fmt.Println(os.Args[0])
	fmt.Println(path.Base(os.Args[0]))

	if !(backendIsAllowed(parsed.Backend)) {
		err = errors.New(fmt.Sprintf("Passed a backend of '%v', which is not valid; backends must be one of '%v'", parsed.Backend, allowedBackends))
		return
	}
	if !(strings.HasSuffix(parsed.ArtifactPath, *extension)) {
		err = errors.New(fmt.Sprintf("Passed an artifact of '%v', which does not have the required extension '%v'", parsed.ArtifactPath, *extension))
		return
	}

	return
}

func main() {
	pa, err := ParseArguments(os.Args...)
	if err != nil {
		fmt.Println(fmt.Sprintf("ERROR: %v", err))
		os.Exit(1)
	}
	fmt.Println(pa)

	// if !*testing {
	// 	fmt.Println("We don't do anything yet, so exit here unless we're running the tests")
	// 	os.Exit(1)
	// }
}
