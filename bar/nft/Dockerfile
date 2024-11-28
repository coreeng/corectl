FROM golang:1.22-alpine3.18 AS builder
WORKDIR $GOPATH/src/go.k6.io/k6
RUN apk --no-cache add git
RUN go install go.k6.io/xk6/cmd/xk6@v0.11.0
RUN xk6 build --with github.com/coreeng/xk6-prometheus@v1.0.2 --output /tmp/k6

FROM alpine:3.18

RUN apk add --no-cache ca-certificates && \
    adduser -D -u 12345 -g 12345 k6
COPY --from=builder /tmp/k6 /usr/bin/k6

WORKDIR /home/k6

ADD ./resources/load-testing/ load-testing
RUN chmod -R +x load-testing && chown -R 12345:12345 load-testing

USER 12345
ENV K6_OUT=prometheus

ENTRYPOINT ["k6"]
