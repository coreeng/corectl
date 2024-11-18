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

.PHONY: default
default: help

## make help				Prints help command
.PHONY: help
help: Makefile
	@echo "Usage: "
	@sed -n 's/^##[ -]/   /p' Makefile
	@echo ""

## make lint				Runs lints
.PHONY: lint
lint:
	golangci-lint run ./...

## make test				Runs tests
.PHONY: test
test:	
	go test ./pkg/... -v

## make build				Builds corectl
.PHONY: build
build:
	go build \
		-o corectl \
		-ldflags ${LDFLAGS} \
		$(CORECTL_MAIN)

## make integration-test-local		Local integration tests
.PHONY: integration-test-local
integration-test-local: build
	rm -f /tmp/corectl-autoupdate && \
		TEST_CORECTL_BINARY="$$(realpath corectl)" \
		TEST_GITHUB_TOKEN=$${GITHUB_TOKEN} \
		go test ./tests/localintegration -v

## make integration-test		Integration tests
.PHONY: integration-test
integration-test: build integration-test-local
	rm -f /tmp/corectl-autoupdate && \
		TEST_CORECTL_BINARY="$$(realpath corectl)" \
		TEST_GITHUB_TOKEN=$${GITHUB_TOKEN} \
		go test ./tests/integration -v

## make install				Installs corectl
.PHONY: install
install:
	go install \
		-ldflags ${LDFLAGS} \
		$(CORECTL_MAIN)

## make dev-env				Build docker image for dev
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

## make dev-env-connect			Run corectl container
.PHONY: dev-env-connect
dev-env-connect:
	docker exec -it corectl-dev-env bash

ARGS=
DEBUG_PORT=12345
## make debug				Debug binary with dlv
.PHONY: debug
debug:
	dlv debug --headless --listen=:$(DEBUG_PORT) --api-version=2 --accept-multiclient $(CORECTL_MAIN) -- $(ARGS)

## make clean				Cleans built artifacts
.PHONY: clean
clean:
	rm -f $(CORECTL_MAIN)
