BINPATH ?= out/agentapi

# WebUI removed - no chat build needed
.PHONY: build
build:
	@echo "Building agentapi without WebUI..."
	go build -o ${BINPATH} main.go

.PHONY: clean
clean:
	rm -rf out/

.PHONY: build-all
build-all:
	@echo "Building for all platforms..."
	GOOS=darwin GOARCH=amd64 go build -o out/agentapi-darwin-amd64 main.go
	GOOS=darwin GOARCH=arm64 go build -o out/agentapi-darwin-arm64 main.go
	GOOS=linux GOARCH=amd64 go build -o out/agentapi-linux-amd64 main.go
	GOOS=linux GOARCH=arm64 go build -o out/agentapi-linux-arm64 main.go
	GOOS=windows GOARCH=amd64 go build -o out/agentapi-windows-amd64.exe main.go