# 1. versioning

Date: 2024-04-08

## Status

Accepted

## Context

`corectl` is a client for Core Platform which simplifies the work.

Since `corectl` and Core Platform are versioned separately, 
we need to define a rules for versioning `corectl` to understand,
which versions of `corectl` and Core Platform are compatible.

### Core Platform versioning state
- Follows [Semantic Versioning](https://semver.org/) (kind of)
- At the moment under v0, which means that [special rules are applied](https://semver.org/#doesnt-this-discourage-rapid-development-and-fast-iteration)
  - Minor version is increased in case of new feature or backward incompatible change
  - We still try to avoid breaking changes

## Decision

While we are at v0:
- Make sure `corectl` and Core Platform with the same minor versions are compatible.
- Core Platform: increment minor version only on breaking changes.
  This way a minor version specifies an interface "base" version between `corectl` and Core Platform,
  meaning that they are compatible.
  And if new functionality or bugfixes is needed on either side, a patch version should be increased.

`corectl` depends on Core Platform and P2P logic.
Matching `corectl` and P2P versions in addition would introduce additional complication,
since Core Platform and P2P versions are not matched at the moment and P2P already has v1.
Most of the P2P configuration duplicates env configuration from the environment specs.
Once we implement a mechanism for this info to be automatically discovered by environment names,
the P2P configuration related to GitHub repo (envs + vars) can be removed completely.
This is true because only "P2P stage -> environment" matching is left,
which can be moved to action input variables (directly to workflow.yaml files).

## Consequences

- It's clear which version of `corectl` to use with the concrete Core Platform version.
- It's kind of counter-intuitive to have significant features introduced as patches, but it's just for v0.
- There is no dependency on P2P version, but we need to rework P2P interface.
