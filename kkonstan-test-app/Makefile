projectDir := $(realpath $(dir $(firstword $(MAKEFILE_LIST))))
os := $(shell uname)
app_name = kkonstan-test-app
tenant_name = kkonstan
image_name = $(app_name)
image_tag = $(VERSION)
helm_test_timeout_duration = 2m

FAST_FEEDBACK_PATH = fast-feedback
EXTENDED_TEST_PATH = extended-test
PROD_PATH = prod

MONITORING=false
DASHBOARDING=false
RUN_EXTENDED_TEST=false

prometheusUrl=http://prometheus-$(tenant_name).$(tenant_name):9090
extended_test_replicas = 0 # until we have autoscaling working
extended_test_wait_timeout = 15m

.PHONY: help-p2p
help-p2p:
	@grep -E '^[a-zA-Z1-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep p2p | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: help-all
help-all:
	@grep -E '^[a-zA-Z1-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

# P2P tasks

.PHONY: p2p-image
p2p-image:
	@echo $(image_name)

.PHONY: p2p-build
p2p-build: lint service-build service-push ## Builds the service image and pushes it to the registry

.PHONY: p2p-functional
p2p-functional: namespace_environment=functional
p2p-functional: build-functional push-functional create-ns-functional deploy-functional ## Runs only functional helm tests
	$(MAKE) helm-test-$(namespace_environment) scale_down=true

.PHONY: p2p-nft
p2p-nft: namespace_environment=nft
p2p-nft: build-nft push-nft create-ns-nft deploy-nft ## Runs only NFT helm tests
	$(MAKE) helm-test-$(namespace_environment) scale_down=true

.PHONY: p2p-integration
p2p-integration: namespace_environment=integration
p2p-integration: build-integration push-integration create-ns-integration deploy-integration ## Runs only Integration helm tests
	$(MAKE) helm-test-$(namespace_environment) scale_down=false

.PHONY: p2p-extended-test
p2p-extended-test: build-extended push-extended create-ns-extended deploy-extended  ## Runs extended tests
	@if [ "$(RUN_EXTENDED_TEST)" != "true" ] ; then \
		echo "Skipping extended test" ; \
	else \
		$(MAKE) run-extended ; \
	fi

.PHONY: p2p-prod
p2p-prod: deploy-prod ## Deploy helm chart of the app to production namespace

## Namespace creation

.PHONY: create-ns-functional
create-ns-functional: env=functional
create-ns-functional: create-namespace

.PHONY: create-ns-nft
create-ns-nft: env=nft
create-ns-nft: create-namespace

.PHONY: create-ns-integration
create-ns-integration: env=integration
create-ns-integration: create-namespace

.PHONY: create-ns-extended
create-ns-extended: env=extended
create-ns-extended: create-namespace


## Internal Tasks, not part of the P2P

.PHONY: create-namespace
create-namespace:
	awk -v NAME="$(tenant_name)" -v ENV="$(env)" '{ \
		sub(/{tenant_name}/, NAME);  \
		sub(/{env}/, ENV);  \
		print;  \
	}' resources/subns-anchor.yaml | kubectl apply -f -

.PHONY: lint
lint: ## Linting go code
	docker run -v "$$(pwd)":/var/app ghcr.io/mgechev/revive:v1.3.2  -config /var/app/revive.toml -formatter stylish  ./var/app/cmd/service ./var/app/cmd/handler

.PHONY: service-build
service-build:
	docker build --file Dockerfile --tag $(REGISTRY)/$(FAST_FEEDBACK_PATH)/$(image_name):$(image_tag) .

.PHONY: service-push
service-push: ## Push the service image
	docker image push $(REGISTRY)/$(FAST_FEEDBACK_PATH)/$(image_name):$(image_tag)

.PHONY: build-functional
build-functional: ## Build the functional application, and package as a Docker image
	docker build --file functional/Dockerfile --tag $(REGISTRY)/$(FAST_FEEDBACK_PATH)/$(image_name)-functional:$(image_tag) .

.PHONY: build-nft
build-nft: ## Build the non-functional-tests application, and package as a Docker image
	docker build --file nft/Dockerfile --tag $(REGISTRY)/$(FAST_FEEDBACK_PATH)/$(image_name)-nft:$(image_tag) .

.PHONY: build-integration
build-integration: ## Build the integration tests application, and package as a Docker image
	docker build --file integration/Dockerfile --tag $(REGISTRY)/$(FAST_FEEDBACK_PATH)/$(image_name)-integration:$(image_tag) .

.PHONY: build-extended
build-extended:
	docker build --file extended/Dockerfile --tag $(REGISTRY)/$(EXTENDED_TEST_PATH)/$(image_name)-extended:$(image_tag) .

.PHONY: push-functional
push-functional: ## Push the Functional Docker image to an image registry
	docker image push $(REGISTRY)/$(FAST_FEEDBACK_PATH)/$(image_name)-functional:$(image_tag)

.PHONY: push-nft
push-nft: ## Push the NFT Docker image to an image registry
	docker image push $(REGISTRY)/$(FAST_FEEDBACK_PATH)/$(image_name)-nft:$(image_tag)

.PHONY: push-integration
push-integration: ## Push the Integration test Docker image to an image registry
	docker image push $(REGISTRY)/$(FAST_FEEDBACK_PATH)/$(image_name)-integration:$(image_tag)

