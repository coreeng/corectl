# 1. Corectl promote-image command

Date: 2024-07-22

## Status

Accepted

## Context

We want a way to promote docker images between environments.
Currently, we implement this via Makefile tasks which means that we have duplicate code to every application 
repository. To reduce the amount of duplicate code, we want to move it to a `corectl` command

## Command interface

TODO: authentication arguments

### Option 1

This option requires operator to know details about the cloud environment, i.e `gcp projectId`

```shell
corectl p2p promote-image 
    --imageUri=${gcp_region}-docker.pkg.dev/${gcp_project_id}/tenant/${tenant}/${stage}/lukasz-app-1:0.0 \ 
    --to-registry=${gcp_region}-docker.pkg.dev/${gcp_project_id}/tenant/${tenant}/${stage}/lukasz-app-1
```

### Option 2

We can rely on `corectl` to get the details about the cloud environment. 

```shell
corectl p2p promote-image 
    --image=lukasz-app-1:0.0 \
    --tenant=lukasz \
    --from-env=gcp-dev \
    --from-stage=fast-feedback \
    --to-env=gcp-dev \
    --to-stage=extended
```
