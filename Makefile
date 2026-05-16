.PHONY: test vet fmt build run windows-package

test:
	go test ./...

vet:
	go vet ./...

fmt:
	go fmt ./...

build:
	go build ./cmd/cims

run:
	go run ./cmd/cims

windows-package:
	powershell -ExecutionPolicy Bypass -File ./scripts/build-windows.ps1
