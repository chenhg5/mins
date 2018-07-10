# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=mins
BINARY_UNIX=$(BINARY_NAME)_mac
BINARY_LINUX=$(BINARY_NAME)_linux
BINARY_NAME_WINDOW=$(BINARY_NAME).exe

all: run

build:
	$(GOBUILD) -o ./build/$(BINARY_UNIX) -v ./

test:
	$(GOTEST) -v ./

clean:
	$(GOCLEAN)
	rm -f ./build/$(BINARY_NAME)
	rm -f ./build/$(BINARY_UNIX)

run:
	$(GOBUILD) -o ./build/$(BINARY_UNIX) -v ./
	./build/$(BINARY_UNIX)

restart:
	kill -INT $$(cat pid)
	$(GOBUILD) -o ./build/$(BINARY_UNIX) -v ./
	./build/$(BINARY_NAME)

deps:
	$(GOGET) github.com/kardianos/govendor
	govendor sync

cross:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o ./build/$(BINARY_NAME) -v ./

crosswindow:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) -o ./build/$(BINARY_NAME_WINDOW) -v ./