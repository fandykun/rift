package main

import (
	"strings"
	"testing"
)

func TestVersionIncludesBuildCommit(t *testing.T) {
	previousVersion := version
	previousCommit := buildCommit
	version = "1.2.3-test"
	buildCommit = "abc1234"
	t.Cleanup(func() {
		version = previousVersion
		buildCommit = previousCommit
	})

	cmd := newRootCommand()
	if got := cmd.Version; got != "1.2.3-test (commit abc1234)" {
		t.Fatalf("Version = %q", got)
	}
}

func TestRootCommandVersionOutput(t *testing.T) {
	previousVersion := version
	previousCommit := buildCommit
	version = "1.2.3-test"
	buildCommit = "abc1234"
	t.Cleanup(func() {
		version = previousVersion
		buildCommit = previousCommit
	})

	cmd := newRootCommand()
	output := &strings.Builder{}
	cmd.SetOut(output)
	cmd.SetErr(output)
	cmd.SetArgs([]string{"--version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := output.String(); !strings.Contains(got, "rift version 1.2.3-test (commit abc1234)") {
		t.Fatalf("version output = %q", got)
	}
}
