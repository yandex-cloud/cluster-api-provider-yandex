.PHONY: prepare fmt vet lint

# Installs all dependencies - do once
prepare:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2

verify: fmt vet lint

fmt:
	go fmt ./...

vet:
	go vet -v ./...

lint:
	${GOPATH}/bin/golangci-lint run