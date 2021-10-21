package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var (
	currDir = ""
)

// lookPath tries to find an absolute path to an executable
// name by searching directories on the PATH
// If the name is an absolute path or a shortened path (./)
// then this path is returned
func lookPath(name string) (string, error) {
	if filepath.IsAbs(name) { //if the user has absolute path then we good
		return name, nil
	}
	// see if the executable name has "./" or "../" in it
	if strings.Contains(name, "./") || strings.Contains(name, "../") {
		absPath := filepath.Join(currDir, name)
		_, err := os.Stat(absPath)
		if !os.IsNotExist(err) {
			return absPath, nil
		}
	}
	path := getenv("PATH")
	if path == "" {
		err := errors.New("executable not found")
		return "", err
	}
	directories := strings.Split(path, ":")
	for _, directory := range directories {
		dirInfo, err := os.ReadDir(directory)
		if err != nil {
			//quashError("%s : %s", errors.Unwrap(err), directory)
			continue
		}
		for _, file := range dirInfo {
			if file.Name() == name && !file.IsDir() {
				return directory + "/" + name, nil
			}
		}
	}
	err := errors.New("executable not found")
	return "", err
}
