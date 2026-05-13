# Hort — minimal daemon image
#
# Stage 1: build the static Go binary.
# Stage 2: minimal runtime that runs the daemon as PID 1.
#
# The daemon listens on a Unix socket. Mount /run/hort as a shared volume
# between this container and any client (e.g. the Fachwerk engine) to
# expose the socket; mount /var/lib/hort/.hort for persistent vault state.
#
# Build:
#   docker build -t ghcr.io/s16e/hort:dev .
# Run:
#   docker run --rm \
#     -e HORT_SOCKET_PATH=/run/hort/daemon.sock \
#     -v hort-state:/var/lib/hort/.hort \
#     -v hort-socket:/run/hort \
#     ghcr.io/s16e/hort:dev

ARG GO_VERSION=1.22
ARG DEBIAN_VERSION=bookworm-20241016-slim

FROM golang:${GO_VERSION}-bookworm AS builder

WORKDIR /src

# Cache module downloads separately from source compilation.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=docker
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

# Static binary, stripped. Same flags GoReleaser uses for the CLI build.
ENV CGO_ENABLED=0
RUN go build \
      -trimpath \
      -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${BUILD_DATE}" \
      -o /out/hort \
      ./cmd/hort

# ---------- Runtime ----------
FROM debian:${DEBIAN_VERSION} AS runtime

RUN apt-get update -y \
 && apt-get install -y --no-install-recommends ca-certificates tini \
 && apt-get clean && rm -rf /var/lib/apt/lists/* \
 && useradd --create-home --home-dir /var/lib/hort --shell /usr/sbin/nologin --uid 1000 hort \
 && mkdir -p /var/lib/hort/.hort /run/hort \
 && chown -R hort:hort /var/lib/hort /run/hort

COPY --from=builder /out/hort /usr/local/bin/hort

USER hort
ENV HOME=/var/lib/hort \
    HORT_SOCKET_PATH=/run/hort/daemon.sock

VOLUME ["/var/lib/hort/.hort", "/run/hort"]

ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/hort"]
CMD ["daemon", "start"]
