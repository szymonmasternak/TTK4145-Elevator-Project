package elevutils

import (
	_ "embed"
)

//go:generate sh -c "printf %s $(git rev-parse HEAD) > githash.txt"
//go:embed githash.txt
var gitHash string

func GetGitHash() string {
	return gitHash
}
