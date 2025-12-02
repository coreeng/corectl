FROM docker.io/golang:1.25.4-alpine3.22 AS build

# hadolint ignore=DL3018
RUN apk --no-cache add build-base && \
    go install github.com/cucumber/godog/cmd/godog@v0.15.0

COPY ./ /opt/app
WORKDIR /opt/app

CMD ["godog"]
