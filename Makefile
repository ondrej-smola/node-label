SHELL := /bin/bash

.PHONY: test build

help:
	@echo Usage:
	@echo
	@echo "  make lint"
	@echo "  make fmt"
	@echo "  make test"
	@echo "  make VERSION=0.0.0-0 image-build"
	@echo "  make VERSION=0.0.0-0 image-push"
	@echo

require-ko:
	@test -n "$$(which ko)" || \
		(if [ "$$DEP_AUTO_INSTALL" != "1" ]; then \
			echo '"ko" (v0.17.1) must be installed (see https://github.com/ko-build/ko). Re-run make with DEP_AUTO_INSTALL=1 to auto-install.' >&2; exit 1; else \
			go install github.com/google/ko@v0.17.1; fi)


require-golangci-init:
	@test -n "$$(which golangci-lint)" || \
		(if [ "$$DEP_AUTO_INSTALL" != "1" ]; then \
			echo '"golang-ci-lint" (v1.63.4) must be installed (see https://github.com/golangci/golangci-lint). Re-run make with DEP_AUTO_INSTALL=1 to auto-install.' >&2; exit 1; else \
			curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/v1.63.4/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.63.4; fi)

lint: require-golangci-init
	golangci-lint run --timeout 10m

lint-fix: require-golangci-init
	golangci-lint run --fix

test:
	go test ./...

test-race:
	go test -race ./...

test-coverage:
	go test -coverprofile cover.out ./...
	go tool cover -html=cover.out -o coverage.html
	rm -f cover.out

KO_DOCKER_REPO ?= docker.io/ondrejsmola/auto-zone-label

image-build: require-ko
	KO_DOCKER_REPO=$(KO_DOCKER_REPO) ko build ./cmd/controller --local -t $(VERSION) --bare

image-push: require-ko
	KO_DOCKER_REPO=$(KO_DOCKER_REPO) ko build ./cmd/controller -t $(VERSION) --bare --sbom none


run:
	@echo "Run the controller with:"
	@echo "  ko run ./cmd/controller --local -p 8080:8080"
	@echo "or build and run the image with:"
	@echo "  make image-build && docker run -p 8080:8080 $(KO_DOCKER_REPO):$(VERSION)"
	@echo
	@echo "To test the controller, you can use:"
	@echo "  kubectl apply -f config/samples/auto_node_label_v1alpha1_autonodelabel.yaml"
	@echo "  kubectl get autonodelabels.autonodelabel.ondrejsmola.com"
