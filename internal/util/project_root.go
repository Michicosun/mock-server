package util

import (
	"os"
	"os/exec"
	"path"
)

func GetProjectRoot() (string, error) {
	dir, set := os.LookupEnv("WORKING_DIRECTORY")
	if set {
		return path.Dir(dir), nil
	}

	cmd := exec.Command("go", "env", "GOMOD")
	stdout, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return path.Dir(string(stdout)), nil
}
