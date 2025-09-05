# syntax=docker/dockerfile:1.7
FROM golang:1.24 AS build
WORKDIR /src
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$(go env GOOS) GOARCH=$(go env GOARCH) \
    go build -trimpath \
      -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
      -o /cdc-gateway ./cmd/cdc-gateway

FROM gcr.io/distroless/base-debian12
COPY --from=build /cdc-gateway /cdc-gateway
EXPOSE 8088 9108
ENTRYPOINT ["/cdc-gateway"]
