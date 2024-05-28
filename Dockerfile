FROM golang:1.22-alpine AS builder

# Run docker build from root of clio.
RUN mkdir -p /go/src/github.com/clio/magna
COPY . /go/src/github.com/openconfig/clio
WORKDIR /go/src/github.com/openconfig/clio
RUN GOOS=linux go build -C cmd -o clio

# Run second stage for the container that we actually run.
FROM alpine:latest
RUN mkdir /app
COPY --from=builder go/src/github.com/openconfig/clio/cmd/ /app

# Copy config file and substitute environment variables (if any).
RUN apk update && apk add envsubst
COPY config/config.yaml /config_raw.yaml
RUN envsubst < /config_raw.yaml > /config.yaml

EXPOSE 4317
EXPOSE 6030

CMD ["/app/clio", "--config", "config.yaml"]