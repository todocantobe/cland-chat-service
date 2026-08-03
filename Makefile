.PHONY: build build-debug run tool lint test e2e clean help

all: build

build:
	set CGO_ENABLED=1 && go build -v .

# Debug build
build-debug:
	set CGO_ENABLED=1 && go build -v -gcflags="all=-N -l" -o cland-chat-service.debug.exe .

# 运行服务（需先初始化数据库 docs/sql/init.sql）
run:
	set CGO_ENABLED=1 && go run .

tool:
	go vet ./...; true
	gofmt -w .

lint:
	golint ./...

# 单测（*_test.go，与源码同目录）
test:
	go test ./...

# 端到端测试：WS 用户通信（Python）+ HTTP（e2e/api_test.http，IDE 运行）
e2e:
	@python3 -c "import websocket" 2>/dev/null || pip3 install --quiet websocket-client
	python3 e2e/ws_user_chat.py

clean:
	rm -rf go-gin-example
	go clean -i .

help:
	@echo "make: compile packages and dependencies"
	@echo "make run: 运行服务"
	@echo "make test: 单测"
	@echo "make e2e: 端到端测试（WS 用户通信）"
	@echo "make tool: run specified go tool"
	@echo "make lint: golint ./..."
	@echo "make clean: remove object files and cached files"
