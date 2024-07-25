# p2p promote command

Date: 2024-07-22

## Status

Accepted

## Context

We want a way to promote docker images between environments.
Currently, we implement this via Makefile tasks which means that we have duplicate code to every application 
repository. To reduce the amount of that duplication, we want to move it to a `corectl` command

## Command interface

### Option 1 - pass parameters currently used by Makefile

For context this is a sample Makefile task for promotion

```makefile
.PHONY: p2p-promote-generic
p2p-promote-generic:  ## Generic promote functionality
	@echo "$(red) Retagging version ${image_tag} from $(SOURCE_REGISTRY) to $(REGISTRY)"
	export CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE=$(SOURCE_AUTH_OVERRIDE) ; \
	gcloud auth configure-docker --quiet europe-west2-docker.pkg.dev; \
	docker pull $(SOURCE_REGISTRY)/$(source_repo_path)/$(image_name):${image_tag} ; \
	docker tag $(SOURCE_REGISTRY)/$(source_repo_path)/$(image_name):${image_tag} $(REGISTRY)/$(dest_repo_path)/$(image_name):${image_tag}
	@echo "$(red) Pushing version ${image_tag}"
	export CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE=$(DEST_AUTH_OVERRIDE) ; \
	docker push $(REGISTRY)/$(dest_repo_path)/$(image_name):${image_tag}

promote-extended: source_repo_path=$(FAST_FEEDBACK_PATH)
promote-extended: dest_repo_path=$(EXTENDED_TEST_PATH)
promote-extended:  p2p-promote-generic
```

We can transfer directly all parameters as they are to the `corectl` command

```shell
corectl p2p promote $(image_name):${image_tag} \
		--source-registry $(SOURCE_REGISTRY) \
		--source-repo-path $(source_repo_path) \
		--source-auth-override $(SOURCE_AUTH_OVERRIDE) \
		--dest-registry $(REGISTRY) \
		--dest-repo-path $(dest_repo_path) \
		--dest-auth-override $(DEST_AUTH_OVERRIDE)
```

Additionally, you can notice that there are 2 types of environment variables used here:
- upper case variables - those are set by p2p github workflow and are not expected to change between different Makefiles implementations
- lower case variables - those are set in the Makefile

By allowing `p2p promote` to also lookup environment variables, we can avoid the need to pass them as parameters.
Resulting in following invocation

```shell
corectl p2p promote $(image_name):${image_tag} \
    --source-repo-path $(source_repo_path) \
    --dest-registry $(REGISTRY) \
    --dest-repo-path $(dest_repo_path)
```

Full interface is as follows

```shell
corectl p2p promote -h                  
Promotes image

Usage:
  corectl p2p promote <image_with_tag> [flags]

Flags:
      --dest-auth-override string     optional, defaults to environment variable: DEST_AUTH_OVERRIDE
      --dest-registry string          required, defaults to environment variable: DEST_REGISTRY
      --dest-repo-path string         required, defaults to environment variable: DEST_REPO_PATH
  -h, --help                          help for promote
      --source-auth-override string   optional, defaults to environment variable: SOURCE_AUTH_OVERRIDE
      --source-registry string        required, defaults to environment variable: SOURCE_REGISTRY
      --source-repo-path string       required, defaults to environment variable: SOURCE_REPO_PATH
```

#### Pros/cons

Pros:
- easy to migrate current Makefiles
- no need to initialize corectl in p2p github workflow
- works well with current state of p2p

Cons:
- not very opinionated, lot of room for errors, i.e. not following p2p defined promotion path
- when executing locally requires knowledge about the cloud environment, i.e. `gcp projectId`

### Option 2 - opinionated, tightly coupled with p2p workflow

Interface:

```shell
corectl init
corectl p2p promote imageName:tag \
  --tenant=tenant \
  --from-stage=fast-feedback \
  --source-auth-override $(SOURCE_AUTH_OVERRIDE) \ 
  --dest-auth-override $(DEST_AUTH_OVERRIDE) 
```

This option requires initialization of `corectl` in p2p github workflow in order to get environments repository.
Once initialized, we can infer docker registry information (environment specific) for each of the p2p stage.
This is a big change as it means p2p worfklow is not longer responsible for handling details of cloud environments.
It presents a few challenges with current p2p implementation

1. P2P allows to promote images to multiple environments. They way it does it is by using a matrix strategy.
   So there could be multiple promotion p2p jobs running. Moving this responsibility to `corectl` would mean that
   it would have to handle multi environment promotion
2. P2P has steps to authenticate to environments and then saves credential files to local disk. If we ware to handle 
   multi environment promotion, we would need to move authentication to `corectl` as well.


#### Pros/cons

Pros:
- opinionated, less error prone
- uses environments repository as source of truth

Cons:
- requires initialization of `corectl` in p2p github workflow
- requires implementation of multi environment promotion

## Decision

I think that Option 1 would be good enough for now. Option 2 requires more high level design on how corectl can fit 
into p2p workflow.
