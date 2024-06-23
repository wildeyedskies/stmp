# Variable declaration
MAIN_PACKAGE_PATH := ./
BINARY_NAME := stmp

# Print help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

# Format code and tidy modfile
.PHONY: tidy
tidy:
	go fmt ./...
	go mod tidy -v

# Build the application
.PHONY: build
build: tidy
	go build -ldflags "-X main.commitHash=`git rev-parse --short HEAD`" -o=./${BINARY_NAME} ${MAIN_PACKAGE_PATH}

# Build the application and run it without arguments
.PHONY: run
run: build
	./${BINARY_NAME}

# Install locally
.PHONY: install
install: build
	mkdir -pv ${HOME}/.local/bin
	cp -v ./${BINARY_NAME} ${HOME}/.local/bin

