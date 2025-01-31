package elevutils

import (
	"testing"
)

func TestGetGitHash(t *testing.T) {
	if GetGitHash() == "" {
		t.Errorf("GetGitHash() = %s, expected a non-empty string", GetGitHash())
	}
}
