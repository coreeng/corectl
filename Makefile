CORECTL_MAIN=./cmd/corectl
MODULE_PATH=$$(go list -m)
GIT_TAG=$$(2> /dev/null git describe --exact-match --tags || echo 'untagged')
GIT_HASH=$$(git rev-parse HEAD)
ARCH=$$(uname -m)
TIMESTAMP=$$(date +"%Y-%m-%dT%H:%M:%S%:z")
LDFLAGS = "\
	-X ${MODULE_PATH}/pkg/cmd/version.version=${GIT_TAG} \
	-X ${MODULE_PATH}/pkg/cmd/version.commit=${GIT_HASH} \
	-X ${MODULE_PATH}/pkg/cmd/version.date=${TIMESTAMP} \
	-X ${MODULE_PATH}/pkg/cmd/version.arch=${ARCH}"


.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: test
test:	
	go test ./pkg/... -v

.PHONY: build
build:
	go build \
		-o corectl \
		-ldflags ${LDFLAGS} \
		$(CORECTL_MAIN)

.PHONY: integration-test
integration-test: build
	TEST_CORECTL_BINARY="$$(realpath corectl)" \
		TEST_GITHUB_TOKEN=$${GITHUB_TOKEN} \
		go test ./tests/integration -v

.PHONY: install
install:
	go install \
		-ldflags ${LDFLAGS} \
		$(CORECTL_MAIN)

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

.PHONY: dev-env-connect
dev-env-connect:
	docker exec -it corectl-dev-env bash

ARGS=
DEBUG_PORT=12345
debug:
	dlv debug --headless --listen=:$(DEBUG_PORT) --api-version=2 --accept-multiclient $(CORECTL_MAIN) -- $(ARGS)
