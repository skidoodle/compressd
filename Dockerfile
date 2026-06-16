FROM debian:trixie-slim AS builder

ENV DEBIAN_FRONTEND=noninteractive
ENV GO_VERSION=1.26.4
ARG VERSION=dev

RUN apt-get update && apt-get install -y \
    build-essential \
    pkg-config \
    curl \
    git \
    libvips-dev \
    libheif-dev \
    libheif-plugin-aomdec \
    libheif-plugin-aomenc \
    libheif-plugin-dav1d \
    libheif-plugin-libde265 \
    libheif-plugin-x265 \
    libwebp-dev \
    patchelf \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN curl -L "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" | tar -C /usr/local -xzf -
ENV PATH=$PATH:/usr/local/go/bin

WORKDIR /app

ENV VIPS_MODULE_PATH=/app/lib

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN export CGO_ENABLED=1 && \
    go build -v -ldflags="-s -w -X main.Version=${VERSION}" -o compressd .

RUN chmod +x bundle_linux.sh && \
    ./bundle_linux.sh compressd

RUN ls -R lib/ && \
    ldd "$(find lib -path '*/vips-heif.so' -print -quit)" && \
    if [ -d lib/libheif ]; then ls -R lib/libheif; fi && \
    ./compressd --version

FROM debian:trixie-slim

RUN apt-get update && apt-get install -y \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/compressd .
COPY --from=builder /app/lib ./lib

ENV LD_LIBRARY_PATH=/app/lib
ENV VIPS_MODULE_PATH=/app/lib
ENV LIBHEIF_PLUGIN_PATH=/app/lib/libheif
ENV VIPS_DISCARD_MMAP=1

ENTRYPOINT ["./compressd"]

CMD ["--help"]
