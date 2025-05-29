FROM golang:1.23 AS builder

WORKDIR /go/src/github.com/medxops/trazr-gen

COPY . .

ARG BUILD_VERSION=0.0.0
ARG BUILD_DATE=1970-01-01T00:00:00Z
ARG COMMIT_ID=unknown

RUN CGO_ENABLED=0 go build -ldflags "-X main.version=${BUILD_VERSION} -X main.date=${BUILD_DATE} -X main.commit=${COMMIT_ID}" \
    -o /usr/bin/trazr-gen -v /go/src/github.com/medxops/trazr-gen/cmd/trazr-gen

FROM cgr.dev/chainguard/static:latest

COPY --from=builder /usr/bin/trazr-gen /

ENTRYPOINT ["/trazr-gen"]
