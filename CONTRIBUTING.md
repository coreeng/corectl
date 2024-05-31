# Contributing

Whereas corectl is open source it depends on closed source modules so it is not suitable for outside contribution yet.
Small contributions are welcome but it is best to raise an issue for larger prices of work.

# Set up

Set env variable: `GOPRIVATE=github.com/coreeng`

Configure git to use https seamlessly. 
For example, with [git-credential-manager](https://github.com/git-ecosystem/git-credential-manager).

Or set git config:

```
git config --global --add url."git@github.com:".insteadOf "https://github.com/"
```

# Developing

## Docker

It's recommended to use docker container for testing `corectl`, so your local filesystem is not affected.
```shell
make dev-env
```