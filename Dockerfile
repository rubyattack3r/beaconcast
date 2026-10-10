# syntax=docker/dockerfile:1.7
ARG GO_VERSION=1.26.9
FROM golang:${GO_VERSION}-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
ARG VERSION=dev
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /out/beaconcast ./cmd/beaconcast

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/beaconcast /usr/local/bin/beaconcast
COPY LICENSE THIRD_PARTY_NOTICES.md /usr/share/doc/beaconcast/
COPY internal/fonts/OFL.txt /usr/share/doc/beaconcast/JetBrainsMono-OFL.txt
COPY licenses /usr/share/doc/beaconcast/licenses
WORKDIR /work
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/beaconcast"]
