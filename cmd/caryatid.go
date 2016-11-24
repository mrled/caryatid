package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"strings"
)

func main() {
	extension := flag.String("extension", ".box", "The file extension of the artifact. Passing a file with any other extension will cause the script to fail. For use with packer and vagrant this must be 'box'. (This option is intended mostly for testing.)")
	testing := flag.Bool("testing", false, "Pass this flag to parse the command line but nothing else. Used in testing, which sees an os.Exit(0) and an exit value of anything else as a failure.")
	flag.Parse()

	cmdInvocation := os.Args[0]
	cmdName := path.Base(cmdInvocation)
	reqArgCount := 7
	allowedBackends := []string{"copy", "scp"}

	backendIsAllowed := func(testBackend string) bool {
		for _, allowedBackend := range allowedBackends { if testBackend == allowedBackend {return true} }
		return false
	}

	if len(flag.Args()) != reqArgCount {
		fmt.Println(fmt.Sprintf("%v args were required but %v args were passed", reqArgCount, len(flag.Args())))
		os.Exit(1)
	}

	var (
		name = flag.Arg(0)
		description = flag.Arg(1)
		version = flag.Arg(2)
		provider = flag.Arg(3)
		artifact = flag.Arg(4)
		backend = flag.Arg(5)
		destination = flag.Arg(6)
	)

	if !(backendIsAllowed(backend)) {
		fmt.Println(fmt.Sprintf("Passed a backend of '%v', which is not valid; backends must be one of '%v'", backend, allowedBackends))
		os.Exit(1)
	}
	if !(strings.HasSuffix(artifact, *extension)) {
		fmt.Println(fmt.Sprintf("Passed an artifact of '%v', which does not have the required extension '%v'", artifact, *extension))
		os.Exit(1)
	}

	fmt.Println("Command invocation:", cmdInvocation)
	fmt.Println("Command name:", cmdName)
	fmt.Println("Box name:", name)
	fmt.Println("Box description:", description)
	fmt.Println("Box version:", version)
	fmt.Println("Box provider:", provider)
	fmt.Println("Artifact path:", artifact)
	fmt.Println("Backend:", backend)
	fmt.Println("Destination directory:", destination)

	if !*testing {
		fmt.Println("We don't do anything yet, so exit here unless we're running the tests")
		os.Exit(1)
	}
}
