# CoreCTL - Core Platform CTL

`corectl` is the CLI for CECG's [Core Platform](https://www.cecg.io/core-platform/).

Core Platform is your ultimate all-in-one developer platform designed to turbocharge your software development journey from Day 1.

Interested in learning more about CECG's Core Platform? Book a demo at [Core Platform](https://www.cecg.io/core-platform/).

# Downloading

Releases for Linux and Mac are published in [releases](https://github.com/coreeng/corectl/releases/)
Download and unzip your platform e.g for Mac with Apple chip download `corectl_Darwin_arm64.tar.gz`
and add to your path.

# Usage 

Before start using the CLI you should initialize it.
For the initialization you have to provide:
- initialization file: [corectl-init-example.yaml](corectl-init-example.yaml)
- your person GitHub token to perform operations on your behalf. See more info [here](#GitHub-Access-Token)

To run initialization run:
```bash
corectl config init
```
It will save a few configuration options and clone configuration repositories:
`cplatform-environments` and `software-templates`.

Please note that you should periodically update local `corectl` configuration by running:
```bash
corectl config update
```
It will pull new changes for configuration repositories.
I

After the initialization you can start using `corectl`. 

To check for available operations run:
```bash 
corectl --help
```

# GitHub Access Token

## Classic Personal Access Token
Scopes required:
- `repo`, since `corectl` needs access to read, create repositories, create PullRequests, configure environments and variables for the repositories.
- `workflow`, since `corectl` may create workflow files when creating new applications.

## Fine-grained tokens
> **_NOTE_**: Your organization has to enable use of fine-grained tokens for this to be possible.

Requirements for the token:
- It should have access to all your organization repositories, since `corectl` might be used to create and configure new repositories.
- Read-Write permissions for Administrations, since `corectl` might be used to create new repositories for applications.
- Read-Write permissions for Contents, since `corectl` will try to clone repositories with configuration and might be used to update contents of the repository.
- Read-Only permissions for Metadata, since `corectl` uses GitHub API with metadata to perform some logic (check if repository exists, for example).
- Read-Write permissions for Workflows, since `corectl` might configure workflow files when creating new applications.
- Read-Write permissions for Environments and Variables, since `corectl` might be used to configure P2P for repositories.
- Read-Write permissions for Pull Requests, since `corectl` might be used to automatically generate Pull Requests with platform configuration updates.
