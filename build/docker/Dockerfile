FROM golang:latest AS build
WORKDIR /src
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 go build -o /cdc-gateway ./cmd/cdc-gateway
FROM gcr.io/distroless/base-debian12
COPY --from=build /cdc-gateway /cdc-gateway
EXPOSE 8088 9108
ENTRYPOINT ["/cdc-gateway"]