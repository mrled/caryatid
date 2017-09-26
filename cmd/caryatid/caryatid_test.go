package main

import (
	"testing"
)

func TestStrArrayContains(t *testing.T) {
	arr := []string{"one", "two", "three", "four"}
	if !strArrayContains(arr, "one") {
		t.Fatal("strArrayContains() failed to find existing item in array")
	}
	if strArrayContains(arr, "on") {
		t.Fatal("strArrayContains() incorrectly found a match on just a partial string")
	}
	if strArrayContains(arr, "zxcv") {
		t.Fatal("strArrayContains() incorrectly found a match when it should not have")
	}
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

func TestStrEnsureArrayContainsAll(t *testing.T) {
	refArray := []string{"one", "two", "three", "four", "five", "six", "seven"}
	panicFmtStr := "Missing item: %v"
	recoverFromPanic := func() {
		_ = recover()
	}

	shouldPass := func() bool {
		defer recoverFromPanic()
		mustContain := []string{"five", "one"}
		strEnsureArrayContainsAll(refArray, mustContain, panicFmtStr)
		return true
	}
	if !shouldPass() {
		t.Fatalf("strEnsureArrayContainsAll() failed to find a match when it should")
	}

	shouldPanic := func() (passed bool) {
		defer recoverFromPanic()
		passed = false
		mustContain := []string{"one", "alpha", "five"}
		strEnsureArrayContainsAll(refArray, mustContain, panicFmtStr)
		return true
	}
	if shouldPanic() {
		t.Fatalf("strEnsureArrayContainsAll() incorrectly found a match when it should not have")
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
		_, err := getManager(pair.Input, "TestBoxName")
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
