GO111MODULE=on
GOOS=linux
GOARCH=amd64
export GO111MODULE
export GOOS
export GOARCH
default: build

# build just builds a native executable
build:
	go get -u -v ./...
	go build -v

# build_linux builds a linux x86_64 binary
linux:
	go get -u -v ./...
	go build -v
