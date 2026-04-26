package version

import "testing"

func TestDefaultVersionValues(t *testing.T) {
	if Version == "" {
		t.Error("Version should not be empty")
	}
	if GitCommit == "" {
		t.Error("GitCommit should not be empty")
	}
}

func TestVersionIsDevByDefault(t *testing.T) {
	if Version != "dev" {
		t.Skipf("Version is %q (overridden via ldflags), skipping default check", Version)
	}
}

func TestGitCommitIsUnknownByDefault(t *testing.T) {
	if GitCommit != "unknown" {
		t.Skipf("GitCommit is %q (overridden via ldflags), skipping default check", GitCommit)
	}
}
