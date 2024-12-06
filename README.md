# CoreCTL - Core Platform CTL

CLI for CECG's [Core Platform](https://www.cecg.io/core-platform/).

Core Platform is your ultimate all-in-one developer platform designed to turbocharge your software development journey from Day 1.

Interested in learning more about CECG's Core Platform? Book a demo at [Core Platform](https://www.cecg.io/core-platform/).

# Downloading

Releases for Linux and Mac are published in [releases](https://github.com/coreeng/corectl/releases/)
Download and unzip your platform e.g for Mac with Apple chip download `corectl_Darwin_arm64.tar.gz`
and add to your path.

# Usage 

## Initialisztion

Before usage, you should initialise `corectl`. This will set up your GitHub integration with your developer environments.

But first, you will need:
  - A [GitHub personal token first](#GitHub-Access-Token)
  - To know which GitHub repository is your "environments repository". Please ask your administrator for this.

To initialise `corectl`, run:

```bash
corectl config init
```

It will ask you to enter your GitHub token and the URL of your environments repository.

## Updates

Periodically update local `corectl` configuration by running:
```bash
corectl config update
```
This command will fetch latest changes for configuration repositories.

After the initialization you can start using `corectl`. 

## Commands
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

Requirements for token:
- It should have access to all your organization repositories, since `corectl` might be used to create and configure new repositories.
- Read-Write permissions for Administrations, since `corectl` might be used to create new repositories for applications.
- Read-Write permissions for Contents, since `corectl` will try to clone repositories with configuration and might be used to update contents of the repository.
- Read-Only permissions for Metadata, since `corectl` uses GitHub API with metadata to perform some logic (check if repository exists, for example).
- Read-Write permissions for Workflows, since `corectl` might configure workflow files when creating new applications.
- Read-Write permissions for Environments and Variables, since `corectl` might be used to configure P2P for repositories.
- Read-Write permissions for Pull Requests, since `corectl` might be used to automatically generate Pull Requests with platform configuration updates.
