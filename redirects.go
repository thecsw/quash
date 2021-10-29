package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

var (
	stdinFileRegex  = regexp.MustCompile(`<\s*([^<> ]+)`)
	stdoutFileRegex = regexp.MustCompile(`1?>\s*([^<> ]+)`)
	stderrFileRegex = regexp.MustCompile(`2>\s*([^<> ]+)`)
)

// setReridects takes a command and sets all redirects if needed
func setReridects(
	command string,
	stdinDestination **os.File,
	stdoutDestination **os.File,
	stderrDestination **os.File,
) (string, error) {
	var err error
	cleanCommand := command
	if stdinFileRegex.MatchString(cleanCommand) {
		cleanCommand, err = setStdinRedirect(
			cleanCommand, stdinDestination)
		if err != nil {
			return "", err
		}
	}
	// stderr regex is stronger and should evaluate before stdout
	if stderrFileRegex.MatchString(cleanCommand) {
		cleanCommand, err = setStderrRedirect(
			cleanCommand, stderrDestination)
		if err != nil {
			return "", err
		}
	}
	if stdoutFileRegex.MatchString(cleanCommand) {
		cleanCommand, err = setStdoutRedirect(
			cleanCommand, stdoutDestination)
		if err != nil {
			return "", err
		}
	}
	return cleanCommand, nil
}

// setStdinRedirect sets on stdin redirect if < is found
func setStdinRedirect(
	command string,
	destination **os.File,
) (string, error) {
	matches := stdinFileRegex.FindAllStringSubmatch(command, -1)
	filename := matches[0][1]
	infile, err := os.Open(filepath.Join(currDir, filename))
	//defer infile.Close()
	if err != nil {
		return "", errors.Wrap(err, "couldn't open in file")
	}
	*destination = infile
	command = stdinFileRegex.ReplaceAllString(command, "")
	return strings.TrimSpace(command), nil
}

// setStdoutRedirect sets an stdout file redirect if > or 1> is found
func setStdoutRedirect(
	command string,
	destination **os.File,
) (string, error) {
	matches := stdoutFileRegex.FindAllStringSubmatch(command, -1)
	filename := matches[0][1]
	outfile, err := os.Create(filepath.Join(currDir, filename))
	//defer outfile.Close()
	if err != nil {
		return "", errors.Wrap(err, "couldn't create out file")
	}
	*destination = outfile
	command = stdoutFileRegex.ReplaceAllString(command, "")
	return strings.TrimSpace(command), nil
}

// setStderrRedirect sets an error file redirect if 2> is found
func setStderrRedirect(
	command string,
	destination **os.File,
) (string, error) {
	matches := stderrFileRegex.FindAllStringSubmatch(command, -1)
	filename := matches[0][1]
	errfile, err := os.Create(filepath.Join(currDir, filename))
	//defer errfile.Close()
	if err != nil {
		return "", errors.Wrap(err, "couldn't create err file")
	}
	*destination = errfile
	command = stderrFileRegex.ReplaceAllString(command, "")
	return strings.TrimSpace(command), nil
}
