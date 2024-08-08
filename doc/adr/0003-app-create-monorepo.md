# p2p promote command

Date: 2024-08-07

## Status

Accepted

## Context

Sometimes we deal with multiple applications in a single repository.
We want corectl to be able to create applications in such setup. 


## Current interface

```shell
corectl app create <app-name> [<local-path>] [flags]

Flags:
      --cplatform string       Path to local repository with core-platform configuration
  -t, --from-template string   Template to use to create an application
      --github-org string      Github organization your company is using
      --github-token string    Personal GitHub token to use for GitHub authentication
  -h, --help                   help for create
      --nonint                 Disable interactive inputs
      --templates string       Path to local repository with software templates
      --tenant string          Tenant to configure for P2P
```

This results in 
- new local git repository created in <local-path>
- application skeleton is rendered in root of the repository
- repository is pushed to remote
- github repository variables are initialized

## Monorepo setup

To achieve multiple applications in a single repository, we need to first initialize a bare repository.
This step is important as we need to initialize a new github repository with p2p related variables.

I can see two options:

### Option 1 - bare flag

We could add a new flag `--bare` to the `app create` command. It would create bare repository without template skeleton being rendered.

```shell
corectl app create <app-name> [<local-path>] --bare
```

### Option 2 - monorepo template

Instead of adding a new flag we can utilize templates to achieve the same result.
I.e. we can create a new template `monorepo` and use it to create a new application.
Monorepo template will only hold `Readme.md` file but no p2p related files.

This is my favourite option as it allows us to use potentially different `monorepo` templates.

```shell
corectl app create app-name --from-template monorepo
```

### Adding applications

Once we have a bare repository, we can add new applications to it.

```shell
corectl app create <app-name> <monorepo-path>/<app-name> --tenant sub-tenant-name
```

Changes to current implementation

1. if local-path is already a git repository
   - don't create a new github repository
   - don't set github variables (as they already set by parent)
   - check if there is no local changes
2. `tenant` should be passed to template to be rendered in Makefile
3. `.github` folder from templates would have to be copied to root of the repository
and all files would be prefixed with `app-name`

