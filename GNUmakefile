GORELEASER_VERSION ?= v2.15.4

default: fmt lint install generate

build:
	go build -v ./...

install: build
	go install -v ./...

lint:
	golangci-lint run

generate:
	cd tools; go generate ./...

fmt:
	gofmt -s -w -e .

test:
	go test -v -cover -timeout=120s -parallel=10 ./...

testacc:
	TF_ACC=1 go test -v -cover -timeout 120m ./...

release-check:
	go run github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION) check

release-snapshot:
	go run github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION) release --snapshot --clean --skip=sign

.PHONY: fmt lint test testacc build install generate release-check release-snapshot
