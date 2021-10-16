package main

import (
	"os"
	"regexp"
	"strings"
)

var (
	// myEnv is the environment that we keep in quash
	myEnv = os.Environ()

	envVarRegex = regexp.MustCompile(`({\$([^\s]+)}|\$([^\s]+))`)
)

func expandEnv(input string) string {
	anyFound := envVarRegex.MatchString(input)
	// if no env vars are found, return the same thing
	if !anyFound {
		return input
	}
	allMatches := envVarRegex.FindAllStringSubmatch(input, -1)
	for _, match := range allMatches {
		// Find the correct regex group match
		foundVar := match[2]
		if foundVar == "" {
			foundVar = match[3]
		}
		foundVal := getenv(foundVar)
		input = strings.ReplaceAll(input, match[1], foundVal)
	}
	return input
}

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
