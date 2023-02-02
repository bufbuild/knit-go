# syntax=docker/dockerfile:1.4
FROM golang:1.20.3-bullseye AS build
WORKDIR /app
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 \
    GOPRIVATE=buf.build/gen/go \
    go build -ldflags="-s -w" -trimpath ./cmd/protoc-gen-knit-go

FROM scratch
COPY --from=build --link --chown=root:root /etc/passwd /etc/passwd
COPY --from=build --link /app/protoc-gen-knit-go /
USER nobody
LABEL org.opencontainers.image.source https://github.com/bufbuild/knit-go
ENTRYPOINT [ "/protoc-gen-knit-go" ]
