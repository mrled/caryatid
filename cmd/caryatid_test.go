package main

import (
	"fmt"
	"os"
	"testing"
)

func TestMain(t *testing.T) {
	fmt.Println("Hiiiiiii")

	// NOTE: running it without the -testing=true flag should fail
	os.Args = []string{"/dummy/path/to/executable", "-testing=true", "1", "2", "3", "4", "5.box", "copy", "7"}
	main()
}
