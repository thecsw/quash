package main

import (
	"os"
	"strings"
)

var (
	// myEnv is the environment that we keep in quash
	myEnv = os.Environ()
)

// getenv gets an env value from myEnv
func getenv(key string) string {
	// Try to find and replace
	for _, env := range myEnv {
		parts := strings.Split(env, "=")
		if parts[0] == key {
			return parts[1]
		}
	}
	// Not found
	return ""
}

// setenv sets an env by key in myEnv
func setenv(key, value string) {
	entry := key + "=" + value
	// Try to find and replace
	for ind, env := range myEnv {
		parts := strings.Split(env, "=")
		if parts[0] == key {
			myEnv[ind] = entry
			return
		}
	}
	// If not found, append
	myEnv = append(myEnv, entry)
}
