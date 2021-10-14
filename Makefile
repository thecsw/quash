GO111MODULE = on
export GO111MODULE

BINARY_NAME = quash

default: build

# build just builds a native executable
build:
	go get -u -v ./...
	go build -v -o ${BINARY_NAME}

# linux builds a linux x86_64 binary
linux:
	go get -u -v ./...
	GOOS=linux GOARCH=amd64 go build -v -o ${BINARY_NAME}

clean:
	go clean
	rm -vf ${BINARY_NAME}
