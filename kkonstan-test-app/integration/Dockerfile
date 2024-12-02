FROM golang:1.22-alpine3.18
RUN apk add build-base
RUN go install github.com/cucumber/godog/cmd/godog@v0.12.6

ADD ./integration/ /opt/app
RUN ls /opt/app
WORKDIR /opt/app

CMD ["godog"]