.PHONY: push-extended
push-extended:
	docker image push $(REGISTRY)/$(EXTENDED_TEST_PATH)/$(image_name)-extended:$(image_tag)

.PHONY: deploy-functional
deploy-functional: ## Deploy helm chart of the app to functional namespace
	helm upgrade --install $(app_name) helm-charts/app -n $(tenant_name)-functional \
		--set registry=$(REGISTRY)/$(FAST_FEEDBACK_PATH) \
		--set domain=$(BASE_DOMAIN) \
		--set appUrlSuffix="-$(tenant_name)-functional" \
		--set tag=$(image_tag) \
		--set service.image=$(image_name) \
		--set functional.image=$(image_name)-functional \
		--set tenantName=$(tenant_name) \
		--set appName=$(app_name) \
		--set service.resources.requests.cpu=0m \
		--atomic
	helm list -n $(tenant_name)-functional ## list installed charts in the given tenant namespace

.PHONY: deploy-nft
deploy-nft:  ## Deploy helm chart of the app to nft namespace
	helm upgrade --install $(app_name) helm-charts/app -n $(tenant_name)-nft \
		--set registry=$(REGISTRY)/$(FAST_FEEDBACK_PATH) \
		--set domain=$(BASE_DOMAIN) \
		--set appUrlSuffix="-$(tenant_name)-nft" \
		--set tag=$(image_tag) \
		--set service.image=$(image_name) \
		--set nft.image=$(image_name)-nft \
		--set monitoring=$(MONITORING) \
		--set dashboarding=$(DASHBOARDING) \
		--set tenantName=$(tenant_name) \
		--set appName=$(app_name) \
		--atomic
	helm list -n $(tenant_name)-nft ## list installed charts in the given tenant namespace

.PHONY: deploy-integration
deploy-integration:  ## Deploy helm chart of the app to integration namespace
	helm upgrade --install $(app_name) helm-charts/app -n $(tenant_name)-integration \
		--set registry=$(REGISTRY)/$(FAST_FEEDBACK_PATH) \
		--set domain=$(BASE_DOMAIN) \
		--set appUrlSuffix="-$(tenant_name)-integration" \
		--set tag=$(image_tag) \
		--set service.image=$(image_name) \
		--set integration.image=$(image_name)-integration \
		--set monitoring=$(MONITORING) \
		--set dashboarding=$(DASHBOARDING) \
		--set tenantName=$(tenant_name) \
		--set appName=$(app_name) \
		--set service.resources.requests.cpu=0m \
		--atomic
	helm list -n $(tenant_name)-integration ## list installed charts in the given tenant namespace

.PHONY: deploy-extended
deploy-extended:
	helm upgrade --install $(app_name) helm-charts/app -n $(tenant_name)-extended \
		--set registry=$(REGISTRY)/$(EXTENDED_TEST_PATH) \
		--set domain=$(BASE_DOMAIN) \
		--set appUrlSuffix="-$(tenant_name)-extended" \
		--set tag=$(image_tag) \
		--set prometheusEndpoint=$(prometheusEndpoint) \
		--set service.replicas=$(extended_test_replicas) \
		--set service.image=$(image_name) \
		--set extended.image=$(image_name)-extended \
		--set monitoring=$(MONITORING) \
		--set dashboarding=$(DASHBOARDING) \
		--set tenantName=$(tenant_name) \
		--set appName=$(app_name) \
		--atomic
	helm list -n $(tenant_name)-extended

.PHONY: run-extended
run-extended: namespace = $(tenant_name)-extended
run-extended: chmod-scripts
	./scripts/extended-test.sh extended $(tenant_name) $(app_name) true $(extended_test_wait_timeout)

.PHONY: deploy-prod
deploy-prod: ## Deploy helm chart of the app to production namespace
	@echo $(image_tag)
	@echo $(REGISTRY)/release
	helm upgrade --install $(app_name) helm-charts/app -n $(tenant_name) \
		--set registry=$(REGISTRY)/$(PROD_PATH) \
		--set domain=$(BASE_DOMAIN) \
		--set tag=$(image_tag) \
		--set tenantName=$(tenant_name) \
		--set appName=$(app_name) \
		--set service.image=$(image_name) \
		--atomic
	helm list -n $(tenant_name) ## list installed charts in the given tenant namespace

.PHONY: p2p-promote-generic
p2p-promote-generic:  ## Generic promote functionality
	corectl p2p promote $(image_name):${image_tag} \
        		--source-stage $(source_repo_path) \
        		--dest-registry $(REGISTRY) \
        		--dest-stage $(dest_repo_path)

.PHONY: p2p-promote-to-extended-test
p2p-promote-to-extended-test: source_repo_path=$(FAST_FEEDBACK_PATH)
p2p-promote-to-extended-test: dest_repo_path=$(EXTENDED_TEST_PATH)
p2p-promote-to-extended-test:  p2p-promote-generic

.PHONY: p2p-promote-to-prod
p2p-promote-to-prod:  source_repo_path=$(EXTENDED_TEST_PATH)
p2p-promote-to-prod:  dest_repo_path=$(PROD_PATH)
p2p-promote-to-prod:  p2p-promote-generic

.PHONY: chmod-scripts
chmod-scripts:
	chmod -R +x ./scripts

.PHONY: helm-test-%
helm-test-%: chmod-scripts
	echo "### helm $* tests run ###"
	./scripts/helm-test.sh $* $(tenant_name) $(app_name) $(scale_down) $(helm_test_timeout_duration)
