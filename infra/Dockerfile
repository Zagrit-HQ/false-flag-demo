# FalseFlag Go service image.
#
# Build with docker buildx bake (preferred) or:
#   docker build --build-arg SERVICE=falseflag-api -f infra/Dockerfile .
#
# SERVICE must be one of: falseflag-api, falseflag-proxy, falseflag-operator,
# falseflag-mcp, falseflag-loadgen — i.e. one of the cmd/* binaries.

ARG GO_VERSION=1.26

FROM golang:${GO_VERSION}-alpine AS build
ARG SERVICE
RUN apk add --no-cache git
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY . .
ARG VERSION=dev
ARG COMMIT=unknown
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux \
    go build \
      -ldflags="-s -w -X github.com/depot/falseflag/internal/buildinfo.Version=${VERSION} -X github.com/depot/falseflag/internal/buildinfo.Commit=${COMMIT}" \
      -o /out/service ./cmd/${SERVICE}

FROM gcr.io/distroless/static-debian12:nonroot
ARG SERVICE
LABEL org.opencontainers.image.source="https://github.com/depot/falseflag"
LABEL org.opencontainers.image.title="${SERVICE}"
COPY --from=build /out/service /usr/local/bin/service
USER nonroot
ENTRYPOINT ["/usr/local/bin/service"]
