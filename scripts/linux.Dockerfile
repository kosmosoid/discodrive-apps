# Reproducible Linux build for the DiscoDrive Wails client.
#
# Wails v2 cannot cross-compile a Linux GUI from macOS (needs the WebKitGTK
# toolchain), and there is no official wailsapp/wails image — so we build inside
# Debian. Builds for the image's native architecture (linux/arm64 on Apple
# Silicon, linux/amd64 on Intel/CI). Pass `--platform=linux/amd64` to `docker
# build` to force amd64 (emulated, slower, on Apple Silicon).
#
# Debian bookworm ships only webkit2gtk-4.1 (4.0 was dropped), so the build uses
# Wails' `webkit2_41` tag.
#
# Build + export the binary to ./dist/linux:
#   docker build -f scripts/linux.Dockerfile -o type=local,dest=dist/linux .

FROM debian:bookworm-slim AS build

ARG GO_VERSION=1.25.0
ARG WAILS_VERSION=v2.12.0
ENV DEBIAN_FRONTEND=noninteractive
ENV PATH=/usr/local/go/bin:/root/go/bin:$PATH

RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates curl git build-essential pkg-config \
      libgtk-3-dev libwebkit2gtk-4.1-dev nodejs npm \
    && rm -rf /var/lib/apt/lists/*

# Go toolchain (architecture-aware: amd64 / arm64).
RUN ARCH="$(dpkg --print-architecture)" \
    && curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${ARCH}.tar.gz" \
       | tar -C /usr/local -xz

RUN go install github.com/wailsapp/wails/v2/cmd/wails@${WAILS_VERSION}

WORKDIR /src
COPY . .
WORKDIR /src/daemon/cmd/discodrive-wails
RUN wails build -tags webkit2_41 -clean

# Export stage: contains only the built binary so `-o type=local` writes a clean dir.
FROM scratch AS export
COPY --from=build /src/daemon/cmd/discodrive-wails/build/bin/ /
