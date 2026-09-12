# syntax=docker/dockerfile:1
FROM golang:1.23-bookworm AS build
WORKDIR /src
COPY . .
ARG TARGETOS=linux
ARG TARGETARCH=arm64
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/agents ./cmd/server

FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/agents /agents
USER nonroot:nonroot
EXPOSE 8000
ENTRYPOINT ["/agents"]
