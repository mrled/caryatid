package main

import (
	"testing"
)

func TestBoxArtifactEquals(t *testing.T) {
	a1 := BoxArtifact{
		"/tmp/path/to/example.box",
		"ExampleBox",
		"ExampleBox description",
		"192.168.0.1",
		"ExampleProvider",
		"http://example.com/Artifact",
		"sha1",
		"0xDECAFBAD",
	}
	a2 := BoxArtifact{
		"/tmp/path/to/example.box",
		"ExampleBox",
		"ExampleBox description",
		"192.168.0.1",
		"ExampleProvider",
		"http://example.com/Artifact",
		"sha1",
		"0xDECAFBAD",
	}
	a3 := BoxArtifact{
		"DIFFERENT/tmp/path/to/example.box",
		"DIFFERENTExampleBox",
		"DIFFERENTExampleBox description",
		"DIFFERENT192.168.0.1",
		"DIFFERENTExampleProvider",
		"DIFFERENThttp://example.com/Artifact",
		"DIFFERENTsha1",
		"DIFFERENT0xDECAFBAD",
	}
	if !a1.Equals(&a2) {
		t.Fatal("Artifacts expected to be the same did not match")
	}
	if a1.Equals(&a3) {
		t.Fatal("Artifacts expected to differ matched")
	}

}
