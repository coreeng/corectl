
.PHONY: dev-env
dev-env:
	docker build -f ./devenv.Dockerfile -t dpctl-dev-env .
	docker run --rm -it \
		-v ${PWD}:/root/workspace \
		-v ${HOME}/.ssh:/root/.ssh \
		-v ${HOME}/.gitconfig:/root/.gitconfig \
		-v ${GOPATH}/go:/go \
		-p 12345:12345 \
		--name dpctl-dev-env \
		dpctl-dev-env bash --init-file ./dev-env/init.sh

.PHONY: dev-env-connect
dev-env-connect:
	docker exec -it dpctl-dev-env bash

ARGS=
DEBUG_PORT=12345
debug:
	dlv debug --headless --listen=:$(DEBUG_PORT) --api-version=2 --accept-multiclient -- $(ARGS)
