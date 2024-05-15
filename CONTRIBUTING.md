# Set up

Set env variable: `GOPRIVATE=github.com/coreeng`

Configure git to use https seamlessly. 
For example, with [git-credential-manager](https://github.com/git-ecosystem/git-credential-manager).

# Developing
It's recommended to use docker container for testing `corectl`, so your local filesystem is not affected.
```shell
make dev-env
```