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
COPY --from=builder go/src/github.com/openconfig/clio/certs /certs

# Copy config file and substitute environment variables (if any).
RUN apk update && apk add envsubst
RUN apk update && apk add socat
COPY config/config.yaml /config_raw.yaml

EXPOSE 4317
EXPOSE 60302

CMD ["/app/clio --config /config.yaml"]
