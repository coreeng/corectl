CORECTL_MAIN=./cmd/corectl

build:
	go build $(CORECTL_MAIN)

.PHONY: dev-env
dev-env:
	docker build -f ./devenv.Dockerfile -t corectl-dev-env .
	docker run --rm -it \
		-v ${PWD}:/root/workspace \
		-v ${HOME}/.ssh:/root/.ssh \
		-v ${HOME}/.gitconfig:/root/.gitconfig \
		-v ${GOPATH}/go:/go \
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
