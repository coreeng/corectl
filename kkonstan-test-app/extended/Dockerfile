FROM golang:1.22-alpine3.18 AS builder
WORKDIR $GOPATH/src/go.k6.io/k6
RUN apk --no-cache add git
RUN go install go.k6.io/xk6/cmd/xk6@v0.11.0
RUN xk6 build --with github.com/coreeng/xk6-prometheus@v1.0.2 --output /tmp/k6

FROM prom/prometheus:v2.51.0 AS prometheus

FROM alpine:3.18

RUN apk add --no-cache ca-certificates && \
    adduser -D -u 12345 -g 12345 k6
COPY --from=builder /tmp/k6 /usr/bin/k6
COPY --from=prometheus /bin/promtool /usr/bin/promtool

USER 12345
WORKDIR /home/k6

ADD ./resources/load-testing/ /home/k6/load-testing

ENV K6_OUT=prometheus

ENTRYPOINT ["k6"]
