package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
)

type QuashTestSuite struct {
	suite.Suite
}

// SetupSuite runs on the testing suite initialization (only once)
func (suite *QuashTestSuite) SetupSuite() {
}

// TestSetStdinRedirect tests SetStdinRedirect function
func (suite *QuashTestSuite) TestSetStdinRedirect() {
	defaultDestination := os.Stdin
	command := "wc < test/example.in"
	newCommand, err := setStdinRedirect(command, &defaultDestination)
	suite.Nil(err)
	suite.Equal("wc", newCommand)
	suite.NotEqual(os.Stdin, defaultDestination)
	b := make([]byte, 34)
	readSize, err := defaultDestination.Read(b)
	suite.Nil(err)
	suite.Equal(len(b), readSize)
	suite.Equal(`this is example input
second line
`, string(b))
	suite.Nil(defaultDestination.Close())
}

// TestSetStdoutRedirect tests SetStdoutRedirect
func (suite *QuashTestSuite) TestSetStdoutRedirect() {
	defaultDestination := os.Stdout
	command := "echo -e 'hello world\nthis is sandy' > test/hello.out"
	newCommand, err := setStdoutRedirect(command, &defaultDestination)
	suite.Nil(err)
	suite.Equal("echo -e 'hello world\nthis is sandy'", newCommand)
	suite.NotEqual(os.Stdout, defaultDestination)
	suite.Nil(defaultDestination.Close())
	// Verify the file's existence (no contents should be there)
	file, err := os.Stat("test/hello.out")
	suite.Nil(err)
	suite.Equal(int64(0), file.Size())
	suite.Nil(os.Remove("test/hello.out"))
}

// TestSetStderrRedirect tests SetStderrRedirect
func (suite *QuashTestSuite) TestSetStderrRedirect() {
	defaultDestination := os.Stderr
	command := "echo -e 'hello world\nthis is sandy' 2> test/hello.err"
	newCommand, err := setStderrRedirect(command, &defaultDestination)
	suite.Nil(err)
	suite.Equal("echo -e 'hello world\nthis is sandy'", newCommand)
	suite.NotEqual(os.Stderr, defaultDestination)
	suite.Nil(defaultDestination.Close())
	// Verify the file's existence (no contents should be there)
	file, err := os.Stat("test/hello.err")
	suite.Nil(err)
	suite.Equal(int64(0), file.Size())
	suite.Nil(os.Remove("test/hello.err"))
}

// TestSetReridects tests setRedirects
func (suite *QuashTestSuite) TestSetReridects() {
	defaultStdin := os.Stdin
	defaultStdout := os.Stdout
	defaultStderr := os.Stderr
	command := "wc < test/example.in > test/jacob.out 2> test/jacob.err"
	newCommand, err := setReridects(command, &defaultStdin, &defaultStdout, &defaultStderr)
	suite.Nil(err)
	suite.Equal("wc", newCommand)
	suite.NotEqual(os.Stdin, defaultStdin)
	suite.NotEqual(os.Stdout, defaultStdout)
	suite.NotEqual(os.Stderr, defaultStderr)
	// Verify all file's existence and correctness of in-file
	// in file
	b := make([]byte, 34)
	readSize, err := defaultStdin.Read(b)
	suite.Nil(err)
	suite.Equal(len(b), readSize)
	suite.Equal(`this is example input
second line
`, string(b))
	suite.Nil(defaultStdin.Close())
	// out file
	outFile, err := os.Stat("test/jacob.out")
	suite.Nil(err)
	suite.Equal(int64(0), outFile.Size())
	suite.Nil(os.Remove("test/jacob.out"))
	// err file
	errFile, err := os.Stat("test/jacob.err")
	suite.Nil(err)
	suite.Equal(int64(0), errFile.Size())
	suite.Nil(os.Remove("test/jacob.err"))
}

// TearDownSuite runs after all tests are finished (no matter
//if it failed or succeeded)
func (suite *QuashTestSuite) TearDownSuite() {
}

// TestArkadicTestSuite fires the testing suite
func TestArkadicTestSuite(t *testing.T) {
	suite.Run(t, new(QuashTestSuite))
}
