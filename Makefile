.PHONY: build clean tool lint help

all: build

build:
	set CGO_ENABLED=1 && go build -v .

# Debug build
build-debug:
	set CGO_ENABLED=1 && go build -v -gcflags="all=-N -l" -o cland-chat-service.debug.exe .

tool:
	go vet ./...; true
	gofmt -w .

lint:
	golint ./...

clean:
	rm -rf go-gin-example
	go clean -i .

help:
	@echo "make: compile packages and dependencies"
	@echo "make tool: run specified go tool"
	@echo "make lint: golint ./..."
	@echo "make clean: remove object files and cached files"
