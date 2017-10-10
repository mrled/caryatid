package main

import (
	"testing"
)

type IoPair struct {
	Input  string
	Output bool
}

func TestTestValidUri(t *testing.T) {
	ioPairs := []IoPair{
		IoPair{"/usr/local/bin", false},
		IoPair{"file://usr/local/bin", true},
		IoPair{"file:///usr/local/bin", true},
		IoPair{"http://google.com/evil.txt", true},
		IoPair{"s3://bucket_name/some/key/value", true},
	}

	for _, pair := range ioPairs {
		if testValidUri(pair.Input) != pair.Output {
			t.Fatalf("testValidUri('%v') did not return suspected value of '%v'", pair.Input, pair.Output)
		}
	}
}

func TestGetManager(t *testing.T) {
	ioPairs := []IoPair{
		IoPair{"file://whatever", true},
		IoPair{"file:///whatever", true},
		IoPair{"whatever", true},
		IoPair{"invalidbackend://whatever", false},
	}

	for _, pair := range ioPairs {
		_, err := getManager(pair.Input)
		if err == nil {
			if pair.Output == false {
				t.Fatalf("Getting a manager with input URI '%v' succeeded, but should have failed", pair.Input)
			}
		} else if err != nil {
			if pair.Output == true {
				t.Fatalf("Getting a manager with input URI '%v' should have succeeded, but failed with error: %v", pair.Input, err)
			}
		}
	}
}
