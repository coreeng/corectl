FROM golang:1.26.4

# hadolint ignore=DL3008
RUN apt-get update \
    && apt-get upgrade --yes \
    && apt-get install vim --yes --no-install-recommends \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /root/workspace

HEALTHCHECK NONE

CMD ["bash"]
