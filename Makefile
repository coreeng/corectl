CORECTL_MAIN=./cmd/corectl
MODULE_PATH=$$(go list -m)
GIT_TAG=$$(2> /dev/null git describe --exact-match --tags || echo 'untagged')
GIT_HASH=$$(git rev-parse HEAD)
ARCH=$$(uname -m)
TIMESTAMP=$$(date +"%Y-%m-%dT%H:%M:%S%:z")
LDFLAGS = "\
	-X ${MODULE_PATH}/pkg/version.Version=${GIT_TAG} \
	-X ${MODULE_PATH}/pkg/version.Commit=${GIT_HASH} \
	-X ${MODULE_PATH}/pkg/version.Date=${TIMESTAMP} \
	-X ${MODULE_PATH}/pkg/version.Arch=${ARCH}"

.DEFAULT_GOAL :=help

REQ_BINS = go,docker,golangci-lint
_ := $(foreach exec,$(REQ_BINS), \
       $(if $(shell which $(exec)),some string,$(error "No $(exec) binary in $$PATH")))

# Lint, test and build
.PHONY: all
all: lint test build

# Prints list of tasks
PHONY: help
help:
	@awk 'BEGIN {FS=":"} /^# .*/,/^[a-zA-Z0-9_-]+:/ { if ($$0 ~ /^# /) { desc=substr($$0, 3) } else { printf "\033[36m%-30s\033[0m %s\n", $$1, desc } }' Makefile | grep -v ".PHONY"

# Run lints from golangci.yaml
.PHONY: lint
lint:
	golangci-lint run ./...

# Runs go test
.PHONY: test
test:
	go test ./pkg/... -v

# Build binary
.PHONY: build
build:
	go build \
		-o corectl \
		-ldflags ${LDFLAGS} \
		$(CORECTL_MAIN)

# Run integration tests locally
.PHONY: integration-test-local
integration-test-local: build
	rm -f /tmp/corectl-autoupdate && \
		TEST_CORECTL_BINARY="$$(realpath corectl)" \
		TEST_GITHUB_TOKEN=$${GITHUB_TOKEN} \
		go test ./tests/localintegration -v

# Run integration tests
.PHONY: integration-test
integration-test: build integration-test-local
	rm -f /tmp/corectl-autoupdate && \
		TEST_CORECTL_BINARY="$$(realpath corectl)" \
		TEST_GITHUB_TOKEN=$${GITHUB_TOKEN} \
		go test ./tests/integration -v

# Installs binary 
.PHONY: install
install:
	go install \
		-ldflags ${LDFLAGS} \
		$(CORECTL_MAIN)

# Build docker image for dev
.PHONY: dev-env
dev-env:
	docker build -f ./devenv.Dockerfile -t corectl-dev-env .
	docker run --rm -it \
		-v ${PWD}:/root/workspace \
		-v ${HOME}/.ssh:/root/.ssh \
		-v ${HOME}/.gitconfig:/root/.gitconfig \
		-v $$(go env GOPATH):/go \
		-e TERM=${TERM} \
		-e COLORTERM=${COLORTERM} \
		-p 12345:12345 \
		--name corectl-dev-env \
		corectl-dev-env bash

# Runs bash in container
.PHONY: dev-env-connect
dev-env-connect: dot-env
	docker exec -it corectl-dev-env bash

ARGS=
DEBUG_PORT=12345
# Debug binary with dlv
.PHONY: debug
debug:
	dlv debug --headless --listen=:$(DEBUG_PORT) --api-version=2 --accept-multiclient $(CORECTL_MAIN) -- $(ARGS)

# Cleans built artifacts
.PHONY: clean
clean:
	rm -f $(CORECTL_MAIN)
